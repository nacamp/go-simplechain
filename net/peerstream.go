package net

import (
	"bufio"
	"errors"
	"fmt"
	"sync"

	libnet "github.com/libp2p/go-libp2p-net"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/rlp"
	"github.com/sirupsen/logrus"
)

const (
	statusInit = iota
	statusHandshakeSucceed
	statusClosed
)

type PeerStream struct {
	mu sync.RWMutex //sync.Mutex
	// peerID              peer.ID
	// hostAddr            ma.Multiaddr
	stream libnet.Stream
	// node                *Node
	// discovery           *Discovery
	// isFinishedHandshake bool
	status int
	// finshedHandshakeCh  chan bool
	HandshakeSucceedCh chan bool
	messageCh          chan *Message
	// SendHelloCh        chan bool
	replys   *sync.Map
	handlers *sync.Map
}

func NewPeerStream(s libnet.Stream) (*PeerStream, error) {
	PeerStream := &PeerStream{
		stream:             s,
		status:             statusInit,
		messageCh:          make(chan *Message, 5),
		HandshakeSucceedCh: make(chan bool),
		replys:             new(sync.Map),
		handlers:           new(sync.Map),
	}
	return PeerStream, nil
}

func (ps *PeerStream) Start() { //isHost bool
	log.CLog().Debug("Start")
	rw := bufio.NewReadWriter(bufio.NewReader(ps.stream), bufio.NewWriter(ps.stream))
	go ps.readData(rw)
	// go ps.writeData(rw)
}

func (ps *PeerStream) readData(rw *bufio.ReadWriter) {
	fmt.Println("readData")
	for {
		message := Message{}
		err := rlp.Decode(rw, &message)
		if err != nil {
			//time.Sleep(30 * time.Second)
			ps.stream.Close()
			log.CLog().Debug("readData  lock before")
			ps.mu.Lock()
			ps.status = statusClosed
			ps.mu.Unlock()
			log.CLog().Debug("readData  Unlock after")
			//ps.node.host.Peerstore().ClearAddrs(ps.peerID)
			//P2PStream.node.host.Peerstore().AddAddr(P2PStream.peerID, P2PStream.addr, 0)
			log.CLog().Debug("client closed")
			return
		}
		switch message.Code {
		case MsgHello:
			ps.onHello(&message)
		case MsgHelloAck:
			ps.onHelloAck(&message)
		default:
			if ps.status != statusHandshakeSucceed {
				continue
			}
		}
		message.PeerID = ps.stream.Conn().RemotePeer()
		v, ok := ps.handlers.Load(message.Code)
		if ok {
			handler := v.(chan interface{})
			handler <- &message
		}
	}
}

/*
client, server
c:SendHello => s:onHello , SendHelloAck  => c:onHelloAck
*/
func (ps *PeerStream) SendHello(hostAddr ma.Multiaddr) error {
	if msg, err := NewRLPMessage(MsgHello, hostAddr.String()); err != nil {
		return err
	} else {
		log.CLog().Debug("SendHello")
		return ps.SendMessage(&msg)
	}
}

func (ps *PeerStream) SendHelloAck() error {
	if msg, err := NewRLPMessage(MsgHelloAck, ""); err != nil {
		return err
	} else {
		log.CLog().Debug("SendHelloAck")
		err := ps.SendMessage(&msg)
		return err
	}
}

func (ps *PeerStream) onHello(message *Message) error {
	defer ps.finshHandshake()
	data := string("")
	rlp.DecodeBytes(message.Payload, &data)
	log.CLog().WithFields(logrus.Fields{
		"Command": message.Code,
		"Data":    data,
	}).Debug("onHello")

	// node := ps.node
	addr, err := ma.NewMultiaddr(data)
	if err != nil {
		return err
	}
	fmt.Println("server receive:", addr)
	//node.nodeRoute.Update(ps.peerID, addr) //P2PStream.addr
	return ps.SendHelloAck()
}

func (ps *PeerStream) onHelloAck(message *Message) error {
	defer ps.finshHandshake()
	data := string("")
	rlp.DecodeBytes(message.Payload, &data)
	log.CLog().WithFields(logrus.Fields{
		"Command": message.Code,
		"Data":    data,
	}).Debug("onHelloAck")
	fmt.Println("client receive:", ps.stream.Conn().RemoteMultiaddr())
	ps.HandshakeSucceedCh <- true

	// node := ps.node
	// addr, err := ma.NewMultiaddr(data)
	// if err != nil {
	// 	return err
	// }
	// node.nodeRoute.Update(ps.peerID, addr) //P2PStream.addr
	return nil
}

func (ps *PeerStream) finshHandshake() {
	log.CLog().Debug("finshHandshake lock before")
	ps.mu.Lock()
	ps.status = statusHandshakeSucceed
	ps.mu.Unlock()
	log.CLog().Debug("finshHandshake Unlock after")
	log.CLog().Debug("finshHandshake")
}

func (ps *PeerStream) SendMessage(message *Message) error {
	if ps.status != statusHandshakeSucceed && !(message.Code == MsgHello || message.Code == MsgHelloAck) {
		return errors.New("Handshake not completed")
	}
	encodedBytes, _ := rlp.EncodeToBytes(message)
	_, err := ps.stream.Write(encodedBytes)
	if err != nil {
		ps.stream.Close()
		log.CLog().Debug("sendMessage lock before")
		ps.mu.Lock()
		ps.status = statusClosed
		ps.mu.Unlock()
		log.CLog().Debug("sendMessage Unlock after")
		//ps.node.host.Peerstore().ClearAddrs(ps.peerID)
		//ps.node.host.Peerstore().AddAddr(ps.peerID, ps.addr, 0)
		log.CLog().Warning("sendMessage: client closed")
		return err
	}
	return nil
}

/*
client, server
c:SendMessageReply => s:onXXXX , XXXXAck  (handler)=> c:onXXXXAck
*/
func (ps *PeerStream) SendMessageReply(message *Message, reply chan interface{}) error {
	ps.replys.Store(message.Code+uint64(1), reply)
	// reply <- "callback"
	return ps.SendMessage(message)
}

func (ps *PeerStream) Register(code uint64, handler chan interface{}) {
	ps.handlers.Store(code, handler)
}

// func (ps *PeerStream) writeData(rw *bufio.ReadWriter) {
// 	for {
// 		for m := range ps.messageCh {
// 			ps.SendMessage(m)
// 		}
// 		// select {
// 		// case message := <-ps.messageCh:
// 		// 	ps.sendMessage(message)
// 		// 	// continue
// 		// 	// default: why deault
// 		// }
// 	}
// }
