package core

import (
	"github.com/nacamp/go-simplechain/common"
)


type ConsensusState interface {
	RootHash() (hash common.Hash)
	ExecuteTransaction(block *Block, txIndex int, account *Account) (err error)
	Clone() (ConsensusState, error)
}

type Consensus interface {
	UpdateLIB()
	ConsensusType() string
	MakeGenesisBlock(block *Block, voters []*Account) error
	AddBlockChain(*BlockChain)
	Start()
	Verify(block *Block) (err error)
	SaveState(block *Block) error
	LoadState(block *Block) (state ConsensusState, err error)
}