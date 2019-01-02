package cmd

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/core"
)

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

func MakeVoterAccountsFromConfig(config *Config) (voters []*core.Account) {
	voters = make([]*core.Account, 3)
	for i, voter := range config.Voters {
		account := &core.Account{}
		copy(account.Address[:], common.FromHex(voter.Address))
		account.Balance = voter.Balance
		voters[i] = account
	}
	return voters
}

func NewConfigFromFile(file string) (config *Config) {
	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		fmt.Println(err.Error())
	}
	jsonParser := json.NewDecoder(configFile)
	if err != nil {
		fmt.Println(err.Error())
	}
	config = &Config{}
	err = jsonParser.Decode(config)
	if err != nil {
		fmt.Println(err.Error())
	}
	return config
}
