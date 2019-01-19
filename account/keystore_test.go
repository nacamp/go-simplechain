package account

import (
	"fmt"
	"os"
	"testing"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/crypto"
	"github.com/stretchr/testify/assert"
)

func TestNewKey(t *testing.T) {
	key := NewKey()
	fmt.Printf("%v", key)
}

func TestStoreAndGet(t *testing.T) {
	path := "./test.dat"
	defer os.Remove(path)

	w := NewWallet(path)
	key := NewKey()
	w.StoreKey(key, "test")

	//different password
	key2, err := w.GetKey(key.Address, "test1")
	assert.Error(t, err)

	key2, err = w.GetKey(key.Address, "test")
	assert.Equal(t, key.Address, key2.Address)
	assert.Equal(t, crypto.PrivateKeyToByte(key.PrivateKey), crypto.PrivateKeyToByte(key2.PrivateKey))

	key = NewKey()
	w.StoreKey(key, "test")
	key2, _ = w.GetKey(key.Address, "test")
	assert.Equal(t, key.Address, key2.Address)
	assert.Equal(t, crypto.PrivateKeyToByte(key.PrivateKey), crypto.PrivateKeyToByte(key2.PrivateKey))

	assert.Equal(t, 2, len(w.keys))

	//remove key map and check reading address from file
	w.keys = make(map[common.Address]*keyByte)
	key2, _ = w.GetKey(key.Address, "test")
	assert.Equal(t, key.Address, key2.Address)
	assert.Equal(t, crypto.PrivateKeyToByte(key.PrivateKey), crypto.PrivateKeyToByte(key2.PrivateKey))

	//check reading address from map
	os.Remove(path)
	key2, _ = w.GetKey(key.Address, "test")
	assert.Equal(t, key.Address, key2.Address)
	assert.Equal(t, crypto.PrivateKeyToByte(key.PrivateKey), crypto.PrivateKeyToByte(key2.PrivateKey))
}
