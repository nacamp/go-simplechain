package main

import (
	"flag"
	"fmt"

	// "github.com/TheBaseblock/go-baseblock/pilot"

	"github.com/najimmy/go-simplechain/net"
	//ma "github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
)

func main() {
	privHexString := flag.String("pv", "08021220a178bc3f8ee6738af0139d9784519e5aa1cb256c12c54444bd63296502f29e94", "Privatekey String")
	seed := flag.String("s", "", "Seed MultiAddr String")
	port := flag.Int("p", 9990, "Source port number")
	flag.Parse()

	log.WithFields(log.Fields{
		"seed": *seed,
		"port": *port, //마지막에도 콤마가 필요
	}).Info("args")
	privKey, err := net.HexStringToPrivkeyTo(*privHexString)
	if err != nil {
	}
	node := net.NewNode(*port, privKey)
	node.Start(*seed)
	if *seed == "" {
		fmt.Println("I am seeder")
	}
	select {}
}

/*
cd /Users/jimmy/go/src/github.com/najimmy/go-simplechain
./go-simplechain  -pv 080212201afa45f64cd5a28cd40e178889ed2e9f987658bc4d48d376ef6ecb1ab1b26211 -p 9991   -s /ip4/127.0.0.1/tcp/9990/ipfs/16Uiu2HAkwR1pV8ZR8ApcZWrMSw5iNMwaJHFpKr91H9a1a65WGehk
./go-simplechain  -pv 08021220114a228984dea82c6d7e6996a85fccc7ae6053249dbf5aa5698ffb14668d68f4 -p 9992   -s /ip4/127.0.0.1/tcp/9990/ipfs/16Uiu2HAkwR1pV8ZR8ApcZWrMSw5iNMwaJHFpKr91H9a1a65WGehk


INFO[0000] args                                          port=9990 seed=
INFO[0000] I am                                          fullAddr=/ip4/127.0.0.1/tcp/9990/ipfs/16Uiu2HAkwR1pV8ZR8ApcZWrMSw5iNMwaJHFpKr91H9a1a65WGehk
INFO[0000] node.Start                                    maddr=/ip4/127.0.0.1/tcp/9990
INFO[0000] 내hostid 등록                                    host.id="<peer.ID 16*5WGehk>"
I am seeder

*/
