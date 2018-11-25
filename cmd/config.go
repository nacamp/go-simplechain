package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/core"
)

func MakeVoterAccountsFromConfig(config *core.Config) (voters []*core.Account) {
	voters = make([]*core.Account, 3)
	for i, voter := range config.Voters {
		account := &core.Account{}
		copy(account.Address[:], common.FromHex(voter.Address))
		account.Balance = voter.Balance
		voters[i] = account
	}
	return voters
}

func NewConfigFromFile(file string) (config *core.Config) {
	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		fmt.Println(err.Error())
	}
	jsonParser := json.NewDecoder(configFile)
	if err != nil {
		fmt.Println(err.Error())
	}
	config = &core.Config{}
	err = jsonParser.Decode(config)
	if err != nil {
		fmt.Println(err.Error())
	}
	return config
}
