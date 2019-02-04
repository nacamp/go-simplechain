package net

import (
	"bufio"
	"errors"
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
	mu                 sync.RWMutex //sync.Mutex
	stream             libnet.Stream
	status             int
	HandshakeSucceedCh chan bool
	messageCh          chan *Message
	replys             *sync.Map
	handlers           *sync.Map
	inboud             bool
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

func (ps *PeerStream) callHandler(message *Message) {
	message.PeerID = ps.stream.Conn().RemotePeer()
	v, ok := ps.handlers.Load(message.Code)
	if ok {
		handler := v.(chan interface{})
		handler <- message
	}
}

func (ps *PeerStream) readData(rw *bufio.ReadWriter) {
	for {
		message := Message{}
		err := rlp.Decode(rw, &message)
		if err != nil {
			ps.stream.Close()
			ps.status = statusClosed
			log.CLog().WithFields(logrus.Fields{
				"Msg": err,
			}).Info("closed")
			ps.callHandler(&Message{Code: StatusStreamClosed})
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
		ps.callHandler(&message)
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
		log.CLog().WithFields(logrus.Fields{}).Debug("hostAddr: ", hostAddr.String())
		return ps.SendMessage(&msg)
	}
}

func (ps *PeerStream) SendHelloAck() error {
	if msg, err := NewRLPMessage(MsgHelloAck, ""); err != nil {
		return err
	} else {
		log.CLog().WithFields(logrus.Fields{}).Debug("ID: ", ps.stream.Conn().RemotePeer())
		err := ps.SendMessage(&msg)
		return err
	}
}

func (ps *PeerStream) onHello(message *Message) error {
	defer ps.finshHandshake()
	log.CLog().WithFields(logrus.Fields{}).Debug("ID: ", ps.stream.Conn().RemotePeer())
	message.PeerID = ps.stream.Conn().RemotePeer()
	v, ok := ps.handlers.Load(message.Code)
	if ok {
		log.CLog().WithFields(logrus.Fields{
			"ID": message.PeerID,
		}).Debug("handler")
		handler := v.(chan interface{})
		handler <- message
	}
	return ps.SendHelloAck()
}

func (ps *PeerStream) onHelloAck(message *Message) error {
	defer ps.finshHandshake()
	data := string("")
	rlp.DecodeBytes(message.Payload, &data)
	log.CLog().WithFields(logrus.Fields{}).Debug("ID: ", ps.stream.Conn().RemotePeer())
	ps.HandshakeSucceedCh <- true
	return nil
}

func (ps *PeerStream) finshHandshake() {
	ps.status = statusHandshakeSucceed
}

func (ps *PeerStream) SendMessage(message *Message) error {
	if ps.status != statusHandshakeSucceed && !(message.Code == MsgHello || message.Code == MsgHelloAck) {
		return errors.New("Handshake not completed")
	}
	encodedBytes, _ := rlp.EncodeToBytes(message)
	_, err := ps.stream.Write(encodedBytes)
	if err != nil {
		ps.stream.Close()
		ps.status = statusClosed
		log.CLog().WithFields(logrus.Fields{
			"Msg": err,
		}).Info("closed")
		ps.callHandler(&Message{})
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
	return ps.SendMessage(message)
}

func (ps *PeerStream) Register(code uint64, handler chan interface{}) {
	ps.handlers.Store(code, handler)
}

func (ps *PeerStream) IsClosed() bool {
	return ps.status == statusClosed
}

func (ps *PeerStream) IsHandshakeSucceed() bool {
	return ps.status == statusHandshakeSucceed
}
