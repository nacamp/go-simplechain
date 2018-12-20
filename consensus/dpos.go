package consensus

import (
	"crypto/ecdsa"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/najimmy/go-simplechain/common"
	"github.com/najimmy/go-simplechain/core"
	"github.com/najimmy/go-simplechain/log"
	"github.com/najimmy/go-simplechain/net"
	"github.com/najimmy/go-simplechain/storage"
	"github.com/najimmy/go-simplechain/trie"
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
	node         *net.Node
	coinbase     common.Address
	priv         *ecdsa.PrivateKey
	enableMining bool
}

// const
// var (
// 	ErrAddressNotEqual = errors.New("address not equal")
// )

func NewDpos() *Dpos {
	return &Dpos{}
}

func (dpos *Dpos) Setup(bc *core.BlockChain, node *net.Node, address common.Address, bpriv []byte) {
	dpos.bc = bc
	dpos.node = node
	dpos.enableMining = true
	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), bpriv)
	dpos.coinbase = common.BytesToAddress(pub.SerializeCompressed())
	dpos.priv = (*ecdsa.PrivateKey)(priv)
	if dpos.coinbase != address {
		log.CLog().WithFields(logrus.Fields{
			"Address": common.Address2Hex(dpos.coinbase),
		}).Panic("Privatekey is different")
	}
}

func (dpos *Dpos) SetupNonMiner(bc *core.BlockChain, node *net.Node) {
	dpos.bc = bc
	dpos.node = node
}

func (dpos *Dpos) MakeBlock(now uint64) *core.Block {
	bc := dpos.bc
	//TODO: check after 3 seconds(block creation) and 3 seconds(mining order)
	//Fix: when ticker is 1 second, server mining...
	turn := (now % 9) / 3
	block, err := bc.NewBlockFromParent(bc.Tail)
	if err != nil {
		log.CLog().Warning(err)
	}
	block.Header.Time = now
	minerGroup, _, err := block.MinerState.GetMinerGroup(bc, block)
	if err != nil {
		log.CLog().Warning(err)
	}
	if minerGroup[turn] == dpos.coinbase {
		parent := bc.GetBlockByHash(bc.Tail.Header.ParentHash)

		if (parent != nil) && (now-parent.Header.Time < 3) { //(3 * 3)
			log.CLog().WithFields(logrus.Fields{
				"address": common.Address2Hex(dpos.coinbase),
			}).Warning("Interval is short")
			return nil
		}

		log.CLog().WithFields(logrus.Fields{
			"address": common.Bytes2Hex(dpos.coinbase[:]),
		}).Debug("my turn")
		block.Header.Coinbase = dpos.coinbase
		block.Header.SnapshotVoterTime = bc.Tail.Header.SnapshotVoterTime // voterBlock.Header.Time
		//because PutMinerState recall GetMinerGroup , here assign  bc.Tail.Header.SnapshotVoterTime , not voterBlock.Header.Time

		//TODO: check double spending ?
		block.Transactions = make([]*core.Transaction, 0)
		accs := block.AccountState
		for {
			tx := bc.TxPool.Pop()
			if tx == nil {
				break
			}
			//TODO: remove code duplicattion in ExecuteTransaction
			fromAccount := accs.GetAccount(tx.From)
			//TODO: check at txpool
			if fromAccount == nil {
				log.CLog().WithFields(logrus.Fields{
					"Address": common.Address2Hex(tx.From),
				}).Warning("Not found account")
			} else if fromAccount.Nonce+1 == tx.Nonce {
				block.Transactions = append(block.Transactions, tx)
			} else if fromAccount.Nonce+1 < tx.Nonce {
				//use in future
				bc.TxPool.Put(tx)
			} else {
				log.CLog().WithFields(logrus.Fields{
					"Address": common.Address2Hex(tx.From),
				}).Warning("cannot accept a transaction with wrong nonce")
			}
		}
		bc.RewardForCoinbase(block)
		bc.ExecuteTransaction(block)
		block.Header.AccountHash = block.AccountState.RootHash()
		block.Header.TransactionHash = block.TransactionState.RootHash()
		// need voterHash at PutMinerState(GetMinerGroup)
		block.Header.VoterHash = block.VoterState.RootHash()
		bc.PutMinerState(block)
		block.Header.MinerHash = block.MinerState.RootHash()
		block.MakeHash()
		return block
	} else {
		log.CLog().WithFields(logrus.Fields{
			"address": common.Bytes2Hex(dpos.coinbase[:]),
		}).Debug("not my turn")
		return nil
	}
}

func (dpos *Dpos) Start() {
	if dpos.enableMining {
		go dpos.loop()
	}
}

func (dpos *Dpos) loop() {
	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case now := <-ticker.C:
			block := dpos.MakeBlock(uint64(now.Unix()))
			if block != nil {
				block.Sign(dpos.priv)
				dpos.bc.PutBlockByCoinbase(block)
				dpos.bc.Consensus.UpdateLIB(dpos.bc)
				dpos.bc.RemoveOrphanBlock()
				message, _ := net.NewRLPMessage(net.MSG_NEW_BLOCK, block.BaseBlock)
				dpos.node.BroadcastMessage(&message)
			}
		}
	}
}

//---------- Consensus
func (d *Dpos) NewMinerState(rootHash common.Hash, storage storage.Storage) (core.MinerState, error) {
	tr, err := trie.NewTrie(common.HashToBytes(rootHash), storage, false)
	return &MinerState{
		Trie: tr,
	}, err
}

func (d *Dpos) UpdateLIB(bc *core.BlockChain) {
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

func (c *Dpos) ExecuteVote(hash common.Hash, tx *core.Transaction) {

}

func (c *Dpos) NewSnapshot(hash common.Hash, addresses []common.Address) {

}
func (cs *Dpos) GetSigners(hash common.Hash) []common.Address {
	return nil
}
