package account

import (
	"fmt"
	"os"
	"testing"
	"time"

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

func TestUnlock(t *testing.T) {
	path := "./test.dat"
	defer os.Remove(path)

	w := NewWallet(path)
	key := NewKey()
	w.StoreKey(key, "test")

	err := w.TimedUnlock(key.Address, "test", 1*time.Millisecond)
	if err != nil {
		fmt.Println(err.Error())
	}
	assert.Equal(t, 1, len(w.unlockKeys))
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 0, len(w.unlockKeys))
}

func TestGet(t *testing.T) {
	/*
		// ./simple account new -config ../../conf/node6.json
		var AddressHex = string("0x1a8dd828a43acdcd9f1286ab437b91e43482bd5dd7a92a2631671554f5179b40d21e46a9")

		path := "/Users/jimmy/go/src/github.com/nacamp/data/keystore4.dat"
		w := NewWallet(path)
		key, err := w.GetKey(common.HexToAddress(AddressHex), "password")
		fmt.Println(err)
		fmt.Println(common.BytesToHex(crypto.PrivateKeyToByte(key.PrivateKey)))
	*/
}
