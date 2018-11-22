package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/najimmy/go-simplechain/core"
)

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
