package pow

import (
	"github.com/nacamp/go-simplechain/common"
	"github.com/nacamp/go-simplechain/core"
)

type PowState struct {
}

func (cs *PowState) Clone() (core.ConsensusState, error) {
	return &PowState{}, nil
}

func (cs *PowState) ExecuteTransaction(block *core.Block, txIndex int, account *core.Account) (err error) {
	return nil
}

func (cs *PowState) RootHash() (hash common.Hash) {
	return common.Hash{}
}
