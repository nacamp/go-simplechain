package core

import (
	"math/big"

	"github.com/najimmy/go-simplechain/common"
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
	UpdateLIB(bc *BlockChain)
}

type ConfigAccount struct {
	Address string   `json:"address"`
	Balance *big.Int `json:"balance"`
}
type Config struct {
	HostId          string          `json:"hostId"`
	MinerAddress    string          `json:"minerAddress"`
	MinerPrivateKey string          `json:"minerPrivateKey"`
	Seeds           []string        `json:"seeds"`
	Voters          []ConfigAccount `json:"voters"`
}
