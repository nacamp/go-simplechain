package net

import (
	"context"
	"fmt"

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
	seedID     peer.ID
	done       chan bool
	privKey    crypto.PrivKey
	maddr      ma.Multiaddr
	host       host.Host
	streamPool *PeerStreamPool
	discovery  *Discovery
}

func NewNode(port int, privKey crypto.PrivKey, streamPool *PeerStreamPool) *Node {
	maddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))
	_node := &Node{
		maddr:      maddr,
		privKey:    privKey,
		done:       make(chan bool, 1),
		streamPool: streamPool,
	}
	return _node
}

func (node *Node) Setup() {
}


func (node *Node) Start(seed string) {
	host, _ := libp2p.New(
		context.Background(),
		libp2p.ListenAddrs(node.maddr),
		libp2p.Identity(node.privKey),
	)
	node.discovery = NewDiscovery(host.ID(), node.maddr, peerstore.NewMetrics(), host.Peerstore(), node.streamPool, node)
	node.streamPool.AddHandler(node.discovery)

	log.CLog().WithFields(logrus.Fields{
		"fullAddr": fmt.Sprintf("/ipfs/%s", host.ID().Pretty()),
	}).Info("My address")
	node.host = host

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
		_, err = node.Connect(info.ID, AddrFromPeerInfo(info))
		node.discovery.Update(info)
		if err != nil {
			log.CLog().WithFields(logrus.Fields{
				"Msg": err,
			}).Panic("seed connect")
		}
	}
	go node.discovery.Start()
	node.host.SetStreamHandler("/simplechain/0.0.1", node.HandleStream)
}

func (node *Node) HandleStream(s libnet.Stream) {
	log.CLog().WithFields(logrus.Fields{
		"RemotePeer": s.Conn().RemotePeer().Pretty(),
	}).Debug("new stream")

	peerStream, err := NewPeerStream(s)
	log.CLog().WithFields(logrus.Fields{
		"ID": peerStream.stream.Conn().RemotePeer(),
	}).Warning("inbound")
	if err != nil {
		log.CLog().Warning(err)
	}
	node.streamPool.AddStream(peerStream)
	peerStream.Start()
}

func (node *Node) Connect(id peer.ID, addr ma.Multiaddr) (*PeerStream, error) {
	peerStream, err := node.streamPool.GetStream(id)
	if err == nil && peerStream.status != statusClosed {
		return peerStream, nil
	}
	log.CLog().WithFields(logrus.Fields{
		"ID": id,
	}).Warning("outbound")
	// Always firt add id and addr at Peerstore
	node.host.Peerstore().AddAddr(id, addr, pstore.PermanentAddrTTL)
	s, err := node.host.NewStream(context.Background(), id, "/simplechain/0.0.1")
	if err != nil {
		return nil, err
	}
	peerStream, err = NewPeerStream(s)
	node.streamPool.AddStream(peerStream)
	peerStream.Start()
	return peerStream, nil
}
