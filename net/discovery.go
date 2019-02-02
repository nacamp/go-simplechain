package net

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	kb "github.com/libp2p/go-libp2p-kbucket"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/nacamp/go-simplechain/rlp"
)

const (
	ConcurrencyLimit = 3
	BucketSize       = 16
)

type Discovery struct {
	peerstore    peerstore.Peerstore
	routingTable *kb.RoutingTable
	_findnode    func(peerInfo *peerstore.PeerInfo, targetID peer.ID) []*peerstore.PeerInfo
	_bond        func(peerInfo *peerstore.PeerInfo) *peerstore.PeerInfo

	MsgNearestPeersCh    chan interface{}
	MsgNearestPeersAckCh chan interface{}
	node                 *Node
}

func NewDiscovery(hostID peer.ID, metrics peerstore.Metrics, peerstore peerstore.Peerstore) *Discovery {
	d := &Discovery{}
	d.routingTable =
		kb.NewRoutingTable(BucketSize, kb.ConvertPeerID(hostID), time.Minute, metrics)
	d.peerstore = peerstore
	return d
}

func (d *Discovery) Update(peerInfo *peerstore.PeerInfo) {
	d.routingTable.Update(peerInfo.ID)
	d.peerstore.AddAddrs(peerInfo.ID, peerInfo.Addrs, time.Duration(3600)*time.Second)
}

func (d *Discovery) RandomPeerInfo() (peerstore.PeerInfo, error) {
	ids := d.peerstore.Peers()
	size := len(ids)
	if size == 0 {
		return peerstore.PeerInfo{}, errors.New("Not found peerinfo")
	}
	rand.Seed(time.Now().Unix())
	id := ids[rand.Intn(len(ids))]
	return d.peerstore.PeerInfo(id), nil
}

func (d *Discovery) bond(peerInfo *peerstore.PeerInfo, reply chan<- *peerstore.PeerInfo) {
	reply <- d._bond(peerInfo)
	d.Update(peerInfo)
}

func sortByDistance(peerInfos []*peerstore.PeerInfo, targetID peer.ID) []*peerstore.PeerInfo {
	IDs := make([]peer.ID, 0, len(peerInfos))
	infos := make(map[peer.ID]*peerstore.PeerInfo)
	for i, id := range peerInfos {
		IDs = append(IDs, id.ID)
		infos[id.ID] = peerInfos[i]
	}

	peers := kb.SortClosestPeers(IDs, kb.ConvertPeerID(targetID))
	closestSize := ConcurrencyLimit
	if len(peers) < ConcurrencyLimit {
		closestSize = len(peers)
	}
	closet := make([]*peerstore.PeerInfo, 0, closestSize)
	for _, ID := range peers[:closestSize] {
		closet = append(closet, infos[ID])
	}
	return closet
}

func (d *Discovery) randomLookup() error {
	//TODO: rlock
	peers := d.routingTable.ListPeers()
	size := len(peers)
	if size == 0 {
		return errors.New("Not found peer")
	}
	rand.Seed(time.Now().Unix())
	id := peers[rand.Intn(size)]

	return d.lookup(id)
}

func (d *Discovery) NearestPeers(peerID peer.ID) []*peerstore.PeerInfo {
	closest := d.routingTable.NearestPeers(kb.ConvertPeerID(peerID), BucketSize)
	closestPeerInfo := make([]*peerstore.PeerInfo, 0, BucketSize)
	for _, id := range closest {
		p := d.peerstore.PeerInfo(id)
		closestPeerInfo = append(closestPeerInfo, &p)
	}
	return closestPeerInfo
}

func (d *Discovery) findnode(peerInfo *peerstore.PeerInfo, targetID peer.ID, reply chan<- []*peerstore.PeerInfo) {
	if d._findnode != nil {
		reply <- d._findnode(peerInfo, targetID)
		return
	}

	peerStream, err := d.node.Connect(peerInfo.ID, peerInfo.Addrs[0])
	//TODO:
	if err != nil {
		fmt.Println(err)
		return
	}
	//TODO: make func
	if peerStream.status != statusHandshakeSucceed {
		peerStream.SendHello(d.node.maddr)
		//TODO: timeout
		<-peerStream.HandshakeSucceedCh
	}

	msg, _ := NewRLPMessage(MsgNearestPeers, targetID)
	replyAck := make(chan interface{}, 1)
	peerStream.SendMessageReply(&msg, replyAck)
	ack := <-replyAck
	reply <- ack.([]*peerstore.PeerInfo)
}

func (d *Discovery) Register(peerStream *PeerStream) {
	peerStream.Register(MsgNearestPeers, d.MsgNearestPeersCh)
	peerStream.Register(MsgNearestPeersAck, d.MsgNearestPeersAckCh)
}

func (d *Discovery) startHandler() {
	go d.handleMsgNearestPeers()
	go d.handleMsgNearestPeersAck()
}

func (d *Discovery) SendNearestPeers(targetID peer.ID, ps *PeerStream) error {
	closestPeerInfo := d.NearestPeers(targetID)
	payload := make([]*PeerInfo2, 0)
	for _, info := range closestPeerInfo {
		// p := d.peerstore.PeerInfo(id)
		payload = append(payload, ToPeerInfo2(info))
	}
	if msg, err := NewRLPMessage(MSG_PEERS_ACK, &payload); err != nil {
		return err
	} else {
		ps.SendMessage(&msg)
	}
	return nil
}

func (d *Discovery) handleMsgNearestPeers() {
	for {
		select {
		case ch := <-d.MsgNearestPeersCh:
			// fmt.Println(msg)
			msg := ch.(*Message)
			fmt.Println(msg.Code)
			var targetID peer.ID
			_ = rlp.DecodeBytes(msg.Payload, &targetID)
			ps, _ := d.node.StreamPool.GetStream(msg.PeerID)
			d.SendNearestPeers(targetID, ps)
		}
	}
}

func (d *Discovery) handleMsgNearestPeersAck() {
	for {
		select {
		case ch := <-d.MsgNearestPeersAckCh:
			msg := ch.(*Message)
			ps, _ := d.node.StreamPool.GetStream(msg.PeerID)
			v, ok := ps.replys.Load(msg.Code)
			if ok {
				payload := make([]*PeerInfo2, 0)
				_ = rlp.DecodeBytes(msg.Payload, &payload)

				reply := make([]*peerstore.PeerInfo, 0)
				for _, info := range payload {
					// p := d.peerstore.PeerInfo(id)
					reply = append(reply, FromPeerInfo2(info))
				}

				replyCh := v.(chan interface{})
				replyCh <- reply
			}
		}
	}
}

func (d *Discovery) lookup(peerID peer.ID) error {
	var (
		ask         = make([]*peerstore.PeerInfo, ConcurrencyLimit)
		asked       = make(map[peer.ID]bool) // called findnode
		seen        = make(map[peer.ID]bool) // called bond
		reply       = make(chan []*peerstore.PeerInfo, ConcurrencyLimit)
		seenInfos   = make([]*peerstore.PeerInfo, 0)
		bondReply   = make(chan *peerstore.PeerInfo, BucketSize*ConcurrencyLimit)
		askPending  = 0
		bondPending = 0
	)

	closest := d.routingTable.NearestPeers(kb.ConvertPeerID(peerID), BucketSize)
	closestPeerInfo := make([]*peerstore.PeerInfo, 0, BucketSize)
	for _, id := range closest {
		p := d.peerstore.PeerInfo(id)
		closestPeerInfo = append(closestPeerInfo, &p)
	}
	ask = sortByDistance(closestPeerInfo, peerID)

	for len(ask) > 0 {
		if askPending == 0 {
			for _, v := range ask {
				askPending++
				asked[v.ID] = true
				//fmt.Println("asked ", len(asked))
				go d.findnode(v, peerID, reply)
			}
		}
		for _, n := range <-reply {
			//if n != nil && !seen[n.ID] && !asked[n.ID] {
			if n != nil && !asked[n.ID] {
				if !seen[n.ID] {
					seen[peerID] = true
					bondPending++
					//fmt.Println(bondPending)
					go d.bond(n, bondReply)
				}
			}
		}
		askPending--
		if askPending == 0 {
			if bondPending == 0 {
				ask = ask[:0]
			} else {
				for n := range bondReply {
					seenInfos = append(seenInfos, n)
					bondPending--
					//fmt.Println(bondPending)
					if bondPending == 0 {
						break
					}
				}
				ask = sortByDistance(seenInfos, peerID)
				seenInfos = seenInfos[:0]
				//fmt.Println("ask ", len(ask))
			}
		}

	}
	close(reply)
	close(bondReply)
	return nil
}
