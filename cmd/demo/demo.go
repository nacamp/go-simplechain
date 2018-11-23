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

	storage, _ := storage.NewMemoryStorage()
	voters := tests.MakeVoterAccountsFromConfig(config)

	dpos := consensus.NewDpos()
	bc := core.NewBlockChain(dpos, storage)
	bc.MakeGenesisBlock(voters)
	bc.PutBlockByCoinbase(bc.GenesisBlock)

	bc.SetNode(node)
	sp.Register(net.MSG_NEW_BLOCK, bc)
	sp.Register(net.MSG_MISSING_BLOCK, bc)
	sp.Start()
	bc.Start()

	dpos.Setup(bc, node, common.HexToAddress(config.MinerAddress))
	dpos.Start()

	// if config.Port == 9990 {
	// 	dpos.Setup(bc, node, common.HexToAddress(config.MinerAddress))
	// 	dpos.Start()
	// } else if config.Port == 9991 {
	// 	dpos.Setup(bc, node, common.HexToAddress("0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3"))
	// } else {
	// 	dpos.Setup(bc, node, common.HexToAddress("0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1"))
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
