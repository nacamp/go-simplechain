package core

import (
	"github.com/nacamp/go-simplechain/common"
)

type MinerState interface {
	Clone() (MinerState, error)
	RootHash() (hash common.Hash)
	Put([]common.Address, common.Hash) (hash common.Hash)
	GetMinerGroup(*BlockChain, *Block) (minerGroup []common.Address, voterBlock *Block, err error)
	MakeMiner(*AccountState, int) ([]common.Address, error)
}

type ConsensusState interface {
	RootHash() (hash common.Hash)
	ExecuteTransaction()
}

type Consensus interface {
	UpdateLIB()
	ConsensusType() string
	SaveMiners(block *Block) error
	LoadConsensusStatus(block *Block) (err error)
	VerifyConsensusStatusHash(block *Block) (err error)
	MakeGenesisBlock(block *Block, voters []*Account) error
	AddBlockChain(*BlockChain)
	CloneFromParentBlock(block *Block, parentBlock *Block) (err error)
	Start()
	Verify(block *Block) (err error)
	SaveState(block *Block) error
}
