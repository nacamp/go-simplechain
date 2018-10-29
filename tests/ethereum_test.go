package tests

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/core"
	"github.com/stretchr/testify/assert"
)

func TestRlp(t *testing.T) {
	//https://godoc.org/github.com/ethereum/go-ethereum/rlp#example-Encoder
	header := core.Header{ParentHash: common.Hash{0x01, 0x02, 0x03}, Time: big.NewInt(1540854071)}
	encodedBytes, _ := rlp.EncodeToBytes(header)
	//fmt.Printf("Encoded value value: %#v\n", encodedBytes)

	var header2 core.Header
	rlp.Decode(bytes.NewReader(encodedBytes), &header2)
	//fmt.Printf("Decoded value: %#v\n", header2)
	assert.Equal(t, header.ParentHash, header2.ParentHash, "Test ParentHash")
	assert.Equal(t, header.Time, header2.Time, "Test Time")
}
