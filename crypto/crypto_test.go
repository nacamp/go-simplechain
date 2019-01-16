package crypto

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/nacamp/go-simplechain/common"
	"golang.org/x/crypto/sha3"

	// "github.com/btcsuite/btcd/btcec"
	"github.com/stretchr/testify/assert"
)

func TestHash(t *testing.T) {
	assert.Equal(t, hex.EncodeToString(Sha3b256([]byte("dummy test"))), "6151d993d53d37941297e3f3e31a26a7cdc1d5fb3efc4a5a25132cdd38e05b15", "test sha3-256")
}

// func TestAddress(t *testing.T) {
// 	priv, err := btcec.NewPrivateKey(btcec.S256())
// 	if err != nil {
// 		return
// 	}
// 	/*
// 		priv/pub
// 		0xe68fb0a479c495910c8351c3593667028b45d679f55ce22b0514c4a8a6bcbdd1 / 0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0
// 		0xf390e256b6ed8a1b283d3ea80b103b868c14c31e5b7114fc32fff21c4cb263eb / 0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3
// 		0xb385aca81e134722cca902bf85443528c3d3a783cf54008cfc34a2ca563fc5b6 / 0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1
// 	*/
// 	pubkey := priv.PubKey()
// 	address := common.BytesToAddress(pubkey.SerializeCompressed())
// 	// fmt.Println(common.ToHex(priv.Serialize()))
// 	// fmt.Println(common.ToHex(pubkey.SerializeCompressed()))
// 	//fmt.Println(common.ToHex(pubkey.SerializeUncompressed()))
// 	assert.Equal(t, pubkey.SerializeCompressed(), address[:], "")
// }

func TestCreateAndEcrecover(t *testing.T) {
	priv, address := CreateAddress()
	fmt.Println(common.Address2Hex(address))
	assert.Equal(t, CreateAddressFromPrivatekey(priv), address, "")

	hash := make([]byte, 32)
	hasher := sha3.New256()
	k := []byte("data...")
	hasher.Write(k)
	hash = hasher.Sum(nil)
	// fmt.Printf("%#v\n", hash)

	signed, _ := Sign(hash, priv)
	pub, _ := Ecrecover(hash, signed)
	// fmt.Printf("%v\n", common.BytesToAddress(pub))
	// fmt.Println(common.Address2Hex(common.BytesToAddress(pub)))
	assert.Equal(t, address, CreateAddressFromPublickeyByte(pub))

	hasher.Write([]byte("."))
	hash = hasher.Sum(nil)
	pub, _ = Ecrecover(hash, signed)
	assert.NotEqual(t, address, CreateAddressFromPublickeyByte(pub))
}

func TestValidateAddress(t *testing.T) {
	_, address := CreateAddress()
	assert.True(t, ValidateAddress(address))

	address[0] += 0x01
	assert.False(t, ValidateAddress(address))
}
