package core

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/rlp"
	"github.com/najimmy/go-simplechain/storage"
	"github.com/najimmy/go-simplechain/trie"
)

var (
	ErrBalanceInsufficient     = errors.New("cannot subtract a value which is bigger than current balance")
	ErrAccountNotFound         = errors.New("cannot found account in storage")
	ErrContractAccountNotFound = errors.New("cannot found contract account in storage please check contract address is valid or deploy is success")
)

type Account struct {
	Address common.Address
	Balance *big.Int
	// Root    common.Hash // Before trie put
}

type AccountState struct {
	Trie *trie.Trie
	// Storage storage.Storage //FIMXE: current not use
}

//no state, but need merkle root
type TransactionState struct {
	Trie *trie.Trie
	// Storage storage.Storage //FIMXE: current not use
}

func (acc *Account) AddBalance(amount *big.Int) {
	if acc.Balance == nil {
		acc.Balance = new(big.Int).SetUint64(0)
	}
	acc.Balance.Add(acc.Balance, amount)
}

func (acc *Account) SubBalance(amount *big.Int) error {

	if acc.Balance == nil {
		return ErrBalanceInsufficient
	}
	if acc.Balance.Cmp(amount) < 0 {
		return ErrBalanceInsufficient
	}
	acc.Balance.Sub(acc.Balance, amount)
	return nil
}

func NewAccountState(storage storage.Storage) (*AccountState, error) {
	// storage, _ := storage.NewMemoryStorage()
	tr, err := trie.NewTrie(nil, storage, false)
	return &AccountState{
		Trie: tr,
		// Storage: storage,
	}, err
}

func NewAccountStateRootHash(rootHash common.Hash, storage storage.Storage) (*AccountState, error) {
	tr, err := trie.NewTrie(rootHash[:], storage, false)
	return &AccountState{
		Trie: tr,
		// Storage: storage,
	}, err
}

func (accs *AccountState) Clone() (*AccountState, error) {
	// storage, _ := storage.NewMemoryStorage()
	tr, err := accs.Trie.Clone()
	return &AccountState{
		Trie: tr,
	}, err
}

//TODO: error
func (accs *AccountState) PutAccount(account *Account) (hash common.Hash) {
	encodedBytes, _ := rlp.EncodeToBytes(account)
	accs.Trie.Put(account.Address[:], encodedBytes)
	copy(hash[:], accs.Trie.RootHash())
	return hash
}

//TODO: error
func (accs *AccountState) GetAccount(address common.Address) (account *Account) {
	decodedBytes, err := accs.Trie.Get(address[:])
	//FIXME: TOBE
	// if err != nil && err != storage.ErrKeyNotFound {
	// 	return nil, err
	// }
	if err == nil {
		rlp.NewStream(bytes.NewReader(decodedBytes), 0).Decode(&account)
		return account
	} else {
		return &Account{Address: address, Balance: new(big.Int).SetUint64(0)}
	}

}

func (accs *AccountState) RootHash() (hash common.Hash) {
	copy(hash[:], accs.Trie.RootHash())
	return hash
}

//-------------------- TransactionState
func NewTransactionState(storage storage.Storage) (*TransactionState, error) {
	//TODO: how to do
	//return NewTransactionStateRootHash(nil, storage)
	tr, err := trie.NewTrie(nil, storage, false)
	return &TransactionState{
		Trie: tr,
		// Storage: storage,
	}, err
}

func NewTransactionStateRootHash(rootHash common.Hash, storage storage.Storage) (*TransactionState, error) {
	tr, err := trie.NewTrie(rootHash[:], storage, false)
	return &TransactionState{
		Trie: tr,
		// Storage: storage,
	}, err
}

func (txs *TransactionState) Clone() (*TransactionState, error) {
	// storage, _ := storage.NewMemoryStorage()
	tr, err := txs.Trie.Clone()
	return &TransactionState{
		Trie: tr,
	}, err
}

func (txs *TransactionState) PutTransaction(tx *Transaction) (hash common.Hash) {
	encodedBytes, _ := rlp.EncodeToBytes(tx)
	txs.Trie.Put(tx.Hash[:], encodedBytes)
	copy(hash[:], txs.Trie.RootHash())
	return hash
}

func (txs *TransactionState) GetTransaction(hash common.Hash) (tx *Transaction) {
	decodedBytes, _ := txs.Trie.Get(hash[:])
	rlp.NewStream(bytes.NewReader(decodedBytes), 0).Decode(&tx)
	return tx
}

func (txs *TransactionState) RootHash() (hash common.Hash) {
	copy(hash[:], txs.Trie.RootHash())
	return hash
}
