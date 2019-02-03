package net

import (
	"bufio"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	crypto "github.com/libp2p/go-libp2p-crypto"
	kb "github.com/libp2p/go-libp2p-kbucket"
	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p-peerstore/pstoremem"
	ma "github.com/multiformats/go-multiaddr"
	multiaddr "github.com/multiformats/go-multiaddr"
)

var IDS = `16Uiu2HAmUPKnHgcfwzhZLheKfpfKP5d9ysRJ7GYmiPkXtRhd9Lha
16Uiu2HAmP2AoC5Z9vNKJR9ejnVePzoiFWqCzxMCBDV98YpLKtgVv
16Uiu2HAmRMh1XZApwoZcpgkrsH1rrUFf6hKhomGH5cyVxS9mg1N5
16Uiu2HAm4NqKeXrnyVMjvR9xBLYpoE46pE3Com6FFgkJJ9Rd1nxv
16Uiu2HAmJfzv6Koem2G3xUHKnCEqUeskybP5JP3QYWHLQWiBd48Y
16Uiu2HAm34s8rn8QyhxsMqS4sGR8nJcdGJB8xDGJMoiSH8bTqpW6
16Uiu2HAm4U5WaprbEQBdJfNioCBjyoCn6Gti2ZsDVTDT5XFxtDXT
16Uiu2HAmPk6NPRM1yDJEHvtP6iXrdjkKEdFnkn4msticmSEoXPME
16Uiu2HAmSbUA1VEMCkkjG9eXzyRXFrGJaUZQFRYoFriZTJFCGxSD
16Uiu2HAmJwn7ZnacsqTMwYKmDeV36bt2u9HVKwmYD4aTPrv7haYe
16Uiu2HAm8c9CE9gX9HTLqjNx3C1dWBoiryH9iffD81noNpncdyy1
16Uiu2HAkyYLGG53HWFdqweP14ZEGx4BMzRWoWYDa5qbAvVwab8JB
16Uiu2HAm66g9JXcDe6oFeh61eaSqQ4yXfBY6Ni7TnETHdemU1WN8
16Uiu2HAmPXHnis4RuCRLMGU2zQXTwmwPTnAstGCrRLjKmHvAxyu3
16Uiu2HAkzNJR37rJb9KgUmiB9re3LidtQaBYanedN4f6XaYDd8Su
16Uiu2HAmG8FjN89BNQfMxuYMq7zbAryCSF4dYNLS2TLZgDy8PSwm
16Uiu2HAkyiZS8HErcTwoPSU8gpkTLj4fCe8uccJKnDbXz9SPtHGY
16Uiu2HAkyiPHimUuYsEEbBenFpnsXrN1BHbvGLL3JPXEoRTzhiQq
16Uiu2HAmK3rEqFLGZQGt9y79x8tMBFWv2QU4Gvob3X6TGciFL34x
16Uiu2HAm3K7TVNnuTSPTy2PkL5RQe7QnxrfHYrq12aAGoMNYHNdQ
16Uiu2HAm5nNKmQoVUWXM3cx3rJhRnkC5bzK3gJue6jdVhVUApvfy
16Uiu2HAmHvn8ik7TM85MB9NDenfHyFUTxhkUMpaEso4e6svk3MB8
16Uiu2HAmM2UPh4CSNhYiEQ2t4sB6un8pZ2GQgCNHZZm9DKPKcRwh
16Uiu2HAm2Uf175v4GA49zzayW2AD79EkYVhJJeE4Cw5AeqR2Djob
16Uiu2HAkzwywTMXH35baa1LwaMEHsgm195Qpsg9kVUMutdQB6QL2
16Uiu2HAmNTr4fDq3vp4qxB3CsfJEC6rUbaoMZzX81ezPcrJnYGWE
16Uiu2HAmRaQgdanB3dadjL4jCGq6nfjJGZfsNSphtojMuXhjnjz1
16Uiu2HAmAgrRKn3hfZpAvCDmDcNTUFFTxyz7cBZrcpGvc1bZyiom
16Uiu2HAmN9mRBxs9CnKfRDz6w1oNSz9WmJxkXWuE9SUmxeUUva1m
16Uiu2HAmSMxDpamp72n4bsEdzuLBqnjcr1MYXAsrc1K5EDPiB64i
16Uiu2HAkv3VeGArcroASSiDcg5ZzH7JG3oo4DThJwS7GDkdN9FY1
16Uiu2HAm8G9u5UUArpq2Q7y6KV3zFNG6DPdfBKFQaD7MfRg4trtU
16Uiu2HAm1kq9aNJWPvxGmvC3Bqq2NrtNYx3gweYYGhQn2GJ6hast
16Uiu2HAmEZamuiBq1GUBU1hu23KRExAmjrQU1Ncfv8SAv35yu7KA
16Uiu2HAmP2tVD3RZNKHnhJMc33xeXQ3uZw9CQaohPFNLiVPNirf9
16Uiu2HAmDnkTdj9nXunCjedeTuUFfVNLzd11btsRSeWsaxioUjLY
16Uiu2HAmLNFA59JhH3GUzPRnpR1pmqFnrygD22ErfRjXE8HJy9CV
16Uiu2HAmGkTjuLzDJdPxQk2pbgkG7Rfb1CRE5HVE2KqkBWYSfr3w
16Uiu2HAmCrTW38GKfdyaA2sHcVGvX4aC8YNduTGab11khLpJrwF6
16Uiu2HAmPwvDrEcLEn4QUcTeMaqH6cVd11pxKzLRhjCr4XU3DUCx
16Uiu2HAm7ooiYFE7mf3kqCEMNJzbMSGb7FvtdNmos5VyGZtBHByg
16Uiu2HAmCStzNwtsjTGn43XhoWGU3MBBdwdoG7uVsAERS1bCKpi1
16Uiu2HAmF7i9HLxwS6Xzgoku6iN1m687nNcxWWrzj9z3msBCHfQY
16Uiu2HAmQjx6tvuycRGYJxbXR97jiFsTBwsswQGqyPMueTe4f2YK
16Uiu2HAmHPwky4vjFYPHDdXpe1nSjhCKKdTFDAQvgzGA3eHhR8Tm
16Uiu2HAmPXe1aA5jVwSv3b7i2aNQq9bsiVcS2K2JVyujWiHtQZ5M
16Uiu2HAkzn3LVKhdBoPSM4thjso2shQRFpsBBvG6nBvjdL4jjKRB
16Uiu2HAmNSMmmP7uiSUST9sov2f8NQAZt6hF79ZwzwtoQuP8Eqhe
16Uiu2HAm1AEukCT4BuiXtEvEWfmzWjVDkdGDYNeFnxMMLBC3UT3F
16Uiu2HAm2GarfJoS2ipqZQd2GC7VrYUG9xRiEMqw67s9yiGppn7N
16Uiu2HAm7vsE7k2CH5iQ5rLCHVSELYwRm9xRQ6uZ6pL59ViQdMRi
16Uiu2HAkxw6QPeDNCKU7TF2uB7PTPYj7hemz3FM99KA7qKfPQzVK
16Uiu2HAkvuxMXqRd5jWYeGz6voT2pTX2oK65uJfi4c8KNRzwcijE
16Uiu2HAmCdSgWctEBeKJ2kXkTaULr32yeGJ2auHwEc4jhPEb73X2
16Uiu2HAm27KA6NrRRm9Lcn1xpxxanXviu8kXj4cG5JeY3A2KRKbA
16Uiu2HAm5wDfTTHqB4WY2TXiFfip1yTPzEjaAHUbF3YQuruEsJLZ
16Uiu2HAmFYWeVwcxwzq5ZdoAu3L3fqKZZJraGcv8dSufUqbp5bqN
16Uiu2HAky6vuupNgmHzt1Z66gWwugM89cMczz6N6znkTVKFWknV2
16Uiu2HAmHw91DniqzF6ZhrrBohvRNdBJfe3g96fQeC6QbYrkncuC
16Uiu2HAkuZA4aqPfJvyZf29mcDMTrkd8CDw5JxFmjg63phk4a4qw
16Uiu2HAmESFW1f17baAnYrQrxzJzdRLer1Tos1uUyBmJa54pokCo
16Uiu2HAmBi25zfxNzxwY7a49XmMHEqS1HtnMKwqUsMP9AeYDt3KR
16Uiu2HAmJ22e6xxXcaMDsEZWeLKcCPM1oLQPjPzPvoCDWJu9vwBg
16Uiu2HAkz4CsYBLJpYWJJA9GjuyWhAzZvTNH9YdJkP6mwytr1o74
16Uiu2HAmDV78j26LPA54a3MfE2fyCGVxWGLk43XZFyjdiMQ1tTMQ
16Uiu2HAmCdRKuHQSsyCPPnEq3VZnjncNqX3e14ULcvF5hibL5G3c
16Uiu2HAmCP9V8rE45DQWSy6QrSvbDsVkPkSNjjr4QrkFskJwRwm4
16Uiu2HAm2xcYQX7bWrqgeaAARg8q2GxCKR92MQCRjFMrjAuLi4Ek
16Uiu2HAm6GG5cwmCgU6tSdfM3GjTMdrG4DZgy8FFZrjx5F5f2hZB
16Uiu2HAmMQEEfVdwS9hju7FFAjQNuSe7XMbVr9vqTgYVvztaihsF
16Uiu2HAm9dZXXnF1tfvamZKDanR21ShH8uD3qkVF3nkeqdVsftgW
16Uiu2HAmKW8mYXFQiFTsXecAXLJpXTbmZLB54Yz4sBsbC4NdVaiu
16Uiu2HAmMviqrSwQPmyi8Sbmur5cCs9AikpcWvRArtgKpiy9Nj9n
16Uiu2HAmJikc3z18eHjzFGTuKenNu3DVhPbNgQiiBWPA1yanxprk
16Uiu2HAmLRyfXqS8223xhFAFyaeUoQHV6fpsK87iZWM4wJDETKFn
16Uiu2HAmQkpXr9HjhCrghbE6HYWY9ZVqeTM3kQoTo8Fdcz8tHAJ2
16Uiu2HAm5tM7ZUWT7Pq9DMEsMW5xuqZ7HhhqcEhDpdpyY8EeghPM
16Uiu2HAkzH2p4FHcnu7k3C71jrcsZqepkhsVNasm4SgVBdYxqoss
16Uiu2HAm2hZeTosuEvEEU8VyTSnXhS8omYG6p5bZoCsPXsTPWjwc
16Uiu2HAmEvTrsM9bt6HTFmC94Fg6iNewSgCeMwzgUG2vUfWMC9fq
16Uiu2HAmNtJ4SgZQ934ngnwBop9gafrwtEyBa2ZEawL5WvqpLKFS
16Uiu2HAmPMQvzvMcq2biRLQ2mz3p6vQ5Jgp8NCwwXPiG5VbhfFvG
16Uiu2HAmFVTvSyeshT11CjDFMzzyPvRvTBvievV5nzUCNbtwPkfo
16Uiu2HAmH6gQPZHq9Zf9xQhSHzfVQ3hQoW58dkBuR2jJd8pTwBbw
16Uiu2HAm7eKJMGaRnjzZrUHpR8GZevYnBgXiJocmG7GHjM56ebDk
16Uiu2HAm3BS21Mii9RhkTmc4z3csYH2NMjvbccNecm5z8P2KWZtG
16Uiu2HAmSCFbGWqA2rfR5e4MCbfYvdR1WMpCsdNvUrcRtzwggdpd
16Uiu2HAmKdAWfeTCeMXo17HAmowkKTnNUM8ycoqj7xzJFc42AW2E
16Uiu2HAmDm88zAtTbsCeskpmHsL8Rjv5qE31stvkUJEUSJCrX213
16Uiu2HAmF7xt54jqJtuPkg9yp8bvT1tZu5du1D4RGqA91A2rXzk3
16Uiu2HAmMZyBjmj9WAYWSdoktVX9UYthZ8myY2YRdp7cxViAGSuM
16Uiu2HAm4nNwRjwEJnMemoPaE2B7KrS57DeXq2VUVxBTbAaGx664
16Uiu2HAkwnCzwF7ozLuvaUcPHJv7mGpzkrqdiodzvQiW8cpgJYCj
16Uiu2HAm8mbaMWDZAyN5x9HotUY5etdvoi1iiePuaVFuWLjkfQnZ
16Uiu2HAkxcoZh7uNWvB7z3HJwDTQJDMXvuexMhrKN1f6ro7SoLUN
16Uiu2HAmMFxwXtCbo9CNWGKvW7Xi6PxQFbw6mpj2k8CrcXEAdjnF
16Uiu2HAmGVHmDCyargJVap1UMRbefDifooMpU7RMkj1YEicLkBSt
16Uiu2HAmAZMMfYdcyq1GHa8VBCdDxjmdp7953BHyGiS3tKgaM6cr
16Uiu2HAmD2T5hUkzf8juKXhXxNWbavqSD6m7tcKwMaEgxdXsSBET
16Uiu2HAkyN6nmD6F5Mzf354YeXKxpvRc7D7m1RF2v8vjk3VurpBY
`

func MakePeerInfo() []*peerstore.PeerInfo {
	scanner := bufio.NewScanner(strings.NewReader(IDS))
	i := 0
	peerInfos := make([]*peerstore.PeerInfo, 0, 100)
	for scanner.Scan() {
		addr, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", i))
		id, _ := peer.IDB58Decode(scanner.Text())
		info := peerstore.PeerInfo{ID: id, Addrs: []ma.Multiaddr{addr}}
		// fmt.Println(info)
		peerInfos = append(peerInfos, &info)
		i++
	}
	return peerInfos
}

func MakeIds() {
	for i := 0; i < 100; i++ {
		_, pub, _ := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
		id, _ := peer.IDFromPublicKey(pub)
		fmt.Println(id.Pretty()) // = fmt.Println(peer.IDB58Encode(id))
		//fmt.Println(id) => <peer.ID 16*GXkYQW>
		// //decoding
		// id2, _ := peer.IDB58Decode(id.Pretty())
		// fmt.Println(id2.Pretty())
	}
}

// type TestNode struct {
// }

// func (node *TestNode) Connect(id peer.ID, addr ma.Multiaddr) (*PeerStream, error) {
// 	return nil, nil
// }

func TestLookup(t *testing.T) {
	peerInfos := MakePeerInfo()

	index := 0
	_findnode := func(peerInfo *peerstore.PeerInfo, targetID peer.ID) []*peerstore.PeerInfo {
		if index == 100 {
			index = 0
		}
		index = index + 5
		//fmt.Println(index)
		return peerInfos[index-5 : index]
	}
	_bond := func(peerInfo *peerstore.PeerInfo) *peerstore.PeerInfo {
		return peerInfo
	}

	// d := NewDiscovery(peerInfos[0].ID, peerstore.NewMetrics(), pstoremem.NewPeerstore())
	d := &Discovery{}
	//100, prevent peer evicted
	d.routingTable = kb.NewRoutingTable(100, kb.ConvertPeerID(peerInfos[0].ID), time.Minute, peerstore.NewMetrics())
	d.peerstore = pstoremem.NewPeerstore()

	d._findnode = _findnode
	d._bond = _bond

	d.Update(peerInfos[1])

	d.lookup(peerInfos[1].ID)
	assert.Equal(t, 100, d.routingTable.Size())
}

func TestRandomPeerInfo(t *testing.T) {
	peerInfos := MakePeerInfo()
	//multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/8080")
	addr, _ := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/8080")
	d := NewDiscovery(peerInfos[0].ID, addr, peerstore.NewMetrics(), pstoremem.NewPeerstore(), NewPeerStreamPool(), nil)
	_, err := d.RandomPeerInfo()
	assert.Error(t, err)
	d.Update(peerInfos[1])
	_, err = d.RandomPeerInfo()
	assert.NoError(t, err)
}

func TestDistance(t *testing.T) {
	peerInfos := MakePeerInfo()
	ids := make([]peer.ID, 0, len(peerInfos))
	infos := make(map[peer.ID]*peerstore.PeerInfo)
	for i, id := range peerInfos {
		ids = append(ids, id.ID)
		infos[id.ID] = peerInfos[i]
	}
	fmt.Printf("%.8b\n", kb.ConvertPeerID(peerInfos[0].ID)[:10])
	fmt.Println("--------------------------")
	peers := kb.SortClosestPeers(ids[1:], kb.ConvertPeerID(ids[0]))

	assert.Equal(t, "16Uiu2HAmQkpXr9HjhCrghbE6HYWY9ZVqeTM3kQoTo8Fdcz8tHAJ2", peers[0].Pretty())
	assert.Equal(t, "16Uiu2HAm2Uf175v4GA49zzayW2AD79EkYVhJJeE4Cw5AeqR2Djob", peers[1].Pretty())
	fmt.Printf("%.8b\n", kb.ConvertPeerID(peers[0])[:10])
	fmt.Printf("%.8b\n", kb.ConvertPeerID(peers[1])[:10])

}
