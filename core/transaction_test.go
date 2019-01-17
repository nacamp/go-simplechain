package core_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/crypto"
)

func TestSign(t *testing.T) {
	/*
			address: 0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44
			priv:  	 0xd7573bb27684e1911b5e8bfb3a553f860ce873562e64016fec0974a6163a5cff

			address: 0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2
			priv:    0x47661aa6cccada84454842404ec0cca83760254191232f1d4cc11653d397ac2e
	*/
	//test same key
	priv := crypto.ByteToPrivatekey(common.FromHex("0xd7573bb27684e1911b5e8bfb3a553f860ce873562e64016fec0974a6163a5cff"))
	from := common.HexToAddress("0xd182458d4f299f73f496b7025912b0688653dbef74bc98638cd73e7e9ca01f8e9d416e44")
	to := common.HexToAddress("0xfdf75c884f7f1d1537177a3a35e783236739a426ee649fa3e2d8aed598b4f29e838170e2")
	tx := core.NewTransaction(from, to, new(big.Int).SetInt64(100), uint64(0))
	tx.MakeHash()
	tx.Sign(priv)
	sig, _ := tx.VerifySign()
	assert.True(t, sig, "")
}
