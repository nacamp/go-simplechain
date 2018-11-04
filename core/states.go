package core

import (
	"bytes"
	"math/big"

	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/rlp"
	"github.com/najimmy/go-simplechain/storage"
	"github.com/najimmy/go-simplechain/trie"
)

type Account struct {
	Address common.Address
	Balance *big.Int
	// Root    common.Hash // Before trie put
}

type AccountState struct {
	Trie    *trie.Trie
	Storage storage.Storage
}

//FIXME: only storage, no state?
type TransactionState struct {
	Trie    *trie.Trie
	Storage storage.Storage
}

func (acc *Account) AddBalance(amount *big.Int) {
	if acc.Balance == nil {
		acc.Balance = new(big.Int).SetUint64(0)
	}
	acc.Balance.Add(acc.Balance, amount)
}

func (acc *Account) SubBalance(amount *big.Int) {
	if acc.Balance == nil {
		acc.Balance = new(big.Int).SetUint64(0)
	}
	acc.Balance.Sub(acc.Balance, amount)
}

func NewAccountState() (*AccountState, error) {
	storage, _ := storage.NewMemoryStorage()
	tr, err := trie.NewTrie(nil, storage, false)
	return &AccountState{
		Trie:    tr,
		Storage: storage,
	}, err
}

func (accs *AccountState) PutAccount(account *Account) (hash common.Hash) {
	encodedBytes, _ := rlp.EncodeToBytes(account)
	accs.Trie.Put(account.Address[:], encodedBytes)
	copy(hash[:], accs.Trie.RootHash())
	return hash
}

func (accs *AccountState) GetAccount(address common.Address) (account *Account) {
	decodedBytes, _ := accs.Trie.Get(address[:])
	rlp.NewStream(bytes.NewReader(decodedBytes), 0).Decode(&account)
	return account
}

func (txs *TransactionState) PutTransaction(tx *Transaction) (hash common.Hash) {
	encodedBytes, _ := rlp.EncodeToBytes(tx)
	txs.Trie.Put(tx.Hash[:], encodedBytes)
	copy(hash[:], txs.Trie.RootHash())
	return hash
}

func (txs *AccountState) GetTransaction(hash common.Hash) (tx *Transaction) {
	decodedBytes, _ := txs.Trie.Get(hash[:])
	rlp.NewStream(bytes.NewReader(decodedBytes), 0).Decode(&tx)
	return tx
}
