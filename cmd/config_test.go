package cmd_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/nacamp/go-simplechain/cmd"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	configStr := `{
		"host_id" : "/ip4/127.0.0.1/tcp/9990/ipfs/16Uiu2HAkwR1pV8ZR8ApcZWrMSw5iNMwaJHFpKr91H9a1a65WGehk",
		"db_path" : "/opt/simplechain/data",
		"miner_address" : "0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0",
		"miner_private_key" : "0xe68fb0a479c495910c8351c3593667028b45d679f55ce22b0514c4a8a6bcbdd1",
		"node_key_path" : "/test/nodekey",
		"seeds" :  ["080212201afa45f64cd5a28cd40e178889ed2e9f987658bc4d48d376ef6ecb1ab1b26211"],
		"voters" : [
			{"address":"0x1a8dd828a43acdcd9f1286ab437b91e43482bd5dd7a92a2631671554f5179b40d21e46a9", "balance":100 },
			{"address":"0xba2a519022ce61342363aac00240184abfe5cb76f7ba4d1c5e419e0703881788b2c75ed5", "balance":90 },
			{"address":"0xc6d40a9bf9fe9d90019511a2147dc0958657da97463ca59d2594d5536dcdfd30ed93707d", "balance":80 },
			{"address":"0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44", "balance":70 },
			{"address":"0xd725b51583b7db7e6732d87b6fa402ee30189fa57bdb514ce1f1928dc87b02af34cfb7df", "balance":60 },
			{"address":"0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2", "balance":50 }
		]
		}`
	contents := []byte(configStr)
	config := &cmd.Config{}

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

	config = &cmd.Config{}
	config = cmd.NewConfigFromFile(tmpfile.Name())
	assert.Equal(t, new(big.Int).SetUint64(100), config.Voters[0].Balance, "")
	assert.Equal(t, "0xba2a519022ce61342363aac00240184abfe5cb76f7ba4d1c5e419e0703881788b2c75ed5", config.Voters[1].Address, "")
	assert.Equal(t, "/opt/simplechain/data", config.DBPath, "")
	assert.Equal(t, "/test/nodekey", config.NodeKeyPath, "")

	//NodeKey
	priv1, _ := config.NodePrivateKey()
	priv2, _ := config.NodePrivateKey()
	assert.Equal(t, priv1, priv2, "")
	_, err = os.Stat(filepath.Join(config.NodeKeyPath, "node_pub.id"))
	assert.False(t, os.IsNotExist(err), "")

	assert.Equal(t, 6, len(cmd.MakeVoterAccountsFromConfig(config)))
}
