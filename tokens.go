package main

import (
	"time"
	"math/big"
	"github.com/ethereum/go-ethereum/common"
)

var (
	// These are the tokens that we are interested in. Add tokens here if you want to add them to the tracker.
	TOKENS = map[common.Address]*EthToken{
		addr("0x48c80F1f4D53D5951e5D5438B54Cba84f29F32a5"): NewToken("REP", "0x48c80F1f4D53D5951e5D5438B54Cba84f29F32a5"),
		addr("0xBB9bc244D798123fDe783fCc1C72d3Bb8C189413"): NewToken("TheDAO", "0xBB9bc244D798123fDe783fCc1C72d3Bb8C189413"),
		addr("0x888666CA69E0f178DED6D75b5726Cee99A87D698"): NewToken("ICONOMI", "0x888666CA69E0f178DED6D75b5726Cee99A87D698"),
		addr("0x57d90b64a1a57749b0f932f1a3395792e12e7055"): NewToken("Elcoin", "0x57d90b64a1a57749b0f932f1a3395792e12e7055"),
		addr("0x4DF812F6064def1e5e029f1ca858777CC98D2D81"): NewToken("Xaurum", "0x4DF812F6064def1e5e029f1ca858777CC98D2D81"),
	}
)

func addr(addr string) common.Address {
	return common.HexToAddress(addr)
}

func NewToken(name, address string) *EthToken {
	return &EthToken{
		Name: name,
		Address: addr(address),
	}
}

type EthToken struct {
	Name    string
	Address common.Address
}

type TokenTx struct {
	Token    *EthToken
	Time     time.Time
	Gas      *big.Int
	GasPrice *big.Int
	Block    uint64
}

func (tx *TokenTx) Fee() *big.Int {
	return (&big.Int{}).Mul(tx.Gas, tx.GasPrice)
}