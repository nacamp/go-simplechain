package net

import (
	"context"
	"fmt"
	"sync"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	libnet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/nacamp/go-simplechain/log"
	"github.com/sirupsen/logrus"
)

type Node struct {
	seedID        peer.ID
	done          chan bool
	privKey       crypto.PrivKey
	maddr         ma.Multiaddr
	host          host.Host
	nodeRoute     *NodeRoute
	p2pStreamMap  *sync.Map
	subsriberPool *SubscriberPool
	StreamPool    *PeerStreamPool
	discovery     *Discovery
}

//TODO: 127.0.0.1 from parameter
func NewNode(port int, privKey crypto.PrivKey) *Node {
	maddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))
	_node := &Node{maddr: maddr, privKey: privKey, p2pStreamMap: new(sync.Map), done: make(chan bool, 1)}
	return _node
}

func (node *Node) Setup() {
	node.subsriberPool = NewSubsriberPool()
}

func (node *Node) RegisterSubscriber(code uint64, subscriber Subscriber) {
	node.subsriberPool.Register(code, subscriber)
}

func (node *Node) Start(seed string) {
	host, _ := libp2p.New(
		context.Background(),
		libp2p.ListenAddrs(node.maddr),
		libp2p.Identity(node.privKey),
	)
	node.discovery = NewDiscovery(host.ID(), peerstore.NewMetrics(), host.Peerstore())
	node.discovery.node = node

	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", host.ID().Pretty()))
	addr := host.Addrs()[0]
	fullAddr := addr.Encapsulate(hostAddr).String()
	log.CLog().WithFields(logrus.Fields{
		"fullAddr": fullAddr,
	}).Info("My address")
	node.host = host
	node.nodeRoute = NewNodeRoute(node)
	if seed != "" {
		node.nodeRoute.AddNodeFromSeedString(seed)
	}
	go node.nodeRoute.Start()
	node.host.SetStreamHandler("/simplechain/0.0.1", node.HandleStream)
	node.subsriberPool.Start()
}

func (node *Node) HandleStream(s libnet.Stream) {
	log.CLog().WithFields(logrus.Fields{
		"RemotePeer": s.Conn().RemotePeer().Pretty(),
	}).Debug("Got a new stream!")

	p2pStream, err := NewP2PStreamWithStream(node, s)
	node.p2pStreamMap.Store(p2pStream.peerID, p2pStream)
	if err != nil {
		log.CLog().Warning(err)
	}

	p2pStream.Start(true)
}

func (node *Node) SendMessage(message *Message, peerID peer.ID) {
	value, ok := node.p2pStreamMap.Load(peerID)
	if ok {
		p2pStream := value.(*P2PStream)
		p2pStream.sendMessage(message)
	}
}

//TODO: Random, current send message at first node
func (node *Node) SendMessageToRandomNode(message *Message) {
	node.p2pStreamMap.Range(func(key, value interface{}) bool {
		p2pStream := value.(*P2PStream)
		p2pStream.sendMessage(message)
		return false
	})
}

func (node *Node) BroadcastMessage(message *Message) {
	node.p2pStreamMap.Range(func(key, value interface{}) bool {
		p2pStream := value.(*P2PStream)
		p2pStream.sendMessage(message)
		return true
	})
}

func (node *Node) Connect(id peer.ID, addr ma.Multiaddr) (*PeerStream, error) {
	if peerStream, err := node.StreamPool.GetStream(id); err == nil {
		return peerStream, nil
	}
	node.host.Peerstore().AddAddr(id, addr, pstore.PermanentAddrTTL)
	s, err := node.host.NewStream(context.Background(), id, "/simplechain/0.0.1")
	if err != nil {
		return nil, err
	}
	peerStream, err := NewPeerStream(s)
	node.StreamPool.AddStream(peerStream)

	return peerStream, nil
}
