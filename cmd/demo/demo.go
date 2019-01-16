package main

import (
	"os"

	"github.com/nacamp/go-simplechain/rpc"
	"github.com/sirupsen/logrus"

	"github.com/nacamp/go-simplechain/cmd"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/consensus"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/net"
	"github.com/nacamp/go-simplechain/storage"
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
	privKey, err := config.NodePrivateKey()
	if err != nil {
	}

	node := net.NewNode(config.Port, privKey)
	node.Setup()

	var db storage.Storage
	if config.DBPath == "" {
		db, _ = storage.NewMemoryStorage()
	} else {
		db, _ = storage.NewLevelDBStorage(config.DBPath)
	}
	//TODO: remove duplicated code
	if config.Consensus == "dpos" {
		cs := consensus.NewDpos(node)
		bc := core.NewBlockChain(db)
		bcs := core.NewBlockChainService(bc, node)
		if config.EnableMining {
			log.CLog().WithFields(logrus.Fields{
				"Address":   config.MinerAddress,
				"Consensus": config.Consensus,
			}).Info("Miner Info")
			cs.Setup(common.HexToAddress(config.MinerAddress), common.FromHex(config.MinerPrivateKey))
		}
		bc.Setup(cs, cmd.MakeVoterAccountsFromConfig(config))
		// bc.Start()
		bcs.Start()
		node.Start(config.Seeds[0])
		cs.Start()

		rpcServer := rpc.NewRpcServer(config.RpcAddress)
		rpcService := &rpc.RpcService{}
		rpcService.Setup(rpcServer, config, bc)
		rpcServer.Start()
	} else {
		cs := consensus.NewPoa(node, db)
		bc := core.NewBlockChain(db)
		bcs := core.NewBlockChainService(bc, node)
		// bc.SetNode(node)
		if config.EnableMining {
			log.CLog().WithFields(logrus.Fields{
				"Address":   config.MinerAddress,
				"Consensus": config.Consensus,
			}).Info("Miner Info")
			cs.Setup(common.HexToAddress(config.MinerAddress), common.FromHex(config.MinerPrivateKey), 3)
		}
		bc.Setup(cs, cmd.MakeVoterAccountsFromConfig(config))
		// bc.Start()
		bcs.Start()
		node.Start(config.Seeds[0])
		cs.Start()

		rpcServer := rpc.NewRpcServer(config.RpcAddress)
		rpcService := &rpc.RpcService{}
		rpcService.Setup(rpcServer, config, bc)
		rpcServer.Start()
	}

	select {}

}

func main() {
	app := cli.NewApp()

	//TODO: input passphrase, to decrypt  encryped privatekey
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
