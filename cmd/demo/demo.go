package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/nacamp/go-simplechain/account"
	"github.com/nacamp/go-simplechain/console"
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
	wallet.Load()
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
	// if len(c.Args()) < 1 {
	// 	log.CLog().Fatal("need privatekey passphrase")
	// }
	privateKey := getPrivateKey()
	passphrase := getPassPhrase("", true, 0, []string{})
	accountImportAction(config.KeystoreFile, privateKey, passphrase)
}

func accountImportAction(path string, priv string, passphrase string) {
	//TODO: priv validate
	key := new(account.Key)
	key.PrivateKey = crypto.ByteToPrivateKey(common.FromHex(priv))
	key.Address = crypto.CreateAddressFromPrivateKey(key.PrivateKey)
	fmt.Printf("address : %v\n", common.Address2Hex(key.Address))
	wallet := account.NewWallet(path)
	wallet.Load()
	wallet.StoreKey(key, passphrase)
}

func getPassPhrase(prompt string, confirmation bool, i int, passwords []string) string {
	// If a list of passwords was supplied, retrieve from them
	if len(passwords) > 0 {
		if i < len(passwords) {
			return passwords[i]
		}
		return passwords[len(passwords)-1]
	}
	// Otherwise prompt the user for the password
	if prompt != "" {
		fmt.Println(prompt)
	}
	password, err := console.Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		utils.Fatalf("Failed to read passphrase: %v", err)
	}
	if confirmation {
		confirm, err := console.Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			utils.Fatalf("Failed to read passphrase confirmation: %v", err)
		}
		if password != confirm {
			utils.Fatalf("Passphrases do not match")
		}
	}
	return password
}

func getPrivateKey() string {
	privateKey, err := console.Stdin.PromptPassword("PrivateKey: ")
	if err != nil {
		utils.Fatalf("Failed to read privateKey: %v", err)
	}
	return privateKey
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

	app.Commands = []cli.Command{
		{
			Name:  "account",
			Usage: "account import|new ...",
			Subcommands: []cli.Command{
				{
					Name:        "import",
					Flags:       app.Flags,
					Usage:       "import privatekey",
					ArgsUsage:   "<privatekey>",
					Action:      AccountImportAction,
					Category:    "ACCOUNT COMMANDS",
					Description: `demo account import`,
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
