package net

import (
	kb "github.com/libp2p/go-libp2p-kbucket"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
)

const (
	kConcurrency = 3
	kBucketSize  = 16
)

type Discovery struct {
	peerstore    peerstore.Peerstore
	routingTable *kb.RoutingTable
}

func (d *Discovery) findnode(peerInfo *peerstore.PeerInfo, targetID peer.ID, reply chan<- []*peerstore.PeerInfo) ([]interface{}, error) {
	return nil, nil
}

func (d *Discovery) bond(peerInfo *peerstore.PeerInfo) {
}

func sortByDistance(peerInfos []*peerstore.PeerInfo, targetID peer.ID) []*peerstore.PeerInfo {
	IDs := make([]peer.ID, 0, len(peerInfos))
	infos := make(map[peer.ID]*peerstore.PeerInfo)
	for i, id := range peerInfos {
		IDs = append(IDs, id.ID)
		infos[id.ID] = peerInfos[i]
	}

	peers := kb.SortClosestPeers(IDs, kb.ConvertPeerID(targetID))
	closestSize := kConcurrency
	if len(peers) < kConcurrency {
		closestSize = len(peers)
	}
	closet := make([]*peerstore.PeerInfo, 0, closestSize)
	for _, ID := range peers {
		closet = append(closet, infos[ID])
	}
	return closet
}

func (d *Discovery) lookup(peerID peer.ID) error {
	var (
		ask       = make([]*peerstore.PeerInfo, 3)
		asked     = make(map[peer.ID]bool) // called findnode
		seen      = make(map[peer.ID]bool) // called bond
		reply     = make(chan []*peerstore.PeerInfo, kConcurrency)
		seenInfos = make([]*peerstore.PeerInfo, 1)
		pending   = 0
	)

	closest := d.routingTable.NearestPeers(kb.ConvertPeerID(peerID), kBucketSize)
	tmp := make([]*peerstore.PeerInfo, kBucketSize)
	for _, id := range closest {
		p := d.peerstore.PeerInfo(id)
		tmp = append(tmp, &p)
	}
	ask = sortByDistance(tmp, peerID)

	for len(ask) > 0 {
		if pending == 0 {
			for _, v := range ask {
				pending++
				asked[v.ID] = true
				go d.findnode(v, peerID, reply)
			}
		}
		for _, n := range <-reply {
			if n != nil && !seen[n.ID] && !asked[n.ID] {
				seen[peerID] = true
				//go d.bond(n)
				seenInfos = append(seenInfos, n)
			}
		}
		pending--
		if pending == 0 {
			ask = sortByDistance(seenInfos, peerID)
			seenInfos = seenInfos[:0]
		}

	}
	return nil
}

/*
	nodeRoute.routingTable.Update(peerid)
		targetPeerAddr, _ := ma.NewMultiaddr(
		fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)))
	targetAddr := ipfsaddr.Decapsulate(targetPeerAddr) //  /ip4/127.0.0.1/tcp/9990
	nodeRoute.Update(peerid, targetAddr)
	nodeRoute.node.host.Peerstore().AddAddr(peerid, targetAddr, pstore.PermanentAddrTTL)
	nodeRoute.node.seedID = peerid

*/
