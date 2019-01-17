package account

import (
	"encoding/gob"
	"errors"
	"os"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/crypto"
)

type wallet interface {
	Accounts()
}
type Wallet struct {
	wallet wallet
	keyStore
	keys     map[common.Address]*keyByte //map[common.Address]*Key //insecure
	filePath string
}

func NewWallet(filePath string) *Wallet {
	return &Wallet{
		filePath: filePath,
		keys:     make(map[common.Address]*keyByte),
	}
}

func (w *Wallet) StoreKey(key *Key, auth string) error {
	kb := &keyByte{Address: key.Address}
	authHash, err := crypto.HashPassword(auth)
	if err != nil {
		return err
	}

	_, cipherData, err := crypto.GcmEncrypt(crypto.PrivatekeyToByte(key.PrivateKey), authHash, true)
	kb.PrivateKey = cipherData

	file, err := os.Create(w.filePath)
	defer file.Close()

	if err == nil {
		encoder := gob.NewEncoder(file)
		w.keys[key.Address] = kb
		encoder.Encode(w.keys)
	}
	return err
}

func (w *Wallet) GetKey(address common.Address, auth string) (key *Key, err error) {
	authHash, err := crypto.HashPassword(auth)
	if err != nil {
		return nil, err
	}
	key = new(Key)
	if k, ok := w.keys[address]; ok {
		plainData, err := crypto.GcmDecrypt(nil, k.PrivateKey, authHash)
		if err != nil {
			return nil, err
		}
		key.Address = address
		key.PrivateKey = crypto.ByteToPrivatekey(plainData)
		return key, nil
	}

	file, err := os.Open(w.filePath)
	keys := make(map[common.Address]*keyByte)
	defer file.Close()
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(&keys)
		if err != nil {
			return nil, err
		}
	}
	w.keys = keys
	if k, ok := w.keys[address]; ok {
		plainData, err := crypto.GcmDecrypt(nil, k.PrivateKey, authHash)
		if err != nil {
			return nil, err
		}
		key.Address = address
		key.PrivateKey = crypto.ByteToPrivatekey(plainData)
		return key, nil
	}
	return nil, errors.New("not founded")
}
