package core_test

import (
	"math/big"
	"testing"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/storage"
	"github.com/stretchr/testify/assert"
)

func TestAccount(t *testing.T) {
	account := core.Account{}
	account.AddBalance(new(big.Int).SetUint64(10))
	assert.Equal(t, new(big.Int).SetUint64(10), account.Balance, "")
	account.SubBalance(new(big.Int).SetUint64(5))
	amount := new(big.Int).SetUint64(5)
	assert.Equal(t, amount, account.Balance, "")
	account.SubBalance(new(big.Int).SetUint64(5))
}

func TestAccountState(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		return
	}
	accountState, _ := core.NewAccountState(storage)
	var hexAddress = "036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0"
	account := core.Account{}
	copy(account.Address[:], common.FromHex(hexAddress))
	account.AddBalance(new(big.Int).SetUint64(10))
	accountState.PutAccount(&account)

	account2 := accountState.GetAccount(account.Address)
	assert.Equal(t, account.Address, account2.Address, "")
	assert.Equal(t, new(big.Int).SetUint64(10), account2.Balance, "")
}
