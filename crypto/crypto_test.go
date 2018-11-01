package crypto_test

import (
	"encoding/hex"
	"testing"

	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/crypto"

	"github.com/btcsuite/btcd/btcec"
	"github.com/stretchr/testify/assert"
)

func TestHash(t *testing.T) {
	assert.Equal(t, hex.EncodeToString(crypto.Sha3b256([]byte("dummy test"))), "6151d993d53d37941297e3f3e31a26a7cdc1d5fb3efc4a5a25132cdd38e05b15", "test sha3-256")
}

func TestAddress(t *testing.T) {
	priv, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return
	}
	/*
		priv/pub
		0xe68fb0a479c495910c8351c3593667028b45d679f55ce22b0514c4a8a6bcbdd1 / 0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0
		0xf390e256b6ed8a1b283d3ea80b103b868c14c31e5b7114fc32fff21c4cb263eb / 0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3
	*/
	pubkey := priv.PubKey()
	address := common.BytesToAddress(pubkey.SerializeCompressed())
	assert.Equal(t, pubkey.SerializeCompressed(), address[:], "")
	//fmt.Println(common.ToHex(priv.Serialize()))
	//fmt.Println(common.ToHex(pubkey.SerializeCompressed()))
	//fmt.Println(common.ToHex(pubkey.SerializeUncompressed()))
}
