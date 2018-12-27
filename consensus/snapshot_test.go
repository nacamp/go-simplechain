package consensus

import (
	"fmt"
	"testing"

	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/storage"
	"github.com/najimmy/go-simplechain/tests"
	"github.com/stretchr/testify/assert"
)

func TestNewAndLoadSnapshot(t *testing.T) {
	hash := common.BytesToHash([]byte{0x01, 0x02, 0x03})
	snap := NewSnapshot(hash, []common.Address{common.HexToAddress(tests.Addr0), common.HexToAddress(tests.Addr1), common.HexToAddress(tests.Addr2)})
	_ = snap
	storage1, _ := storage.NewMemoryStorage()
	err := snap.Store(storage1)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	snap2, _ := LoadSnapshot(storage1, hash)
	assert.Equal(t, snap.Signers, snap2.Signers, "")
}

func TestCast(t *testing.T) {
	hash := common.BytesToHash([]byte{0x01, 0x02, 0x03})
	snap := NewSnapshot(hash, []common.Address{common.HexToAddress(tests.Addr0), common.HexToAddress(tests.Addr1)})
	_ = snap

	r := snap.Cast(common.HexToAddress(tests.Addr0), common.HexToAddress(tests.Addr1), true)
	assert.False(t, r)

	r = snap.Cast(common.HexToAddress(tests.Addr0), common.HexToAddress(tests.Addr2), true)
	assert.True(t, r)
	assert.Equal(t, common.HexToAddress(tests.Addr2), snap.Votes[appendAddress(common.HexToAddress(tests.Addr0), common.HexToAddress(tests.Addr2))].Address)
	assert.Equal(t, true, snap.Votes[appendAddress(common.HexToAddress(tests.Addr0), common.HexToAddress(tests.Addr2))].Authorize)
	assert.Equal(t, 1, snap.Candidates[common.HexToAddress(tests.Addr2)].Votes)

	r = snap.Cast(common.HexToAddress(tests.Addr1), common.HexToAddress(tests.Addr2), true)
	assert.True(t, r)
	assert.Equal(t, common.HexToAddress(tests.Addr2), snap.Votes[appendAddress(common.HexToAddress(tests.Addr1), common.HexToAddress(tests.Addr2))].Address)
	assert.Equal(t, true, snap.Votes[appendAddress(common.HexToAddress(tests.Addr1), common.HexToAddress(tests.Addr2))].Authorize)
	assert.Equal(t, 2, snap.Candidates[common.HexToAddress(tests.Addr2)].Votes)

	storage1, _ := storage.NewMemoryStorage()
	err := snap.Store(storage1)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	snap2, _ := LoadSnapshot(storage1, hash)
	assert.Equal(t, common.HexToAddress(tests.Addr2), snap2.Votes[appendAddress(common.HexToAddress(tests.Addr1), common.HexToAddress(tests.Addr2))].Address)
	assert.Equal(t, true, snap2.Votes[appendAddress(common.HexToAddress(tests.Addr1), common.HexToAddress(tests.Addr2))].Authorize)
	assert.Equal(t, 2, snap2.Candidates[common.HexToAddress(tests.Addr2)].Votes)
}

var (
	addr0 = common.HexToAddress("0x000407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0")
	addr1 = common.HexToAddress("0x100407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0")
	addr2 = common.HexToAddress("0x200407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0")
	addr3 = common.HexToAddress("0x300407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0")
	addr4 = common.HexToAddress("0x400407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0")
	addr5 = common.HexToAddress("0x500407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0")
	addr6 = common.HexToAddress("0x600407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0")
	addr7 = common.HexToAddress("0x700407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0")
	addr8 = common.HexToAddress("0x800407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0")
	addr9 = common.HexToAddress("0x900407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0")
)

func TestApply(t *testing.T) {
	hash := common.BytesToHash([]byte{0x01, 0x02, 0x03})
	var ok bool
	snap := NewSnapshot(hash, []common.Address{addr0, addr1, addr2, addr3, addr4})
	//addr4 evict addr0,1,2
	snap.Cast(addr4, addr0, false)
	snap.Cast(addr4, addr1, false)
	snap.Cast(addr4, addr2, false)

	snap.Cast(addr0, addr4, false)
	snap.Apply()
	assert.Equal(t, 5, len(snap.Signers))

	snap.Cast(addr1, addr4, false)
	snap.Apply()
	assert.Equal(t, 5, len(snap.Signers))

	//before being evicted
	assert.Equal(t, 1, snap.Candidates[addr0].Votes)
	assert.Equal(t, 1, snap.Candidates[addr1].Votes)
	assert.Equal(t, 1, snap.Candidates[addr2].Votes)
	_, ok = snap.Votes[appendAddress(addr4, addr0)]
	assert.True(t, ok)

	snap.Cast(addr2, addr4, false)
	snap.Apply()
	assert.Equal(t, 4, len(snap.Signers))
	_, ok = snap.Signers[addr4]
	assert.False(t, ok)

	//after being evicted
	assert.Equal(t, 0, snap.Candidates[addr0].Votes)
	assert.Equal(t, 0, snap.Candidates[addr1].Votes)
	assert.Equal(t, 0, snap.Candidates[addr2].Votes)
	_, ok = snap.Votes[appendAddress(addr4, addr0)]
	assert.False(t, ok)
}
