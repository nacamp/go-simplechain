package net

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"

	libnet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
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

func (P2PStream *P2PStream) readData(rw *bufio.ReadWriter) {
	for {
		b, err := rw.ReadBytes('\n')
		if len(b) == 0 {
			//time.Sleep(30 * time.Second)
			P2PStream.stream.Close()
			P2PStream.mu.Lock()
			P2PStream.isClosed = true
			P2PStream.mu.Unlock()
			P2PStream.node.host.Peerstore().ClearAddrs(P2PStream.peerID)
			//P2PStream.node.host.Peerstore().AddAddr(P2PStream.peerID, P2PStream.addr, 0)
			log.Warning("client closed")
			return
		}
		if len(b) < 2 {
			continue
		}
		if err != nil {
			//return err
		}
		var message Message
		message.Unmarshal(b)

		switch message.Command {
		case "HELLO":
			P2PStream.onHello(&message)
		case "HELLO-ACK":
			P2PStream.onHelloAck(&message)
		default:
			fmt.Println("lock...")
			if !P2PStream.isFinishedHandshake {
				continue
			}
			fmt.Println("unlock...")
		}

		switch message.Command {
		case "PEERS":
			P2PStream.onPeers(&message)
		case "PEERS-ACK":
			P2PStream.onPeersAck(&message)
		case "BLOCKS":
			fmt.Println("Blocks")
		case "BLOCKS-ACK":
			fmt.Println("Blocks")
		case "BYE":
			fmt.Println("Bye")
		case "BYE-ACK":
			fmt.Println("Bye")
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
	msg := Message{"HELLO", P2PStream.node.maddr.String()}
	log.Info("SendHello")
	return P2PStream.sendMessage(&msg)
}

func (P2PStream *P2PStream) SendHelloAck() error {
	P2PStream.prevSendMsgType = HELLO
	msg := Message{"HELLO-ACK", P2PStream.node.maddr.String()}
	log.Info("SendHelloAck")
	return P2PStream.sendMessage(&msg)
}

func (P2PStream *P2PStream) onHello(message *Message) error {
	defer P2PStream.finshHandshake()
	log.WithFields(log.Fields{
		"Command": message.Command,
		"Data":    message.Data,
	}).Info("onHello")
	node := P2PStream.node
	addr, err := ma.NewMultiaddr(message.Data)
	if err != nil {
		return err
	}
	node.nodeRoute.Update(P2PStream.peerID, addr) //P2PStream.addr
	return P2PStream.SendHelloAck()
}

func (P2PStream *P2PStream) onHelloAck(message *Message) error {
	defer P2PStream.finshHandshake()
	log.WithFields(log.Fields{
		"Command": message.Command,
		"Data":    message.Data,
	}).Info("onHello")
	node := P2PStream.node
	addr, err := ma.NewMultiaddr(message.Data)
	if err != nil {
		return err
	}
	node.nodeRoute.Update(P2PStream.peerID, addr) //P2PStream.addr
	return nil
}

//send request peers
func (P2PStream *P2PStream) SendPeers() error {
	P2PStream.prevSendMsgType = PEERS
	msg := Message{"PEERS", "version 0.1"}
	log.Info("SendPeers")
	return P2PStream.sendMessage(&msg)
}

func (P2PStream *P2PStream) SendPeersAck() error {
	log.Info("SendPeersAck>>>>>")
	//먼저 어딘가에 저장...
	node := P2PStream.node

	peers := node.nodeRoute.NearestPeers(P2PStream.peerID, 10)
	var m map[string]string
	m = make(map[string]string)
	for k, addr := range peers {
		m[k.Pretty()] = addr.String()
	}
	b, err := json.Marshal(m)
	if err != nil {
		fmt.Println(err)
		return err
	}
	var m2 map[string]string
	m2 = make(map[string]string)
	json.Unmarshal(b, &m2)

	P2PStream.prevSendMsgType = PEERS
	msg := Message{"PEERS-ACK", hex.EncodeToString(b)}
	log.Info("<<<<<SendPeersAck")
	return P2PStream.sendMessage(&msg)
}

func (P2PStream *P2PStream) onPeers(message *Message) error {
	log.WithFields(log.Fields{
		"Command": message.Command,
		"Data":    message.Data,
	}).Info("onPeers>>>>>")
	log.Info("<<<<<onPeers")
	return P2PStream.SendPeersAck()
}

func (P2PStream *P2PStream) onPeersAck(message *Message) error {
	// log.WithFields(log.Fields{
	// 	"Command": message.Command,
	// 	"Data":    message.Data,
	// }).Info("onPeersAck>>>>>")
	log.Info("onPeersAck>>>>>")
	b, err := hex.DecodeString(message.Data)
	if err != nil {
		return err
	}
	var m map[string]string
	m = make(map[string]string)
	json.Unmarshal(b, &m)

	node := P2PStream.node
	for k, addr := range m {
		id, _ := peer.IDB58Decode(k)
		addr2, _ := ma.NewMultiaddr(addr)
		node.nodeRoute.Update(id, addr2)
	}
	node.nodeRoute.NearestPeers(node.host.ID(), 10)
	log.Info("<<<<<onPeersAck")
	return nil
}

func (P2PStream *P2PStream) sendMessage(message *Message) error {
	_, err := P2PStream.stream.Write(append(message.Marshal(), '\n'))
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

func receiveMessage(s libnet.Stream) (*Message, error) {
	buf := bufio.NewReader(s)
	out, err := buf.ReadBytes('\n')
	fmt.Println(len(out))
	// out, err := ioutil.ReadAll(s)
	if err != nil {
		//log.Fatalln(err)
		return nil, err
	}
	var message Message
	message.Unmarshal(out)
	return &message, nil
}
