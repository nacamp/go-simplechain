package core_test

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/btcsuite/btcd/btcec"
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/core"
)

func TestSign(t *testing.T) {
	/*
		priv/pub
		0xe68fb0a479c495910c8351c3593667028b45d679f55ce22b0514c4a8a6bcbdd1 / 0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0
		0xf390e256b6ed8a1b283d3ea80b103b868c14c31e5b7114fc32fff21c4cb263eb / 0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3
	*/
	//test same key
	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), common.Hex2Bytes("e68fb0a479c495910c8351c3593667028b45d679f55ce22b0514c4a8a6bcbdd1"))
	assert.Equal(t, common.Hex2Bytes("036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0"), pub.SerializeCompressed(), "")
	assert.Equal(t, common.Hex2Bytes("e68fb0a479c495910c8351c3593667028b45d679f55ce22b0514c4a8a6bcbdd1"), priv.Serialize(), "")

	from := common.BytesToAddress(common.Hex2Bytes("036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0"))
	to := common.BytesToAddress(common.Hex2Bytes("03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3"))
	tx := core.NewTransaction(from, to, new(big.Int).SetInt64(100))
	tx.MakeHash()
	tx.Sign((*ecdsa.PrivateKey)(priv))
	sig, _ := tx.VerifySign()
	assert.True(t, sig, "")
}
