package net

import (
	"errors"
	"math/rand"
	"time"

	kb "github.com/libp2p/go-libp2p-kbucket"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
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

func (d *Discovery) findnode(peerInfo *peerstore.PeerInfo, targetID peer.ID, reply chan<- []*peerstore.PeerInfo) {
	reply <- d._findnode(peerInfo, targetID)
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
