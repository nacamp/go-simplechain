package main

import (
	"os"
	"time"

	"github.com/nacamp/go-simplechain/account"
	"github.com/nacamp/go-simplechain/crypto"
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
	wallet := account.NewWallet(config.KeystoreFile)
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
			err := wallet.TimedUnlock(common.HexToAddress(config.MinerAddress), config.MinerPassphrase, time.Duration(0))
			if err != nil {
				log.CLog().Fatal(err)
			}
			cs.Setup(common.HexToAddress(config.MinerAddress), wallet)
		}
		bc.Setup(cs, cmd.MakeVoterAccountsFromConfig(config))
		// bc.Start()
		bcs.Start()
		node.Start(config.Seeds[0])
		cs.Start()

		wallet := account.NewWallet(config.KeystoreFile)

		rpcServer := rpc.NewRpcServer(config.RpcAddress)
		rpcService := &rpc.RpcService{}
		rpcService.Setup(rpcServer, config, bc, wallet)
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
			err := wallet.TimedUnlock(common.HexToAddress(config.MinerAddress), config.MinerPassphrase, time.Duration(0))
			if err != nil {
				log.CLog().Fatal(err)
			}
			cs.Setup(common.HexToAddress(config.MinerAddress), wallet, 3)
		}
		bc.Setup(cs, cmd.MakeVoterAccountsFromConfig(config))
		// bc.Start()
		bcs.Start()
		node.Start(config.Seeds[0])
		cs.Start()

		wallet := account.NewWallet(config.KeystoreFile)

		rpcServer := rpc.NewRpcServer(config.RpcAddress)
		rpcService := &rpc.RpcService{}
		rpcService.Setup(rpcServer, config, bc, wallet)
		rpcServer.Start()
	}

	select {}

}

func AccountImportAction(c *cli.Context) {
	if c.String("config") == "" {
		log.CLog().Fatal("not found config")
		return
	}
	config := cmd.NewConfigFromFile(c.String("config"))
	if len(c.Args()) < 2 {
		log.CLog().Fatal("need privatekey passphrase")
	}
	accountImportAction(config.KeystoreFile, c.Args()[0], c.Args()[1])
}

func accountImportAction(path string, priv string, passphrase string) {
	//TODO: priv validate
	key := new(account.Key)
	key.PrivateKey = crypto.ByteToPrivateKey(common.FromHex(priv))
	key.Address = crypto.CreateAddressFromPrivateKey(key.PrivateKey)
	//fmt.Printf("%v\n", common.Address2Hex(key.Address))
	wallet := account.NewWallet(path)
	wallet.StoreKey(key, passphrase)
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
	//TODO: read passphrase securely 
	app.Commands = []cli.Command{
		{
			Name:  "account",
			Usage: "account import|new ...",
			Subcommands: []cli.Command{
				{
					Name:        "import",
					Flags:       app.Flags,
					Usage:       "import privatekey passphrase",
					ArgsUsage:   "<privatekey>",
					Action:      AccountImportAction,
					Category:    "ACCOUNT COMMANDS",
					Description: `demo account import privatekey passphrase`,
				},
			},
		},
	}
	app.Action = run
	err := app.Run(os.Args)
	if err != nil {
		log.CLog().Fatal(err)
	}
}
