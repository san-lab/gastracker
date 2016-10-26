package main

import (
	"time"
	"sync"
	"log"
	"math/big"
	"github.com/ethereum/go-ethereum/common"
)

var (
	ETHER_TRANSFER_GAS = big.NewInt(21000)
)

type GasTracker struct {
	tokens *TokenNotifier
	prices *PriceTracker
	influx *Influx
	// process flow
	quit   chan struct{}
	wg     sync.WaitGroup
}

// Start the main gastracker module.
// It will start submodules TokenNotifier, PriceTracker and InfluxDB
func StartGasTracker(rpcEndpoint string) (*GasTracker, error) {
	// price tracker
	pt := StartPriceTracker(time.Minute)
	// influxDB
	influx, err := StartInflux()
	if err != nil {
		return nil, err
	}
	lastBlock, err := influx.GetLatestPointBlock()
	log.Printf("Latest block in DB: %v\n", lastBlock)
	if err != nil || lastBlock <= 0 {
		if err != nil {
			log.Printf("ERROR retrieving last block number: %s", err)
		}
		lastBlock = 2500000 // approx a month ago
	}
	// token notifier
	tn, err := StartTokenNotifier(rpcEndpoint, lastBlock)
	if err != nil {
		return nil, err
	}
	// done
	gt := &GasTracker{
		tokens: tn,
		prices: pt,
		influx: influx,
		quit: make(chan struct{}),
	}
	go gt.run()
	return gt, nil
}

func (gt *GasTracker) run() {
	gt.wg.Add(1)
	defer gt.wg.Done()
	// subscribe to token txs
	for {
		select {
		case txs := <-gt.tokens.TokenStream:
			gt.handleTxs(txs)
		case <-gt.quit:
			log.Println("Gas Tracker is closing...")
			return
		}
	}
}

func (gt *GasTracker) handleTxs(txs []*TokenTx) {
	var points []*TxPoint
	for _, tx := range txs {
		log.Printf("New tx for token %s\n", tx.Token.Name)
		etherWei := (&big.Float{}).SetInt(common.Ether)
		// token stats
		tokenWeiFee := (&big.Float{}).SetInt(tx.Fee())
		tokenEtherFee, _ := tokenWeiFee.Quo(tokenWeiFee, etherWei).Float64()
		// separately track gas price expressed in fee for ether transfer
		etherWeiFee := (&big.Float{}).SetInt((&big.Int{}).Mul(ETHER_TRANSFER_GAS, tx.GasPrice))
		etherEtherFee, _ := etherWeiFee.Quo(etherWeiFee, etherWei).Float64()
		points = append(points,
			&TxPoint{
				Time: tx.Time,
				Token: tx.Token.Name,
				Gas: tx.Gas.Uint64(),
				FeeMap: map[string]float64{
					"ETH": tokenEtherFee,
					"USD": tokenEtherFee * gt.prices.Get("USD"),
					"EUR": tokenEtherFee * gt.prices.Get("EUR"),
				},
				Block: tx.Block,
			},
			&TxPoint{
				Time: tx.Time,
				Token: "ETH",
				Gas: ETHER_TRANSFER_GAS.Uint64(),
				FeeMap: map[string]float64{
					"ETH": etherEtherFee,
					"USD": etherEtherFee * gt.prices.Get("USD"),
					"EUR": etherEtherFee * gt.prices.Get("EUR"),
				},
				Block: tx.Block,
			},
		)
	}
	gt.influx.AddTxPoints(points)
}

func (gt *GasTracker) Stop() {
	gt.prices.Stop()
	gt.tokens.Stop()
	gt.influx.Stop()
	gt.quit <- struct{}{}
	gt.wg.Wait()
}
