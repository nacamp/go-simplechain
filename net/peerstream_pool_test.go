package net

import (
	// "fmt"
	"fmt"
	"testing"
	"time"

	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/nacamp/go-simplechain/rlp"
	"github.com/stretchr/testify/assert"
	// peer "github.com/libp2p/go-libp2p-peer"
	// ma "github.com/multiformats/go-multiaddr"
	// "github.com/stretchr/testify/assert"
)

func TestPeerStreamPool(t *testing.T) {
	pool := NewPeerStreamPool()
	hander := NewTestPeerStreamHandler()
	pool.AddHandler(hander)

	//16Uiu2HAkwR1pV8ZR8ApcZWrMSw5iNMwaJHFpKr91H9a1a65WGehk
	sn := NewTestNode("08021220a178bc3f8ee6738af0139d9784519e5aa1cb256c12c54444bd63296502f29e94", "/ip4/127.0.0.1/tcp/9991")
	//16Uiu2HAkxKaG3PHSLfDhfZ7a8YzP6w6fKooBTY1gmfSXxGYbsNuN
	cn := NewTestNode("080212201afa45f64cd5a28cd40e178889ed2e9f987658bc4d48d376ef6ecb1ab1b26211", "/ip4/127.0.0.1/tcp/9992")
	sn.Start(true)
	cn.Start(false)
	addr, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/9991")
	id, _ := peer.IDB58Decode("16Uiu2HAkwR1pV8ZR8ApcZWrMSw5iNMwaJHFpKr91H9a1a65WGehk")
	cn.Connect(id, addr)
	sn.peerStream.Start()
	cn.peerStream.Start()

	cn.peerStream.SendHello(cn.maddr)
	<-cn.peerStream.HandshakeSucceedCh

	assert.Equal(t, int32(0), pool.count)
	pool.AddStream(sn.peerStream)
	pool.AddStream(cn.peerStream)
	assert.Equal(t, int32(2), pool.count)

	ps2, _ := pool.GetStream(sn.peerStream.stream.Conn().RemotePeer())
	assert.Equal(t, sn.peerStream, ps2)

	msg, _ := NewRLPMessage(MsgNewBlock, "new block")
	sn.peerStream.SendMessage(&msg)
	msg2, _ := NewRLPMessage(MsgNewTx, "waiting")
	sn.peerStream.SendMessage(&msg2)

	cn.peerStream.stream.Close()
	sn.peerStream.stream.Close()

	ticker := time.NewTicker(100 * time.Millisecond)
	for t := range ticker.C {
		fmt.Println("Tick at", t)
		if int32(0) == pool.count {
			break
		}
	}
	<-hander.MsgNewTxCh
}

type TestPeerStreamHandler struct {
	MsgNewBlockCh chan interface{}
	MsgNewTxCh    chan interface{}
	pool          *PeerStreamPool
}

// func NewTestPeerStreamHandler(pool *PeerStreamPool) *TestPeerStreamHandler {
// 	return &TestPeerStreamHandler{
// 		MsgNewBlockCh: make(chan interface{}, 1),
// 		MsgNewTxCh:    make(chan interface{}, 1),
// 		pool:          pool,
// 	}
// }

func NewTestPeerStreamHandler() *TestPeerStreamHandler {
	return &TestPeerStreamHandler{
		MsgNewBlockCh: make(chan interface{}, 1),
		MsgNewTxCh:    make(chan interface{}, 1),
	}
}

func (p *TestPeerStreamHandler) Register(peerStream *PeerStream) {
	peerStream.Register(MsgNewBlock, p.MsgNewBlockCh)
	peerStream.Register(MsgNewTx, p.MsgNewTxCh)
}

func (p *TestPeerStreamHandler) StartHandler() {
	go func() {
		for {
			select {
			case ch := <-p.MsgNewBlockCh:
				msg := ch.(*Message)
				data := string("")
				rlp.DecodeBytes(msg.Payload, &data)
				fmt.Println(data)
			}
		}

	}()
}
