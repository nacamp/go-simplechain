package core_test

import (
	"math/big"
	"testing"

	"github.com/nacamp/go-simplechain/tests"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/storage"
	"github.com/stretchr/testify/assert"
)

func TestAccount(t *testing.T) {
	account := core.NewAccount()
	account.AddBalance(new(big.Int).SetUint64(10))
	assert.Equal(t, new(big.Int).SetUint64(10), account.Balance, "")
	account.SubBalance(new(big.Int).SetUint64(5))
	amount := new(big.Int).SetUint64(5)
	assert.Equal(t, amount, account.Balance, "")
}

func TestAccountState(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		return
	}
	accountState, _ := core.NewAccountState(storage)
	account := core.NewAccount()
	account.Address = common.HexToAddress(tests.Addr0)
	account.AddBalance(new(big.Int).SetUint64(10))
	accountState.PutAccount(account)

	account2 := accountState.GetAccount(account.Address)
	assert.Equal(t, account.Address, account2.Address, "")
	assert.Equal(t, new(big.Int).SetUint64(10), account2.Balance, "")
}

func TestAccountStake(t *testing.T) {
	var err error
	account := core.NewAccount()
	account.AddBalance(new(big.Int).SetUint64(10))
	assert.Equal(t, new(big.Int).SetUint64(10), account.AvailableBalance(), "")
	assert.Equal(t, new(big.Int).SetUint64(0), account.TotalStaking(), "")

	// 8 = 10 -2
	account.Stake(common.HexToAddress(tests.Addr0), new(big.Int).SetUint64(2))
	assert.Equal(t, new(big.Int).SetUint64(8), account.AvailableBalance(), "")
	assert.Equal(t, new(big.Int).SetUint64(2), account.TotalStaking(), "")
	err = account.Stake(common.HexToAddress(tests.Addr0), new(big.Int).SetUint64(9))
	assert.Error(t, err)

	//Unstaking
	err = account.UnStake(common.HexToAddress(tests.Addr1), new(big.Int).SetUint64(2))
	assert.Error(t, err)
	err = account.UnStake(common.HexToAddress(tests.Addr0), new(big.Int).SetUint64(3))
	assert.Error(t, err)
	err = account.UnStake(common.HexToAddress(tests.Addr0), new(big.Int).SetUint64(2))
	assert.NoError(t, err)
	assert.Equal(t, new(big.Int).SetUint64(10), account.AvailableBalance(), "")
	assert.Equal(t, new(big.Int).SetUint64(0), account.TotalStaking(), "")

	//TotalStaking
	account.Stake(common.HexToAddress(tests.Addr0), new(big.Int).SetUint64(3))
	account.TotalPeggedStake = new(big.Int).SetUint64(3)
	account.Stake(common.HexToAddress(tests.Addr1), new(big.Int).SetUint64(5))
	assert.Equal(t, new(big.Int).SetUint64(2), account.AvailableBalance(), "")
	err = account.UnStake(common.HexToAddress(tests.Addr0), new(big.Int).SetUint64(3))
	err = account.UnStake(common.HexToAddress(tests.Addr1), new(big.Int).SetUint64(5))
	assert.Equal(t, new(big.Int).SetUint64(0), account.TotalStaking(), "")     // non include TotalPeggedStake
	assert.Equal(t, new(big.Int).SetUint64(7), account.AvailableBalance(), "") // 10 -3(TotalPeggedStake if TotalStaking() < TotalPeggedStake )
}

func TestAccountStateStake(t *testing.T) {
	storage, err := storage.NewMemoryStorage()
	if err != nil {
		return
	}
	accountState, _ := core.NewAccountState(storage)
	account := core.NewAccount()
	account.Address = common.HexToAddress(tests.Addr0)
	account.AddBalance(new(big.Int).SetUint64(10))
	account.Stake(common.HexToAddress(tests.Addr0), new(big.Int).SetUint64(3))
	account.TotalPeggedStake = new(big.Int).SetUint64(3)
	accountState.PutAccount(account)

	account2 := accountState.GetAccount(account.Address)
	assert.Equal(t, account.Address, account2.Address, "")
	assert.Equal(t, new(big.Int).SetUint64(10), account2.Balance, "")
	assert.Equal(t, new(big.Int).SetUint64(3), account2.Staking[common.HexToAddress(tests.Addr0)], "")
	assert.Equal(t, new(big.Int).SetUint64(3), account2.TotalPeggedStake, "")
}
