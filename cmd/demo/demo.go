package main

import (
	"os"

	"github.com/najimmy/go-simplechain/cmd"
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/consensus"
	"github.com/najimmy/go-simplechain/core"
	"github.com/najimmy/go-simplechain/log"
	"github.com/najimmy/go-simplechain/net"
	"github.com/najimmy/go-simplechain/storage"
	"github.com/najimmy/go-simplechain/tests"
	"github.com/urfave/cli"
)

func run(c *cli.Context) {
	if c.String("config") == "" {
		log.CLog().Fatal("not found config")
		return
	}
	config := cmd.NewConfigFromFile(c.String("config"))

	log.Init("", log.InfoLevel, 0)
	//TODO change node private key
	privKey, err := net.HexStringToPrivkeyTo(config.NodePrivateKey)
	if err != nil {
	}

	node := net.NewNode(config.Port, privKey)
	node.Start(config.Seeds[0])
	node.SetSubscriberPool(net.NewSubsriberPool())

	sp := node.GetSubscriberPool()

	storage, _ := storage.NewLevelDBStorage(config.DBPath)
	// storage, _ := storage.NewMemoryStorage()
	voters := tests.MakeVoterAccountsFromConfig(config)

	dpos := consensus.NewDpos()
	bc := core.NewBlockChain(dpos, storage)
	bc.Setup(voters)
	// bc.MakeGenesisBlock(voters)
	// bc.PutBlockByCoinbase(bc.GenesisBlock)

	bc.SetNode(node)
	sp.Register(net.MSG_NEW_BLOCK, bc)
	sp.Register(net.MSG_MISSING_BLOCK, bc)
	sp.Start()
	bc.Start()

	dpos.Setup(bc, node, common.HexToAddress(config.MinerAddress))
	dpos.Start()
	// if config.Port == 9991 {
	// 	dpos.Start()
	// }
	select {}

}

func main() {
	app := cli.NewApp()

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Value: "",
			Usage: "config file path",
		},
	}

	app.Action = run
	err := app.Run(os.Args)
	if err != nil {
		log.CLog().Fatal(err)
	}
}
