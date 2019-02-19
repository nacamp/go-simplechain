package dpos

import (
	"errors"
	"time"

	"github.com/nacamp/go-simplechain/account"
	"github.com/nacamp/go-simplechain/common"

	"github.com/nacamp/go-simplechain/core"
	"github.com/nacamp/go-simplechain/log"
	"github.com/nacamp/go-simplechain/net"
	"github.com/sirupsen/logrus"
)

//Demo 3 accounts
var GenesisCoinbaseAddress = string("0x036407c079c962872d0ddadc121affba13090d99a9739e0d602ccfda2dab5b63c0")
var keystore = map[string]string{
	GenesisCoinbaseAddress: "0xe68fb0a479c495910c8351c3593667028b45d679f55ce22b0514c4a8a6bcbdd1",
	"0x03fdefdefbb2478f3d1ed3221d38b8bad6d939e50f17ffda40f0510b4d28506bd3": "0xf390e256b6ed8a1b283d3ea80b103b868c14c31e5b7114fc32fff21c4cb263eb",
	"0x03e864b08b08f632c61c6727cde0e23d125f7784b5a5a188446fc5c91ffa51faa1": "0xb385aca81e134722cca902bf85443528c3d3a783cf54008cfc34a2ca563fc5b6",
}

type Dpos struct {
	bc           *core.BlockChain
	coinbase     common.Address
	wallet       *account.Wallet
	enableMining bool
	streamPool   *net.PeerStreamPool
	// state        *DposState
}

// const
// var (
// 	ErrAddressNotEqual = errors.New("address not equal")
// )

// func NewDpos(node *net.Node) *Dpos {
// 	return &Dpos{node: node}
// }

func NewDpos(streamPool *net.PeerStreamPool) *Dpos {
	return &Dpos{streamPool: streamPool}
}

func (cs *Dpos) Setup(address common.Address, wallet *account.Wallet, period int) {
	cs.enableMining = true
	cs.coinbase = address
	cs.wallet = wallet
}

func (cs *Dpos) MakeBlock(now uint64) *core.Block {
	bc := cs.bc
	//TODO: check after 3 seconds(block creation) and 3 seconds(mining order)
	//Fix: when ticker is 1 second, server mining...
	turn := (now % 9) / 3
	block, err := bc.NewBlockFromParent(bc.Tail)
	if err != nil {
		log.CLog().Warning(err)
	}
	block.Header.Time = now
	state := block.ConsensusState.(*DposState)
	electedTime := state.GetNewElectedTime(state.ElectedTime, block.Header.Time, 3, 3, 3)

	//electedTime := cs.state.GetNewElectedTime(bc.Tail.Hash(), now, 3, 3, 3)
	var minerGroup []common.Address
	if electedTime != now {
		minerGroup, err = state.GetMiners(state.MinersHash)
		//minerGroup, _, err := block.MinerState.GetMinerGroup(bc, block)
		if err != nil {
			log.CLog().Warning(err)
		}
	}
	if electedTime == now || minerGroup[turn] == cs.coinbase {
		parent := bc.GetBlockByHash(bc.Tail.Header.ParentHash)

		if (parent != nil) && (now-parent.Header.Time < 3) { //(3 * 3)
			log.CLog().WithFields(logrus.Fields{
				"address": common.AddressToHex(cs.coinbase),
			}).Warning("Interval is short")
			return nil
		}

		log.CLog().WithFields(logrus.Fields{
			"address": common.BytesToHex(cs.coinbase[:]),
		}).Debug("my turn")
		block.Header.Coinbase = cs.coinbase
		//block.Header.SnapshotVoterTime = bc.Tail.Header.SnapshotVoterTime // voterBlock.Header.Time
		//because PutMinerState recall GetMinerGroup , here assign  bc.Tail.Header.SnapshotVoterTime , not voterBlock.Header.Time

		//TODO: check double spending ?
		block.Transactions = make([]*core.Transaction, 0)
		accs := block.AccountState
		for i := 0; i < bc.TxPool.Len(); i++ {
			tx := bc.TxPool.Pop()
			if tx == nil {
				break
			}
			//TODO: remove code duplicattion in ExecuteTransaction
			fromAccount := accs.GetAccount(tx.From)
			//TODO: check at txpool
			if fromAccount == nil {
				log.CLog().WithFields(logrus.Fields{
					"Address": common.AddressToHex(tx.From),
				}).Warning("Not found account")
			} else if fromAccount.Nonce+1 == tx.Nonce {
				block.Transactions = append(block.Transactions, tx)
			} else if fromAccount.Nonce+1 < tx.Nonce {
				//use in future
				bc.TxPool.Put(tx)
			} else {
				log.CLog().WithFields(logrus.Fields{
					"Address": common.AddressToHex(tx.From),
				}).Warning("cannot accept a transaction with wrong nonce")
			}
		}
		bc.RewardForCoinbase(block)
		bc.ExecuteTransaction(block)
		block.Header.AccountHash = block.AccountState.RootHash()
		block.Header.TransactionHash = block.TransactionState.RootHash()
		// need voterHash at PutMinerState(GetMinerGroup)
		cs.SaveState(block)
		if err := cs.Verify(block); err != nil {
			log.CLog().WithFields(logrus.Fields{
				"address": common.BytesToHex(cs.coinbase[:]),
			}).Debug("not my turn")
		}
		block.Header.ConsensusHash = state.RootHash()
		//이곳에 hash
		//block.Header.VoterHash = block.VoterState.RootHash()
		//bc.PutMinerState(block)
		// block.Header.MinerHash = block.MinerState.RootHash()
		block.MakeHash()
		return block
	} else {
		log.CLog().WithFields(logrus.Fields{
			"address": common.BytesToHex(cs.coinbase[:]),
		}).Debug("not my turn")
		return nil
	}
}

func (dpos *Dpos) Start() {
	if dpos.enableMining {
		go dpos.loop()
	}
}

func (cs *Dpos) loop() {
	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case now := <-ticker.C:
			block := cs.MakeBlock(uint64(now.Unix()))
			if block != nil {
				sig, err := cs.wallet.SignHash(cs.coinbase, block.Header.Hash[:])
				if err != nil {
					log.CLog().WithFields(logrus.Fields{
						"Msg": err,
					}).Warning("SignHash")
				}
				block.SignWithSignature(sig)
				cs.bc.PutBlockByCoinbase(block)
				cs.bc.Consensus.UpdateLIB()
				cs.bc.RemoveOrphanBlock()
				message, _ := net.NewRLPMessage(net.MsgNewBlock, block.BaseBlock)
				cs.streamPool.BroadcastMessage(&message)
			}
		}
	}
}

// func (d *Dpos) NewMinerState(rootHash common.Hash, storage storage.Storage) (core.MinerState, error) {
// 	tr, err := trie.NewTrie(common.HashToBytes(rootHash), storage, false)
// 	return &MinerState{
// 		Trie: tr,
// 	}, err
// }

// start to add new code

func (cs *Dpos) Verify(block *core.Block) (err error) {
	//block.Header.Coinbase
	state := block.ConsensusState.(*DposState)
	miners, err := state.GetMiners(state.MinersHash)
	if err != nil {
		return err
	}
	turn := (block.Header.Time % 9) / 3
	if miners[turn] != block.Header.Coinbase {
		return errors.New("This time is not your turn")
	}
	return nil
}

func (cs *Dpos) SaveState(block *core.Block) (err error) {
	state := block.ConsensusState.(*DposState)

	electedTime := state.GetNewElectedTime(state.ElectedTime, block.Header.Time, 3, 3, 3)

	if electedTime == block.Header.Time {
		miners, err := state.GetNewRoundMiners(block.Header.Time, 3)
		if err != nil {
			return err
		}
		state.MinersHash, err = state.PutMiners(miners)
		state.ElectedTime = block.Header.Time
	}
	state.Put(block.Header.Height, state.ElectedTime, state.MinersHash)
	return nil
}

// end...

//----------    Consensus  ----------------//

func (d *Dpos) UpdateLIB() {
	bc := d.bc
	block := bc.Tail
	//FIXME: consider timestamp, changed minerGroup
	miners := make(map[common.Address]bool)
	turn := 1
	for bc.Lib.Hash() != block.Hash() {
		miners[block.Header.Coinbase] = true
		//minerGroup, _, _ := block.MinerState.GetMinerGroup(bc, block)
		if turn == 3 {
			if len(miners) == 3 {
				bc.SetLib(block)
				log.CLog().WithFields(logrus.Fields{
					"Height": block.Header.Height,
					//"address": common.Hash2Hex(block.Hash()),
				}).Info("Updated Lib")
				return
			}
			miners = make(map[common.Address]bool)
			miners[block.Header.Coinbase] = true
			turn = 0
		}
		block = bc.GetBlockByHash(block.Header.ParentHash)
		turn++
	}
	return
}

func (c *Dpos) ConsensusType() string {
	return "DPOS"
}

func (cs *Dpos) LoadConsensusStatus(block *core.Block) (err error) {
	bc := cs.bc

	block.ConsensusState, err = NewState(block.Header.ConsensusHash, block.Header.Height, bc.Storage)
	if err != nil {
		return err
	}
	return nil
}

func (cs *Dpos) VerifyConsensusStatusHash(block *core.Block) (err error) {
	if block.VoterState.RootHash() != block.Header.VoterHash {
		return errors.New("block.VoterState.RootHash() != block.Header.VoterHash")
	}
	if block.MinerState.RootHash() != block.Header.MinerHash {
		return errors.New("block.MinerState.RootHash() != block.Header.MinerHash")
	}
	return nil
}

func (cs *Dpos) MakeGenesisBlock(block *core.Block, voters []*core.Account) error {
	bc := cs.bc

	state, err := NewState(common.Hash{}, 0, bc.Storage)
	if err != nil {
		return err
	}
	for _, v := range voters {
		state.Stake(v.Address, v.Balance)
	}
	miners, err := state.GetNewRoundMiners(block.Header.Time, 3)
	if err != nil {
		return err
	}
	state.MinersHash, err = state.PutMiners(miners)
	state.ElectedTime = block.Header.Time
	state.Put(block.Header.Height, state.ElectedTime, state.MinersHash)

	block.Header.ConsensusHash = state.RootHash()
	bc.GenesisBlock = block
	bc.GenesisBlock.MakeHash()
	return nil
}

// func (cs *Dpos) MakeGenesisBlock(block *core.Block, voters []*core.Account) error {
// 	bc := cs.bc
// 	//VoterState
// 	vs, err := core.NewAccountState(bc.Storage)
// 	if err != nil {
// 		return err
// 	}
// 	for _, account := range voters {
// 		vs.PutAccount(account)
// 	}
// 	block.VoterState = vs
// 	block.Header.VoterHash = vs.RootHash()
// 	bc.GenesisBlock = block

// 	// MinerState
// 	ms, err := cs.NewMinerState(common.Hash{}, bc.Storage)
// 	if err != nil {
// 		return err
// 	}
// 	bc.GenesisBlock.MinerState = ms
// 	minerGroup, _, err := ms.GetMinerGroup(bc, block)
// 	if err != nil {
// 		return err
// 	}
// 	ms.Put(minerGroup, bc.GenesisBlock.VoterState.RootHash())
// 	bc.GenesisBlock = block
// 	bc.GenesisBlock.Header.MinerHash = ms.RootHash()
// 	bc.GenesisBlock.Header.SnapshotVoterTime = bc.GenesisBlock.Header.Time
// 	bc.GenesisBlock.MakeHash()
// 	return nil
// }

func (cs *Dpos) AddBlockChain(bc *core.BlockChain) {
	cs.bc = bc
}

func (cs *Dpos) CloneFromParentBlock(block *core.Block, parentBlock *core.Block) (err error) {
	block.VoterState, err = parentBlock.VoterState.Clone()
	if err != nil {
		return err
	}
	block.MinerState, err = parentBlock.MinerState.Clone()
	if err != nil {
		return err
	}
	return nil
}

func (cs *Dpos) SaveMiners(block *core.Block) (err error) {
	bc := cs.bc
	if err := bc.PutMinerState(block); err != nil {
		return err
	}
	return nil
}
