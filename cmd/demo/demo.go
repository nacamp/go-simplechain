package main

import (
	"os"

	"github.com/najimmy/go-simplechain/rpc"
	"github.com/sirupsen/logrus"

	"github.com/najimmy/go-simplechain/cmd"
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/consensus"
	"github.com/najimmy/go-simplechain/core"
	"github.com/najimmy/go-simplechain/log"
	"github.com/najimmy/go-simplechain/net"
	"github.com/najimmy/go-simplechain/storage"
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
	node.Setup()

	var db storage.Storage
	if config.DBPath == "" {
		db, _ = storage.NewMemoryStorage()
	} else {
		db, _ = storage.NewLevelDBStorage(config.DBPath)
	}
	//TODO: remove duplicated code
	if config.Consensus == "dpos" {
		cs := consensus.NewDpos()
		bc := core.NewBlockChain(cs, db)
		bc.Setup(cmd.MakeVoterAccountsFromConfig(config))
		bc.SetNode(node)
		if config.EnableMining {
			log.CLog().WithFields(logrus.Fields{
				"Address": config.MinerAddress,
			}).Info("Miner address")
			cs.Setup(bc, node, common.HexToAddress(config.MinerAddress), common.FromHex(config.MinerPrivateKey))
		} else {
			cs.SetupNonMiner(bc, node)
		}
		bc.Start()
		node.Start(config.Seeds[0])
		cs.Start()

		rpcServer := rpc.NewRpcServer(config.RpcAddress)
		rpcService := &rpc.RpcService{}
		rpcService.Setup(rpcServer, config, bc)
		rpcServer.Start()
	} else {
		cs := consensus.NewPoa(db)
		bc := core.NewBlockChain(cs, db)
		bc.Setup(cmd.MakeVoterAccountsFromConfig(config))
		bc.SetNode(node)
		if config.EnableMining {
			log.CLog().WithFields(logrus.Fields{
				"Address": config.MinerAddress,
			}).Info("Miner address")
			cs.Setup(bc, node, common.HexToAddress(config.MinerAddress), common.FromHex(config.MinerPrivateKey), 3)
		} else {
			cs.SetupNonMiner(bc, node)
		}
		bc.Start()
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
