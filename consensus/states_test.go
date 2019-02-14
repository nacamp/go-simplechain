package consensus

import (
	"testing"

	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/tests"
)

func TestStates(t *testing.T) {
	addrs := []common.Address{
		common.HexToAddress(tests.Addr0),
		common.HexToAddress(tests.Addr1),
		common.HexToAddress(tests.Addr2),
	}
	shuffle(addrs)
}
