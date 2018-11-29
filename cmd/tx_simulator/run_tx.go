package main

// import (
// 	"fmt"
// 	"math/big"
// 	"time"

// 	"github.com/najimmy/go-simplechain/log"
// 	"github.com/najimmy/go-simplechain/net"
// 	"github.com/najimmy/go-simplechain/tests"
// 	"github.com/sirupsen/logrus"
// )

// func main() {
// 	start()
// 	select {}
// }

import (
	"math/big"
	"os"
	"time"

	"github.com/najimmy/go-simplechain/cmd"
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/consensus"
	"github.com/najimmy/go-simplechain/core"
	"github.com/najimmy/go-simplechain/log"
	"github.com/najimmy/go-simplechain/net"
	"github.com/najimmy/go-simplechain/storage"
	"github.com/najimmy/go-simplechain/tests"
	"github.com/sirupsen/logrus"
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
	dpos := consensus.NewDpos()
	bc := core.NewBlockChain(dpos, db)
	bc.Setup(cmd.MakeVoterAccountsFromConfig(config))
	bc.SetNode(node)
	if config.EnableMining {
		dpos.Setup(bc, node, common.HexToAddress(config.MinerAddress), common.FromHex(config.MinerPrivateKey))
	} else {
		dpos.SetupNonMiner(bc, node)
	}
	bc.Start()
	node.Start(config.Seeds[0])
	dpos.Start()

	// add code for tx>>>>>>>>>
	time.Sleep(10 * time.Second)
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			send_tx(node)
		}
	}
	select {}
	// add code for tx<<<<<<<<

}

func send_tx(node *net.Node) {
	tx1 := tests.MakeTransaction(tests.Addr0, tests.Addr1, new(big.Int).SetUint64(2))
	tx2 := tests.MakeTransaction(tests.Addr1, tests.Addr2, new(big.Int).SetUint64(1))

	message, _ := net.NewRLPMessage(net.MSG_NEW_TX, tx1)
	node.BroadcastMessage(&message)
	log.CLog().WithFields(logrus.Fields{
		"Height": common.Hash2Hex(tx1.Hash),
	}).Info("Send Tx")

	message, _ = net.NewRLPMessage(net.MSG_NEW_TX, tx2)
	node.BroadcastMessage(&message)
	log.CLog().WithFields(logrus.Fields{
		"Height": common.Hash2Hex(tx1.Hash),
	}).Info("Send Tx")
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
