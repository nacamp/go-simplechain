package tests

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	peer "github.com/libp2p/go-libp2p-peer"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/net"
	"github.com/nacamp/go-simplechain/rlp"
	"github.com/stretchr/testify/assert"
)

func TestBigint(t *testing.T) {
	i := new(big.Int).SetUint64(10)
	encodedBytes, err := rlp.EncodeToBytes(i)
	if err != nil {
		fmt.Printf("%#v\n", err)
	}
	i2 := new(big.Int)
	rlp.Decode(bytes.NewReader(encodedBytes), i2)
	assert.Equal(t, i, i2)
	fmt.Println(i, i2)
}

func TestPeerInfo(t *testing.T) {
	addr, _ := ma.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/100"))
	id, _ := peer.IDB58Decode("16Uiu2HAkyN6nmD6F5Mzf354YeXKxpvRc7D7m1RF2v8vjk3VurpBY")
	info := peerstore.PeerInfo{ID: id, Addrs: []ma.Multiaddr{addr}}
	encodedBytes, err := rlp.EncodeToBytes(net.ToPeerInfo2(&info))
	if err != nil {
		fmt.Printf("%#v\n", err)
	}
	//Multiaddr is interface, rlp not support interface
	info2 := net.PeerInfo2{}
	rlp.Decode(bytes.NewReader(encodedBytes), &info2)
	assert.Equal(t, &info, net.FromPeerInfo2(&info2), "")
}

type T1 struct {
	T1_Text string
}

type T2 struct {
	T2_Text string
	T1
}

func TestMixtureType(t *testing.T) {
	tt1 := T2{
		T2_Text: "Test",
		T1:      T1{T1_Text: "Test"},
	}
	encodedBytes, err := rlp.EncodeToBytes(tt1)
	if err != nil {
		fmt.Printf("%#v\n", err)
	}

	tt2 := &T2{}
	rlp.Decode(bytes.NewReader(encodedBytes), &tt2)
	assert.Equal(t, tt1.T2_Text, tt2.T2_Text, "")
}

func TestRlpMapSlice(t *testing.T) {
	m := make(map[string]string)
	m["key0"] = "val0"
	encodedBytes, err := rlp.EncodeToBytes(m)
	assert.Error(t, err, "xxx is not RLP-serializable")

	s := make([][]string, 3)
	s[0] = make([]string, 2)
	s[0][0] = "key0"
	s[0][1] = "val0"
	s[1] = []string{"key1", "val2"}

	encodedBytes, err = rlp.EncodeToBytes(s)
	assert.NoError(t, err, "xxx is not RLP-serializable")

	s2 := make([][]string, 3)
	rlp.Decode(bytes.NewReader(encodedBytes), &s2)
	// fmt.Printf("%#v", s2)

	assert.Equal(t, s[0], s2[0], "")
	assert.Equal(t, s[1], s2[1], "")
}

func TestMessage(t *testing.T) {
	msg, _ := net.NewRLPMessage(1, "test")
	encodedBytes, _ := rlp.EncodeToBytes(msg)

	msg2 := net.Message{}
	rlp.Decode(bytes.NewReader(encodedBytes), &msg2)
	assert.Equal(t, msg.Code, msg2.Code, "")

	str := string("")
	rlp.DecodeBytes(msg2.Payload, &str)
	assert.Equal(t, "test", str, "")
}
func TestRlp(t *testing.T) {
	//https://godoc.org/github.com/ethereum/go-ethereum/rlp#example-Encoder
	header := core.Header{ParentHash: common.Hash{0x01, 0x02, 0x03}, Time: uint64(1540854071)}
	encodedBytes, _ := rlp.EncodeToBytes(header)
	//fmt.Printf("Encoded value value: %#v\n", encodedBytes)

	var header2 core.Header
	rlp.Decode(bytes.NewReader(encodedBytes), &header2)
	//fmt.Printf("Decoded value: %#v\n", header2)
	assert.Equal(t, header.ParentHash, header2.ParentHash, "Test ParentHash")
	assert.Equal(t, header.Time, header2.Time, "Test Time")

	header2 = core.Header{}
	rlp.NewStream(bytes.NewReader(encodedBytes), 0).Decode(&header2)
	// s:=rlp.NewStream(bytes.NewReader(encodedBytes), 0)
	// if _, err := s.List(); err != nil {
	// 	fmt.Printf("List error: %v\n", err)
	// 	return
	// }
	// s.Decode(&header2)
	assert.Equal(t, header.ParentHash, header2.ParentHash, "Test ParentHash")
	assert.Equal(t, header.Time, header2.Time, "Test Time")

	s := rlp.NewStream(bytes.NewReader(encodedBytes), 0)
	kind, size, _ := s.Kind()
	fmt.Printf("Kind: %v size:%d\n", kind, size)
	if _, err := s.List(); err != nil {
		fmt.Printf("List error: %v\n", err)
		return
	}
	kind, size, _ = s.Kind()
	fmt.Printf("Kind1: %v size:%d\n", kind, size)
	fmt.Println(s.Bytes())
	kind, size, _ = s.Kind()
	fmt.Printf("Kind2: %v size:%d\n", kind, size)
	fmt.Println(s.Bytes())
	kind, size, _ = s.Kind()
	fmt.Printf("Kind3: %v size:%d\n", kind, size)
	fmt.Println(s.Uint())
	kind, size, _ = s.Kind()
	fmt.Printf("Kind4: %v size:%d\n", kind, size)
	fmt.Println(s.Uint())
	if err := s.ListEnd(); err != nil {
		fmt.Printf("ListEnd error: %v\n", err)
	}
}
