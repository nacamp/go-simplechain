package core

import (
	"math/big"

	"github.com/najimmy/go-simplechain/common"
)

// Simple Header
type Header struct {
	ParentHash common.Hash
	Coinbase   common.Address
	Number     *big.Int
	Time       *big.Int
}

// Simple Block
type Block struct {
	Header *Header
}
