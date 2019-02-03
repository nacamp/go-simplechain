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
	streamPool    *PeerStreamPool
	discovery     *Discovery
}

// p2pStreamMap: new(sync.Map),
//TODO: 127.0.0.1 from parameter
func NewNode(port int, privKey crypto.PrivKey) *Node {
	maddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port))
	_node := &Node{maddr: maddr, privKey: privKey, done: make(chan bool, 1)}
	_node.streamPool = NewPeerStreamPool()
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
	node.discovery = NewDiscovery(host.ID(), node.maddr, peerstore.NewMetrics(), host.Peerstore(), node.streamPool, node)

	node.streamPool.AddHandler(node.discovery)

	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", host.ID().Pretty()))
	addr := host.Addrs()[0]
	fullAddr := addr.Encapsulate(hostAddr).String()
	log.CLog().WithFields(logrus.Fields{
		"fullAddr": fullAddr,
	}).Info("My address")
	node.host = host

	// // node.nodeRoute = NewNodeRoute(node)
	if seed != "" {
		addr, err := ma.NewMultiaddr(seed)
		if err != nil {
			log.CLog().WithFields(logrus.Fields{
				"Msg": err,
			}).Panic("seed")
		}
		info, err := peerstore.InfoFromP2pAddr(addr)
		if err != nil {
			log.CLog().WithFields(logrus.Fields{
				"Address": addr,
				"Msg":     err,
			}).Panic("seed")
		}
		_, err = node.Connect(info.ID, info.Addrs[0])
		node.discovery.Update(info)
		if err != nil {
			log.CLog().WithFields(logrus.Fields{
				"Msg": err,
			}).Panic("seed connect")
		}
		// ps.SendHello(node.maddr)
		// if ok := <-ps.HandshakeSucceedCh; !ok {
		// 	log.CLog().WithFields(logrus.Fields{
		// 		"Msg": err,
		// 	}).Panic("seed sendhello")
		// }
		//"/ip4/127.0.0.1/tcp/9991/ipfs/16Uiu2HAm7qHFiJPzG6bkKGtRuF9eaPSbp79xTdFKU3MwFmTMuGN7"
		//loopup
		//node.nodeRoute.AddNodeFromSeedString(seed)
		//go node.discovery.randomLookup()
		// go node.discovery.Start()
	}
	go node.discovery.Start()

	// go node.nodeRoute.Start()
	node.host.SetStreamHandler("/simplechain/0.0.1", node.HandleStream)

	// node.subsriberPool.Start()

	// node.nodeRoute = NewNodeRoute(node)
	// if seed != "" {
	// 	node.nodeRoute.AddNodeFromSeedString(seed)
	// }
	// go node.nodeRoute.Start()
	// node.host.SetStreamHandler("/simplechain/0.0.1", node.HandleStream)
	// node.subsriberPool.Start()
}

/*
	cn.peerStream.SendHello(cn.maddr)
	if ok := <-cn.peerStream.HandshakeSucceedCh; !ok {
		fmt.Println("false")
	}
	fmt.Println("true")


	handler2 := make(chan interface{}, 1)
	cn.peerStream.Register(MsgNearestPeersAck, handler2)
	go func() {
		msg := <-handler2
		fmt.Println(msg)
		msg2 := msg.(*Message)
		fmt.Println(msg2.Code)
		v, ok := cn.peerStream.replys.Load(msg2.Code)
		if ok {
			reply := v.(chan interface{})
			reply <- "hi"
		}
	}()

	handler := make(chan interface{}, 1)
	sn.peerStream.Register(MsgNearestPeers, handler)
	go func() {
		msg := <-handler
		fmt.Println(msg)
		msg2 := msg.(*Message)
		fmt.Println(msg2.Code)
		msg3, _ := NewRLPMessage(MsgNearestPeersAck, id)
		sn.peerStream.SendMessage(&msg3)
	}()

	// //Handshake is completed
	// //RequestNearestPeers(id peer.ID)
	msg, _ = NewRLPMessage(MsgNearestPeers, id)
	reply := make(chan interface{}, 1)
	assert.NoError(t, cn.peerStream.SendMessageReply(&msg, reply))
*/

func (node *Node) HandleStream(s libnet.Stream) {
	log.CLog().WithFields(logrus.Fields{
		"RemotePeer": s.Conn().RemotePeer().Pretty(),
	}).Debug("new stream")

	// p2pStream, err := NewP2PStreamWithStream(node, s)
	// // node.p2pStreamMap.Store(p2pStream.peerID, p2pStream)
	// if err != nil {
	// 	log.CLog().Warning(err)
	// }

	// p2pStream.Start(true)

	peerStream, err := NewPeerStream(s)
	if err != nil {
		log.CLog().Warning(err)
	}
	node.streamPool.AddStream(peerStream)
	peerStream.Start()
}

func (node *Node) SendMessage(message *Message, peerID peer.ID) {
	// value, ok := node.p2pStreamMap.Load(peerID)
	// if ok {
	// 	p2pStream := value.(*P2PStream)
	// 	p2pStream.sendMessage(message)
	// }
}

//TODO: Random, current send message at first node
func (node *Node) SendMessageToRandomNode(message *Message) {
	// node.p2pStreamMap.Range(func(key, value interface{}) bool {
	// 	p2pStream := value.(*P2PStream)
	// 	p2pStream.sendMessage(message)
	// 	return false
	// })
}

func (node *Node) BroadcastMessage(message *Message) {
	// node.p2pStreamMap.Range(func(key, value interface{}) bool {
	// 	p2pStream := value.(*P2PStream)
	// 	p2pStream.sendMessage(message)
	// 	return true
	// })
}

func (node *Node) Connect(id peer.ID, addr ma.Multiaddr) (*PeerStream, error) {
	// if peerStream, err := node.streamPool.GetStream(id); err == nil {
	// 	return peerStream, nil
	// }
	peerStream, err := node.streamPool.GetStream(id)
	if err == nil && peerStream.status != statusClosed {
		return peerStream, nil
	}

	node.host.Peerstore().AddAddr(id, addr, pstore.PermanentAddrTTL)
	s, err := node.host.NewStream(context.Background(), id, "/simplechain/0.0.1")
	if err != nil {
		return nil, err
	}
	peerStream, err = NewPeerStream(s)
	node.streamPool.AddStream(peerStream)
	peerStream.Start()
	// info := peerstore.PeerInfo{ID: id, Addrs: []multiaddr.Multiaddr{addr}}
	// node.discovery.Update(&info)
	return peerStream, nil
}
