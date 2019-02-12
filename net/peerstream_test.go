package net

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	libnet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

func TestPeerStream(t *testing.T) {
	//16Uiu2HAkwR1pV8ZR8ApcZWrMSw5iNMwaJHFpKr91H9a1a65WGehk
	sn := NewTestNode("08021220a178bc3f8ee6738af0139d9784519e5aa1cb256c12c54444bd63296502f29e94", "/ip4/127.0.0.1/tcp/9991")
	//16Uiu2HAkxKaG3PHSLfDhfZ7a8YzP6w6fKooBTY1gmfSXxGYbsNuN
	cn := NewTestNode("080212201afa45f64cd5a28cd40e178889ed2e9f987658bc4d48d376ef6ecb1ab1b26211", "/ip4/127.0.0.1/tcp/9992")
	sn.Start(true)
	cn.Start(false)
	addr, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/9991")
	id, _ := peer.IDB58Decode("16Uiu2HAkwR1pV8ZR8ApcZWrMSw5iNMwaJHFpKr91H9a1a65WGehk")
	cn.Connect(id, addr)
	//sn.done <- true

	//Handshake is not completed
	msg, _ := NewRLPMessage(MsgNearestPeers, id)
	assert.Error(t, cn.peerStream.SendMessage(&msg))

	// //Handshake is completed
	// //RequestNearestPeers(id peer.ID)
	// msg, _ = NewRLPMessage(MsgNearestPeers, id)
	// assert.NoError(t, cn.peerStream.SendMessage(&msg))

	cn.peerStream.SendHello(cn.maddr)
	if ok := <-cn.peerStream.HandshakeSucceedCh; !ok {
		fmt.Println("false")
	}
	fmt.Println("true")

	handler2 := make(chan interface{}, 1)
	cn.peerStream.Register(MsgNearestPeersAck, handler2)
	go func() {
		msg := <-handler2
		// fmt.Println(msg)
		msg2 := msg.(*Message)
		//fmt.Println(msg2.Code)
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
		// msg2 := msg.(*Message)
		// fmt.Println(msg2.Code)
		msg3, _ := NewRLPMessage(MsgNearestPeersAck, id)
		sn.peerStream.SendMessage(&msg3)
	}()

	// //Handshake is completed
	// //RequestNearestPeers(id peer.ID)
	msg, _ = NewRLPMessage(MsgNearestPeers, id)
	reply := make(chan interface{}, 1)
	assert.NoError(t, cn.peerStream.SendMessageReply(&msg, reply))
	assert.Equal(t, "hi", <-reply)
	assert.True(t, sn.peerStream.IsHandshakeSucceed())
	assert.False(t, sn.peerStream.IsClosed())
	cn.peerStream.Close()

	ticker := time.NewTicker(100 * time.Millisecond)
	for t := range ticker.C {
		fmt.Println("Tick at", t)
		if cn.peerStream.IsClosed() == true && sn.peerStream.IsClosed() == true {
			break
		}
	}
}

type TestNode struct {
	done       chan bool
	privKey    crypto.PrivKey
	maddr      ma.Multiaddr
	host       host.Host
	peerStream *PeerStream
}

func NewTestNode(privStr string, addr string) *TestNode {
	b, _ := hex.DecodeString(privStr)
	priv, _ := crypto.UnmarshalPrivateKey(b)
	maddr, _ := ma.NewMultiaddr(addr)
	_node := &TestNode{maddr: maddr, privKey: priv, done: make(chan bool, 1)}
	return _node
}

func (node *TestNode) Start(isServer bool) {
	host, _ := libp2p.New(
		context.Background(),
		libp2p.ListenAddrs(node.maddr),
		libp2p.Identity(node.privKey),
	)
	node.host = host
	if isServer {
		node.host.SetStreamHandler("/simplechain/0.0.1", node.HandleStream)
		// go func() {
		// 	<-node.done
		// }()
	}
	go func() {
		<-node.done
	}()

}

func (node *TestNode) HandleStream(s libnet.Stream) {
	fmt.Println("RemotePeer", s.Conn().RemotePeer().Pretty())
	fmt.Println("inbound")
	peerStream, _ := NewPeerStream(s)
	node.peerStream = peerStream
	peerStream.Start()
}

func (node *TestNode) Connect(peerid peer.ID, addr ma.Multiaddr) {
	node.host.Peerstore().AddAddr(peerid, addr, pstore.PermanentAddrTTL)
	s, err := node.host.NewStream(context.Background(), peerid, "/simplechain/0.0.1")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("outbound")
	peerStream, _ := NewPeerStream(s)
	node.peerStream = peerStream
	peerStream.Start()
}
