package main

import (
	"flag"

	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/consensus"
	"github.com/najimmy/go-simplechain/core"
	"github.com/najimmy/go-simplechain/log"
	"github.com/najimmy/go-simplechain/net"
	"github.com/sirupsen/logrus"
	// log "github.com/sirupsen/logrus"
)

func main() {
	log.Init("", log.InfoLevel, 0)
	privHexString := flag.String("pv", "08021220a178bc3f8ee6738af0139d9784519e5aa1cb256c12c54444bd63296502f29e94", "privatekey")
	seed := flag.String("s", "", "ipfs")
	port := flag.Int("p", 9990, "port")
	flag.Parse()
	log.CLog().WithFields(logrus.Fields{
		"seed": *seed,
		"port": *port,
	}).Info("flags ")
	privKey, err := net.HexStringToPrivkeyTo(*privHexString)
	if err != nil {
	}

	node := net.NewNode(*port, privKey)
	node.Start(*seed)
	node.SetSubscriberPool(net.NewSubsriberPool())

	sp := node.GetSubscriberPool()
	dpos := consensus.NewDpos()
	bc, _ := core.NewBlockChain(dpos)
	sp.Register(net.MSG_NEW_BLOCK, bc)
	sp.Start()

	if *port == 9990 {
		dpos.Setup(bc, node, common.HexToAddress("0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0"))
		dpos.Start()
	} else if *port == 9991 {
		dpos.Setup(bc, node, common.HexToAddress("0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3"))
	} else {
		dpos.Setup(bc, node, common.HexToAddress("0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1"))
	}
	select {}
}

/*
./demo  #seed node
./demo  -pv 080212201afa45f64cd5a28cd40e178889ed2e9f987658bc4d48d376ef6ecb1ab1b26211 -p 9991   -s /ip4/127.0.0.1/tcp/9990/ipfs/16Uiu2HAkwR1pV8ZR8ApcZWrMSw5iNMwaJHFpKr91H9a1a65WGehk
./demo  -pv 08021220114a228984dea82c6d7e6996a85fccc7ae6053249dbf5aa5698ffb14668d68f4 -p 9992   -s /ip4/127.0.0.1/tcp/9990/ipfs/16Uiu2HAkwR1pV8ZR8ApcZWrMSw5iNMwaJHFpKr91H9a1a65WGehk
*/
