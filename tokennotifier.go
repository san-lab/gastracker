package main

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"sync"
	"context"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/common"
	"log"
	"time"
	"github.com/ethereum/go-ethereum"
	"math/big"

	"github.com/hashicorp/golang-lru"
)

const (
	IDLE_TIME_BETWEEN_QUERIES = 2 * time.Second
	BLOCK_STEPS = 5000
)

// sha3("Transfer(address,address,uint256)") =
var TXTYPE_TRANSFER_TOPIC = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

type TokenNotifier struct {
	eth            *ethclient.Client
	TokenStream    chan []*TokenTx
	// internal data
	blockTimeCache *lru.Cache
	// process flow
	quit           chan struct{}
	wg             sync.WaitGroup
}

// Start a daemon that will keep updates with the Ethereum network and exposes a channel of relevant token-related
// transactions.
func StartTokenNotifier(rpcEndpoint string, lastBlock uint64) (*TokenNotifier, error) {
	eth, err := ethclient.Dial(rpcEndpoint)
	if (err != nil) {
		return nil, err
	}
	blockTimeCache, err := lru.New(25)
	if err != nil {
		return nil, err
	}
	tn := &TokenNotifier{
		eth: eth,
		TokenStream: make(chan []*TokenTx),
		blockTimeCache: blockTimeCache,
		quit: make(chan struct{}),
	}
	go tn.run(lastBlock)
	return tn, nil
}

func (tn *TokenNotifier) run(lastBlock uint64) {
	tn.wg.Add(1)
	defer tn.wg.Done()
	var addresses []common.Address
	for k := range TOKENS {
		addresses = append(addresses, k)
	}
	for {
		select {
		case <-tn.quit:
			log.Println("Token Notifier is closing...")
			return
		default:
		}
		blk, err := tn.eth.BlockByNumber(context.TODO(), nil)
		if err != nil {
			log.Printf("ERROR getting latest header: %s\n", err)
			log.Println("Retrying in a moment...")
			time.Sleep(IDLE_TIME_BETWEEN_QUERIES)
			continue
		}
		header := blk.Header()
		latestBlock := header.Number.Uint64()
		if latestBlock > lastBlock {
			// update incrementally instead of all blocks at once
			updateUntilBlock := min(latestBlock, lastBlock + BLOCK_STEPS + 1)
			// query logs that are interesting for us
			logs, err := tn.eth.FilterLogs(context.TODO(), ethereum.FilterQuery{
				FromBlock: new(big.Int).SetUint64(lastBlock + 1),
				ToBlock: new(big.Int).SetUint64(updateUntilBlock),
				Addresses: addresses,
				//TODO uncomment once ethereum/go-ethereum#3330 is fixed
				//Topics: [][]common.Hash{{TXTYPE_TRANSFER_TOPIC}},
			})
			if err != nil {
				log.Printf("ERROR retrieving logs: %s\n", err)
				log.Println("Retrying in a moment...")
				time.Sleep(IDLE_TIME_BETWEEN_QUERIES)
				continue
			}
			txs := tn.handleLogs(logs, latestBlock)
			tn.TokenStream <- txs
			lastBlock = updateUntilBlock
		} else {
			log.Println("No new blocks, sleeping...")
		}
		time.Sleep(IDLE_TIME_BETWEEN_QUERIES)
	}
}

func (tn *TokenNotifier) handleLogs(logs []vm.Log, latestBlock uint64) []*TokenTx {
	var txs []*TokenTx
	LogsLoop:
	for _, lg := range logs {
		//TODO remove once ethereum/go-ethereum#3330 is fixed
		// look for relevant topic
		relevant := false
		TopicsLoop:
		for _, t := range lg.Topics {
			if t == TXTYPE_TRANSFER_TOPIC {
				relevant = true
				break TopicsLoop
			}
		}
		if !relevant {
			continue LogsLoop
		}

		// retrieve the full tx
		tx, err := tn.eth.TransactionByHash(context.TODO(), lg.TxHash)
		if err != nil {
			log.Printf("ERROR retrieving transaction %s: %s\n", lg.TxHash.Hex(), err)
			continue LogsLoop
		}
		// parse transactions and save relevant ones
		if tx.To() == nil {
			// contract creation tx
			continue LogsLoop
		}
		if token, relevant := TOKENS[*tx.To()]; relevant {
			var blockTime time.Time
			if latestBlock - lg.BlockNumber < 2 {
				blockTime = time.Now()
			} else {
				blockTime, err = tn.getBlockTime(lg.BlockNumber)
				if err != nil {
					log.Printf("ERROR retrieving block time: %s\n", err)
					continue LogsLoop
				}
			}
			tokenTx := &TokenTx{
				Token: token,
				Time: blockTime,
				Gas: tx.Gas(),
				GasPrice: tx.GasPrice(),
				Block: lg.BlockNumber,
			}
			txs = append(txs, tokenTx)
		} else {
			log.Printf("Irrelevant transaction for address %s: %s\n", tx.To().Hex(), lg.TxHash.Hex())
		}
	}
	return txs
}

func (tn *TokenNotifier) getBlockTime(blockNumber uint64) (time.Time, error) {
	var blockTime time.Time
	bt, found := tn.blockTimeCache.Get(blockNumber)
	if !found {
		// retrieve block time and store it in cache
		header, err := tn.eth.HeaderByNumber(context.TODO(), new(big.Int).SetUint64(blockNumber))
		if err != nil {
			return time.Time{}, err
		}
		blockTime = time.Unix(header.Time.Int64(), 0)
		tn.blockTimeCache.Add(blockNumber, blockTime)
	} else {
		blockTime = bt.(time.Time)
	}
	return blockTime, nil
}

func (tn *TokenNotifier) Stop() {
	tn.quit <- struct{}{}
	tn.wg.Wait()
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}