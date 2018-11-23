package cmd_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"testing"

	"github.com/najimmy/go-simplechain/cmd"
	"github.com/najimmy/go-simplechain/core"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	configStr := `{
		"host_id" : "/ip4/127.0.0.1/tcp/9990/ipfs/16Uiu2HAkwR1pV8ZR8ApcZWrMSw5iNMwaJHFpKr91H9a1a65WGehk",
		"db_path" : "/opt/simplechain/data",
		"miner_address" : "0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0",
		"miner_private_key" : "0xe68fb0a479c495910c8351c3593667028b45d679f55ce22b0514c4a8a6bcbdd1",
		"seeds" :  ["080212201afa45f64cd5a28cd40e178889ed2e9f987658bc4d48d376ef6ecb1ab1b26211"],
		"voters" : [{"address":"0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0", "balance":100 },
				    {"address":"0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3", "balance":20 },
					{"address":"0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1", "balance":50 }]
		}`
	contents := []byte(configStr)
	config := &core.Config{}
	// err := json.Unmarshal([]byte(contents), config)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Println(config.HostId)
	// fmt.Println(config.MinerAddress)
	// fmt.Println(config.MinerPrivateKey)
	// fmt.Println(config.Seeds)
	// fmt.Println(config.Voters[0].Address, config.Voters[0].Balance)
	// fmt.Println(config.Voters[1].Address, config.Voters[1].Balance)
	// assert.Equal(t, 1, 1, "")

	//tempfile
	tmpfile, err := ioutil.TempFile("", "test_")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write(contents); err != nil {
		fmt.Println(err)
	}
	if err := tmpfile.Close(); err != nil {
		fmt.Println(err)
	}

	config = &core.Config{}
	config = cmd.NewConfigFromFile(tmpfile.Name())
	// fmt.Println(config.HostId)
	// fmt.Println(config.MinerAddress)
	// fmt.Println(config.MinerPrivateKey)
	// fmt.Println(config.Seeds)
	// fmt.Println(config.Voters[0].Address, config.Voters[0].Balance)
	// fmt.Println(config.Voters[1].Address, config.Voters[1].Balance)
	assert.Equal(t, new(big.Int).SetUint64(100), config.Voters[0].Balance, "")
	assert.Equal(t, "0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3", config.Voters[1].Address, "")
	assert.Equal(t, "/opt/simplechain/data", config.DBPath, "")
}
