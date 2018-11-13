package net

import (
	"bufio"
	"context"
	"fmt"
	"sync"

	libnet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/najimmy/go-simplechain/rlp"
	log "github.com/sirupsen/logrus"
)

const (
	HELLO = iota + 1
	PEERS
	BLOCKS
	BYE
)

type P2PStream struct {
	mu                  sync.Mutex
	peerID              peer.ID
	addr                ma.Multiaddr
	stream              libnet.Stream
	node                *Node
	isFinishedHandshake bool
	isClosed            bool
	finshedHandshake    chan bool
	prevSendMsgType     int8
}

func NewP2PStream(node *Node, peerID peer.ID) (*P2PStream, error) {
	s, err := node.host.NewStream(context.Background(), peerID, "/simplechain/0.0.1")
	if err != nil {
		log.Warning("NewP2PStream : ", err)
		return nil, err
	}
	P2PStream := &P2PStream{node: node, stream: s, peerID: peerID, addr: s.Conn().RemoteMultiaddr(), finshedHandshake: make(chan bool, 1)}
	return P2PStream, nil
}

func NewP2PStreamWithStream(node *Node, s libnet.Stream) (*P2PStream, error) {
	fmt.Println(s.Conn().RemoteMultiaddr())
	P2PStream := &P2PStream{node: node, stream: s, peerID: s.Conn().RemotePeer(), addr: s.Conn().RemoteMultiaddr(), finshedHandshake: make(chan bool, 1)}
	return P2PStream, nil
}

func (P2PStream *P2PStream) Start() {
	rw := bufio.NewReadWriter(bufio.NewReader(P2PStream.stream), bufio.NewWriter(P2PStream.stream))
	go P2PStream.readData(rw)
}

func (ps *P2PStream) readData(rw *bufio.ReadWriter) {
	for {
		message := Message{}
		err := rlp.Decode(rw, &message)
		if err != nil {
			//time.Sleep(30 * time.Second)
			ps.stream.Close()
			ps.mu.Lock()
			ps.isClosed = true
			ps.mu.Unlock()
			ps.node.host.Peerstore().ClearAddrs(ps.peerID)
			//P2PStream.node.host.Peerstore().AddAddr(P2PStream.peerID, P2PStream.addr, 0)
			log.Warning("client closed")
			return
		}

		switch message.Code {
		case CMD_HELLO:
			ps.onHello(&message)
		case CMD_HELLO_ACK:
			ps.onHelloAck(&message)
		default:
			fmt.Println("lock...")
			if !ps.isFinishedHandshake {
				continue
			}
			fmt.Println("unlock...")
		}

		switch message.Code {
		case CMD_PEERS:
			ps.onPeers(&message)
		case CMD_PEERS_ACK:
			ps.onPeersAck(&message)

		}
	}
}

func writeData(rw *bufio.ReadWriter) {
	for {
	}
}

func (P2PStream *P2PStream) WaitFinshedHandshake() {
	P2PStream.mu.Lock()
	if !P2PStream.isFinishedHandshake {
		<-P2PStream.finshedHandshake
	}
	P2PStream.mu.Unlock()
}

//send Hello
func (P2PStream *P2PStream) SendHello() error {
	P2PStream.prevSendMsgType = HELLO
	if msg, err := NewRLPMessage(CMD_HELLO, P2PStream.node.maddr.String()); err != nil {
		return err
	} else {
		log.Info("SendHello")
		return P2PStream.sendMessage(&msg)
	}
}

func (P2PStream *P2PStream) SendHelloAck() error {
	P2PStream.prevSendMsgType = HELLO
	if msg, err := NewRLPMessage(CMD_HELLO_ACK, P2PStream.node.maddr.String()); err != nil {
		return err
	} else {
		log.Info("SendHelloAck")
		return P2PStream.sendMessage(&msg)
	}
}

func (P2PStream *P2PStream) onHello(message *Message) error {
	defer P2PStream.finshHandshake()
	data := string("")
	rlp.DecodeBytes(message.Payload, &data)
	log.WithFields(log.Fields{
		"Command": message.Code,
		"Data":    data,
	}).Info("onHello")
	node := P2PStream.node
	addr, err := ma.NewMultiaddr(data)
	if err != nil {
		return err
	}
	node.nodeRoute.Update(P2PStream.peerID, addr) //P2PStream.addr
	return P2PStream.SendHelloAck()
}

func (P2PStream *P2PStream) onHelloAck(message *Message) error {
	defer P2PStream.finshHandshake()
	data := string("")
	rlp.DecodeBytes(message.Payload, &data)
	log.WithFields(log.Fields{
		"Command": message.Code,
		"Data":    data,
	}).Info("onHello")
	node := P2PStream.node
	addr, err := ma.NewMultiaddr(data)
	if err != nil {
		return err
	}
	node.nodeRoute.Update(P2PStream.peerID, addr) //P2PStream.addr
	return nil
}

//send request peers
func (P2PStream *P2PStream) SendPeers() error {
	P2PStream.prevSendMsgType = PEERS
	if msg, err := NewRLPMessage(CMD_PEERS, "version 0.1"); err != nil {
		return err
	} else {
		log.Info("SendPeers")
		return P2PStream.sendMessage(&msg)
	}
}

func (P2PStream *P2PStream) SendPeersAck() error {
	log.Info("SendPeersAck>>>>>")
	node := P2PStream.node

	peers := node.nodeRoute.NearestPeers(P2PStream.peerID, 10)
	payload := make([][]string, 0)
	for k, addr := range peers {
		payload = append(payload, []string{k.Pretty(), addr.String()})
	}

	P2PStream.prevSendMsgType = PEERS
	//msg := Message{CMD_PEERS_ACK, hex.EncodeToString(b)}
	if msg, err := NewRLPMessage(CMD_PEERS_ACK, &payload); err != nil {
		return err
	} else {
		log.Info("<<<<<SendPeersAck")
		return P2PStream.sendMessage(&msg)
	}
}

func (P2PStream *P2PStream) onPeers(message *Message) error {
	data := string("")
	rlp.DecodeBytes(message.Payload, &data)
	log.WithFields(log.Fields{
		"Command": message.Code,
		"Data":    data,
	}).Info("onPeers>>>>>")
	log.Info("<<<<<onPeers")
	return P2PStream.SendPeersAck()
}

func (P2PStream *P2PStream) onPeersAck(message *Message) error {
	log.Info("onPeersAck>>>>>")
	payload := make([][]string, 0)

	err := rlp.DecodeBytes(message.Payload, &payload)
	if err != nil {
		fmt.Println(err)
		return err
	}

	node := P2PStream.node
	for _, addr := range payload {
		fmt.Printf("%v\n", addr)
		id, _ := peer.IDB58Decode(addr[0])
		maddr, _ := ma.NewMultiaddr(addr[1])
		node.nodeRoute.Update(id, maddr)
	}

	node.nodeRoute.NearestPeers(node.host.ID(), 10)
	log.Info("<<<<<onPeersAck")
	return nil
}

func (P2PStream *P2PStream) sendMessage(message *Message) error {
	encodedBytes, _ := rlp.EncodeToBytes(message)
	_, err := P2PStream.stream.Write(encodedBytes)
	if err != nil {
		//test host입장에서 muliple stream인건지
		//time.Sleep(30 * time.Second)
		P2PStream.stream.Close()
		P2PStream.mu.Lock()
		P2PStream.isClosed = true
		P2PStream.mu.Unlock()
		P2PStream.node.host.Peerstore().ClearAddrs(P2PStream.peerID)
		//P2PStream.node.host.Peerstore().AddAddr(P2PStream.peerID, P2PStream.addr, 0)
		log.Warning("sendMessage: client closed")
		return err
	}
	return nil
}

func (P2PStream *P2PStream) finshHandshake() {
	P2PStream.finshedHandshake <- true
	P2PStream.mu.Lock()
	P2PStream.isFinishedHandshake = true
	P2PStream.mu.Unlock()
}
