package core_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/crypto"
	"github.com/nacamp/go-simplechain/rlp"
	"github.com/nacamp/go-simplechain/tests"
	"github.com/stretchr/testify/assert"
)

func TestHash(t *testing.T) {
	h := core.Header{}
	h.ParentHash.SetBytes(crypto.Sha3b256([]byte("dummy test")))
	assert.Equal(t, "6151d993d53d37941297e3f3e31a26a7cdc1d5fb3efc4a5a25132cdd38e05b15", hex.EncodeToString(h.ParentHash[:]), "")
}

func TestRlp(t *testing.T) {
	h := core.Header{ParentHash: common.Hash{0x01, 0x02, 0x03}, Time: 1540854071} //big.NewInt(1540854071)
	block := core.Block{BaseBlock: core.BaseBlock{Header: &h}}
	// fmt.Printf("%#v\n", block)
	encodedBytes, _ := rlp.EncodeToBytes(block)
	// fmt.Printf("Encoded value value: %#v\n", encodedBytes)
	var block2 core.Block
	rlp.NewStream(bytes.NewReader(encodedBytes), 0).Decode(&block2)
	// fmt.Printf("%#v\n", block2)
	assert.Equal(t, block.Header.ParentHash, block2.Header.ParentHash, "")
	assert.Equal(t, block.Header.Time, block2.Header.Time, "")

}

func TestSignAndVerify(t *testing.T) {
	priv := crypto.ByteToPrivateKey(common.FromHex(tests.Keystore[tests.AddressHex0]))
	h := core.Header{Coinbase: common.HexToAddress(tests.AddressHex0), ParentHash: common.Hash{0x01, 0x02, 0x03}, Time: 1540854071} //big.NewInt(1540854071)
	block := core.Block{BaseBlock: core.BaseBlock{Header: &h}}
	block.MakeHash()
	err := block.Sign(priv)
	assert.NoError(t, err, "")
	err = block.VerifySign()
	assert.NoError(t, err, "")

	block.Header.Coinbase = common.HexToAddress(tests.AddressHex1)
	err = block.VerifySign()
	assert.Error(t, err, "")

}
