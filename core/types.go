package core

import (
	"github.com/najimmy/go-simplechain/common"
)

type MinerState interface {
	Clone() (MinerState, error)
	RootHash() (hash common.Hash)
	Put([]common.Address, common.Hash) (hash common.Hash)
	GetMinerGroup(*BlockChain, *Block) (minerGroup []common.Address, voterBlock *Block, err error)
	MakeMiner(*AccountState, int) ([]common.Address, error)
}

type Consensus interface {
	UpdateLIB()
	ConsensusType() string
	GetMiners(hash common.Hash) ([]common.Address, error)
	SaveMiners(hash common.Hash, block *Block) error
	VerifyMinerTurn(block *Block) error

	LoadConsensusStatus(block *Block) (err error)
	MakeGenesisBlock(block *Block, voters []*Account) error
	AddBlockChain(*BlockChain)
	CloneFromParentBlock(block *Block, parentBlock *Block) (err error)
}
