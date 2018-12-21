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
	ConsensusType() string
	ExecuteVote(hash common.Hash, tx *Transaction)
	InitSaveSnapshot(hash common.Hash, addresses []common.Address)
	GetMiners(hash common.Hash) ([]common.Address, error)
	SaveMiners(block *Block) error
	VerifyMinerTurn(block *Block) error
}

type ConfigAccount struct {
	Address string   `json:"address"`
	Balance *big.Int `json:"balance"`
}
type Config struct {
	HostId          string          `json:"host_id"`
	RpcAddress      string          `json:"rpc_address"`
	DBPath          string          `json:"db_path"`
	MinerAddress    string          `json:"miner_address"`
	MinerPrivateKey string          `json:"miner_private_key"`
	NodePrivateKey  string          `json:"node_private_key"`
	Port            int             `json:"port"`
	Seeds           []string        `json:"seeds"`
	Voters          []ConfigAccount `json:"voters"`
	EnableMining    bool            `json:"enable_mining"`
	Consensus       string          `json:"consensus"`
}

/*
period


*/
