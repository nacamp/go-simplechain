package account

import (
	"encoding/gob"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/crypto"
)

type wallet interface {
	Accounts()
}
type Wallet struct {
	wallet wallet
	keyStore
	unlockKeys map[common.Address]*Key     //insecure
	keys       map[common.Address]*keyByte //map[common.Address]*Key //insecure
	filePath   string
	mu         sync.RWMutex
}

func NewWallet(filePath string) *Wallet {
	return &Wallet{
		filePath:   filePath,
		keys:       make(map[common.Address]*keyByte),
		unlockKeys: make(map[common.Address]*Key),
	}
}

func (w *Wallet) StoreKey(key *Key, auth string) error {
	kb := &keyByte{Address: key.Address}
	authHash, err := crypto.HashPassword(auth)
	if err != nil {
		return err
	}

	_, cipherData, err := crypto.GcmEncrypt(crypto.PrivateKeyToByte(key.PrivateKey), authHash, true)
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
		key.PrivateKey = crypto.ByteToPrivateKey(plainData)
		return key, nil
	}
	err = w.Load()
	if err != nil {
		return nil, err
	}
	if k, ok := w.keys[address]; ok {
		plainData, err := crypto.GcmDecrypt(nil, k.PrivateKey, authHash)
		if err != nil {
			return nil, err
		}
		key.Address = address
		key.PrivateKey = crypto.ByteToPrivateKey(plainData)
		return key, nil
	}
	return nil, errors.New("not founded")
}

func (w *Wallet) Load() error {
	file, err := os.Open(w.filePath)
	keys := make(map[common.Address]*keyByte)
	defer file.Close()
	if err != nil {
		return err
	}
	decoder := gob.NewDecoder(file)
	err = decoder.Decode(&keys)
	if err != nil {
		return err
	}
	w.keys = keys
	return nil
}

func (w *Wallet) TimedUnlock(address common.Address, auth string, timeout time.Duration) error {
	key, err := w.GetKey(address, auth)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.unlockKeys[address] = key
	if timeout > 0 {
		go w.expire(address, key, timeout)
	}
	return nil
}

func (w *Wallet) expire(addr common.Address, u *Key, timeout time.Duration) {
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case <-t.C:
		w.mu.Lock()
		if w.unlockKeys[addr] == u {
			//zeroKey(u.PrivateKey)
			delete(w.unlockKeys, addr)
		}
		w.mu.Unlock()
	}
}

func (w *Wallet) SignHash(addr common.Address, hash []byte) ([]byte, error) {
	// Look up the key to sign with and abort if it cannot be found
	w.mu.RLock()
	defer w.mu.RUnlock()

	unlockedKey, found := w.unlockKeys[addr]
	if !found {
		return nil, errors.New("not founded")
	}
	// Sign the hash using plain ECDSA operations
	return crypto.Sign(hash, unlockedKey.PrivateKey)
}
