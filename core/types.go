package core

import (
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/net"
	"github.com/najimmy/go-simplechain/storage"
)

type MinerState interface {
	Clone() (MinerState, error)
	RootHash() (hash common.Hash)
	Put([]common.Address, common.Hash) (hash common.Hash)
	GetMinerGroup(*BlockChain, *Block) (minerGroup []common.Address, voterBlock *Block, err error)
	MakeMiner(*AccountState, int) ([]common.Address, error)
}

type Consensus interface {
	NewMinerState(rootHash common.Hash, storage storage.Storage) (MinerState, error)
	UpdateLIB()
	ConsensusType() string
	InitSaveSnapshot(hash common.Hash, addresses []common.Address)
	GetMiners(hash common.Hash) ([]common.Address, error)
	SaveMiners(hash common.Hash, block *Block) error
	VerifyMinerTurn(block *Block) error
	SetupNonMiner(bc *BlockChain, node *net.Node)
}

/*
period


*/
