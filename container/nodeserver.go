package container

import (
	"time"

	"github.com/nacamp/go-simplechain/account"
	"github.com/nacamp/go-simplechain/cmd"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/consensus"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/core/service"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/net"
	"github.com/nacamp/go-simplechain/rpc"
	"github.com/nacamp/go-simplechain/storage"
	"github.com/sirupsen/logrus"
)

type NodeServer struct {
	consensus  core.Consensus
	bc         *core.BlockChain
	rpcServer  *rpc.RpcServer
	wallet     *account.Wallet
	bcService  *service.BlockChainService
	db         storage.Storage
	streamPool *net.PeerStreamPool
}

func NewNodeServer(config *cmd.Config) *NodeServer {
	ns := NodeServer{}

	if config.DBPath == "" {
		ns.db, _ = storage.NewMemoryStorage()
	} else {
		ns.db, _ = storage.NewLevelDBStorage(config.DBPath)
	}
	ns.bc = core.NewBlockChain(ns.db)

	ns.wallet = account.NewWallet(config.KeystoreFile)
	// ns.wallet.Load()

	//FIXME
	if config.Consensus == "dpos" {
		ns.consensus = consensus.NewDpos(nil) //node
	} else {
		ns.consensus = consensus.NewPoa(nil, ns.db)
		if config.EnableMining {
			log.CLog().WithFields(logrus.Fields{
				"Address":   config.MinerAddress,
				"Consensus": config.Consensus,
			}).Info("Miner Info")
			err := ns.wallet.TimedUnlock(common.HexToAddress(config.MinerAddress), config.MinerPassphrase, time.Duration(0))
			if err != nil {
				log.CLog().Fatal(err)
			}
			//FIXME:
			//ns.consensus.Setup(common.HexToAddress(config.MinerAddress), wallet)
			ns.bc.Setup(ns.consensus, cmd.MakeVoterAccountsFromConfig(config))
		}
	}

	ns.bcService = service.NewBlockChainService(ns.bc, ns.streamPool)
	ns.streamPool.AddHandler(ns.bcService)

	ns.rpcServer = rpc.NewRpcServer(config.RpcAddress)
	rpcService := &rpc.RpcService{}
	rpcService.Setup(ns.rpcServer, config, ns.bc, ns.wallet)

	return nil
}

func (n *NodeServer) Start() error {
	return nil
}