package core

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/rlp"
	"github.com/nacamp/go-simplechain/storage"
	"github.com/nacamp/go-simplechain/trie"
	"github.com/sirupsen/logrus"
)

var (
	ErrTransactionNonce        = errors.New("cannot accept a transaction with wrong nonce")
	ErrBalanceInsufficient     = errors.New("cannot subtract a value which is bigger than current balance")
	ErrAccountNotFound         = errors.New("cannot found account in storage")
	ErrContractAccountNotFound = errors.New("cannot found contract account in storage please check contract address is valid or deploy is success")
)

/*
The stake amount that you voted for in the minors is added up in the next round to elect a new minor and
can't be withdrawn to next round after the minors have been elected.
*/
type Account struct {
	Address          common.Address
	Balance          *big.Int
	Nonce            uint64
	Staking          map[common.Address]*big.Int
	TotalPeggedStake *big.Int //Non-withdrawable stake
}

type BasicAccount struct {
	Address common.Address
	Balance *big.Int
}

type rlpAccount struct {
	Address common.Address
	Balance *big.Int
	Nonce   uint64

	Staking          []BasicAccount
	TotalPeggedStake *big.Int
}

type AccountState struct {
	Trie *trie.Trie
}

type TransactionState struct {
	Trie *trie.Trie
}

func NewAccount() *Account {
	return &Account{
		Balance:          new(big.Int).SetUint64(0),
		Staking:          make(map[common.Address]*big.Int),
		TotalPeggedStake: new(big.Int).SetUint64(0),
	}
}

func (acc *Account) AddBalance(amount *big.Int) {
	// if acc.Balance == nil {
	// 	acc.Balance = new(big.Int).SetUint64(0)
	// }
	acc.Balance.Add(acc.Balance, amount)
}

func (acc *Account) SubBalance(amount *big.Int) error {
	// if acc.Balance == nil {
	// 	return ErrBalanceInsufficient
	// }
	if acc.Balance.Cmp(amount) < 0 {
		return ErrBalanceInsufficient
	}
	acc.Balance.Sub(acc.Balance, amount)
	return nil
}

func (acc *Account) AvailableBalance() *big.Int {
	tot := new(big.Int)
	for _, v := range acc.Staking {
		tot.Add(tot, v)
	}
	if tot.Cmp(acc.TotalPeggedStake) > 0 {
		return tot.Sub(acc.Balance, tot)
	}
	return tot.Sub(acc.Balance, acc.TotalPeggedStake)
}

func (acc *Account) TotalStaking() *big.Int {
	tot := new(big.Int)
	for _, v := range acc.Staking {
		tot.Add(tot, v)
	}
	return tot
}

func (acc *Account) Stake(address common.Address, amount *big.Int) error {
	tmp := new(big.Int)
	//acc.Balance - acc.TotalStaking <  amount
	if tmp.Sub(acc.Balance, acc.TotalStaking()).Cmp(amount) < 0 {
		return errors.New("There is insufficient stake.")
	}

	v, ok := acc.Staking[address]
	if ok {
		v.Add(v, amount)
	} else {
		acc.Staking[address] = amount
	}
	return nil
}

func (acc *Account) UnStake(address common.Address, amount *big.Int) (err error) {
	v, ok := acc.Staking[address]
	if !ok {
		return errors.New("There is insufficient stake.")
	}
	if v.Cmp(amount) < 0 {
		return errors.New("There is insufficient stake.")
	}
	v.Sub(v, amount)
	return nil
}

func (acc *Account) CalcSetTotalPeggedStake() {
	acc.TotalPeggedStake = acc.TotalStaking()
}

func NewAccountState(storage storage.Storage) (*AccountState, error) {
	tr, err := trie.NewTrie(nil, storage, false)
	return &AccountState{
		Trie: tr,
	}, err
}

func NewAccountStateRootHash(rootHash common.Hash, storage storage.Storage) (*AccountState, error) {
	tr, err := trie.NewTrie(rootHash[:], storage, false)
	return &AccountState{
		Trie: tr,
	}, err
}

func (accs *AccountState) Clone() (*AccountState, error) {
	tr, err := accs.Trie.Clone()
	return &AccountState{
		Trie: tr,
	}, err
}

func (accs *AccountState) PutAccount(account *Account) (hash common.Hash) {
	rlpAcc := rlpAccount{
		Address:          account.Address,
		Balance:          account.Balance,
		Nonce:            account.Nonce,
		Staking:          make([]BasicAccount, 0),
		TotalPeggedStake: account.TotalPeggedStake,
	}
	for k, v := range account.Staking {
		rlpAcc.Staking = append(rlpAcc.Staking, BasicAccount{Address: k, Balance: v})
	}

	encodedBytes, err := rlp.EncodeToBytes(rlpAcc)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	accs.Trie.Put(account.Address[:], encodedBytes)
	copy(hash[:], accs.Trie.RootHash())
	return hash
}

func (accs *AccountState) GetAccount(address common.Address) (account *Account) {
	decodedBytes, err := accs.Trie.Get(address[:])
	if err == nil {
		rlpAcc := new(rlpAccount)
		rlp.NewStream(bytes.NewReader(decodedBytes), 0).Decode(rlpAcc)
		account := Account{
			Address:          rlpAcc.Address,
			Balance:          rlpAcc.Balance,
			Nonce:            rlpAcc.Nonce,
			Staking:          make(map[common.Address]*big.Int),
			TotalPeggedStake: rlpAcc.TotalPeggedStake,
		}
		for _, v := range rlpAcc.Staking {
			account.Staking[v.Address] = v.Balance
		}
		return &account
	} else {
		if err == storage.ErrKeyNotFound {
			account := NewAccount()
			account.Address = address
			return account
		}
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
		return nil
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
	encodedBytes, err := rlp.EncodeToBytes(tx)
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	txs.Trie.Put(tx.Hash[:], encodedBytes)
	copy(hash[:], txs.Trie.RootHash())
	return hash
}

func (txs *TransactionState) GetTransaction(hash common.Hash) (tx *Transaction) {
	tx = new(Transaction)
	encodedBytes, err := txs.Trie.Get(hash[:])
	if err != nil {
		log.CLog().WithFields(logrus.Fields{}).Panic(err)
	}
	rlp.NewStream(bytes.NewReader(encodedBytes), 0).Decode(&tx)
	return tx
}

func (txs *TransactionState) RootHash() (hash common.Hash) {
	copy(hash[:], txs.Trie.RootHash())
	return hash
}
