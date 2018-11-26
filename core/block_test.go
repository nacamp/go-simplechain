package core_test

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/core"
	"github.com/najimmy/go-simplechain/crypto"
	"github.com/najimmy/go-simplechain/rlp"
	"github.com/najimmy/go-simplechain/tests"
	"github.com/stretchr/testify/assert"
)

func TestHash(t *testing.T) {
	h := core.Header{}
	h.ParentHash.SetBytes(crypto.Sha3b256([]byte("dummy test")))
	assert.Equal(t, "6151d993d53d37941297e3f3e31a26a7cdc1d5fb3efc4a5a25132cdd38e05b15", hex.EncodeToString(h.ParentHash[:]), "")
}

func TestRlp(t *testing.T) {
	h := core.Header{ParentHash: common.Hash{0x01, 0x02, 0x03}, Time: 1540854071} //big.NewInt(1540854071)
	block := core.Block{Header: &h}
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
	priv, _ := btcec.PrivKeyFromBytes(btcec.S256(), common.FromHex(tests.Keystore[tests.Addr0]))
	h := core.Header{ParentHash: common.Hash{0x01, 0x02, 0x03}, Time: 1540854071} //big.NewInt(1540854071)
	block := core.Block{Header: &h}
	block.Sign((*ecdsa.PrivateKey)(priv))
	b, err := block.VerifySign()
	assert.True(t, b, "")
	assert.NoError(t, err, "")

	block.Header.Coinbase = common.HexToAddress(tests.Addr1)
	b, err = block.VerifySign()
	assert.False(t, b, "")
	assert.Error(t, err, "")

}
