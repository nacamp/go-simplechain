package net

import (
	"bufio"
	"context"
	"fmt"
	"sync"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	libnet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
)

type Node struct {
	seedID       peer.ID
	done         chan bool
	privKey      crypto.PrivKey
	maddr        ma.Multiaddr
	host         host.Host
	nodeRoute    *NodeRoute
	p2pStreamMap *sync.Map
}

//TODO: 127.0.0.1 from parameter
func NewNode(port int, privKey crypto.PrivKey) *Node {
	maddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))
	_node := &Node{maddr: maddr, privKey: privKey, p2pStreamMap: new(sync.Map), done: make(chan bool, 1)}
	return _node
}

func (node *Node) Start(seed string) {
	host, _ := libp2p.New(
		context.Background(),
		libp2p.ListenAddrs(node.maddr),
		libp2p.Identity(node.privKey),
	)

	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", host.ID().Pretty()))
	addr := host.Addrs()[0]
	fullAddr := addr.Encapsulate(hostAddr).String()
	log.WithFields(log.Fields{
		"fullAddr": fullAddr,
	}).Info("I am ")

	log.WithFields(log.Fields{
		"maddr": node.maddr,
	}).Info("node.Start")
	node.host = host
	node.nodeRoute = NewNodeRoute(node)
	if seed != "" {
		node.nodeRoute.AddNodeFromSeedString(seed)
	}
	go node.nodeRoute.Start()

	log.WithFields(log.Fields{
		"host.id": host.ID(),
	}).Info("regisiter my hostid")

	node.host.SetStreamHandler("/simplechain/0.0.1", node.HandleStream)
}

func (node *Node) HandleStream(s libnet.Stream) {
	log.WithFields(log.Fields{
		"RemotePeer": s.Conn().RemotePeer().Pretty(),
	}).Info("Got a new stream!")

	p2pStream, err := NewP2PStreamWithStream(node, s)
	node.p2pStreamMap.Store(p2pStream.peerID, p2pStream)
	if err != nil {
		log.Fatal("HandleStream", err)
	}

	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	go p2pStream.readData(rw)
}
