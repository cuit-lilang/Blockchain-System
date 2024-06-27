package maker

import (
	"cxchain223/blockchain"
	"cxchain223/statdb"
	"cxchain223/statemachine"
	"cxchain223/trie"
	"cxchain223/txpool"
	"cxchain223/types"
	"cxchain223/utils/rlp"
	"cxchain223/utils/xtime"
	"encoding/binary"
	"time"
)

type ChainConfig struct {
	Duration   time.Duration
	Coinbase   types.Address
	Difficulty uint64
}

type BlockMaker struct {
	txpool txpool.TxPool
	state  statdb.StatDB
	exec   statemachine.IMachine

	config ChainConfig
	chain  blockchain.Blockchain

	nextHeader *blockchain.Header
	nextBody   *blockchain.Body

	interupt chan bool
	txDB     trie.ITrie
}

func NewBlockMaker(txpool txpool.TxPool, state statdb.StatDB, exec statemachine.StateMachine, chain blockchain.Blockchain) *BlockMaker {
	return &BlockMaker{
		txpool: txpool,
		state:  state,
		exec:   exec,
		chain:  chain,
	}
}

func (maker BlockMaker) NewBlock() {
	maker.nextBody = blockchain.NewBlock()
	maker.nextHeader = blockchain.NewHeader(maker.chain.CurrentHeader)
	maker.nextHeader.Coinbase = maker.config.Coinbase
}

func (maker BlockMaker) Pack() {
	end := time.After(maker.config.Duration)
	for {
		select {
		case <-maker.interupt:
			break
		case <-end:
			break
		default:
			maker.pack()
		}
	}
}

func (maker BlockMaker) pack() {
	tx := maker.txpool.Pop()
	receiption := maker.exec.Execute1(maker.state, *tx)
	maker.nextBody.Transactions = append(maker.nextBody.Transactions, *tx)
	maker.nextBody.Receiptions = append(maker.nextBody.Receiptions, *receiption)
}

func (maker BlockMaker) Interupt() {
	maker.interupt <- true
}

func (maker BlockMaker) Finalize() (*blockchain.Header, *blockchain.Body) {
	maker.nextHeader.Timestamp = xtime.Now()
	maker.nextHeader.Nonce = 0
	// TODO
	for n := uint64(0); ; n++ {
		maker.nextHeader.Nonce = n
		hash := maker.nextHeader.Hash()
		_hash := binary.BigEndian.Uint64(hash[:8]) //hash.Hash transfer to uint64
		if _hash < maker.config.Difficulty {
			break
		}
	}
	return maker.nextHeader, maker.nextBody
}
func (maker BlockMaker) Seal() {
	maker.NewBlock()
	for {
		if maker.txpool.Has() {
			maker.Pack()
		}
		if <-maker.interupt {
			return
		}
	}

}
func (maker BlockMaker) SealWithTime(t time.Duration) {
	go maker.Seal()
	ticker := time.NewTicker(t)
	for maker.txpool.Has() {
		select {
		case <-ticker.C:
			maker.Interupt()
			if len(maker.nextBody.Transactions) > 0 {
				blk := blockchain.Block{
					*maker.nextHeader,
					*maker.nextBody,
				}
				bts, err := rlp.EncodeToBytes(blk)
				if err != nil {
					panic(err)
				}
				err = maker.txDB.Store(blk.Root[:], bts)
				if err != nil {
					panic(err)
				}
				maker.chain.CurrentHeader = &blk.Header
				maker.NewBlock()
			}
			go maker.Seal()
		}
	}
}

//func GenerateChain(config *ChainConfig, parent *types.Block, engine consensus.Engine, db ethdb.Database, n int, gen func(int, *BlockGen)) ([]*types.Block, []types.Receipts) {
//	if config == nil {
//		//config = TestChainConfig
//	}
//	if engine == nil {
//		panic("nil consensus engine")
//	}
//	cm := newChainMaker(parent, config, engine)
//
//	genblock := func(i int, parent *types.Block, triedb *triedb.Database, statedb *state.StateDB) (*types.Block, types.Receipts) {
//		b := &BlockGen{i: i, cm: cm, parent: parent, statedb: statedb, engine: engine}
//		b.header = cm.makeHeader(parent, statedb, b.engine)
//
//		// Set the difficulty for clique block. The chain maker doesn't have access
//		// to a chain, so the difficulty will be left unset (nil). Set it here to the
//		// correct value.
//		if b.header.Difficulty == nil {
//			if config.TerminalTotalDifficulty == nil {
//				// Clique chain
//				b.header.Difficulty = big.NewInt(2)
//			} else {
//				// Post-merge chain
//				b.header.Difficulty = big.NewInt(0)
//			}
//		}
//		// Mutate the state and block according to any hard-fork specs
//		if daoBlock := config.DAOForkBlock; daoBlock != nil {
//			limit := new(big.Int).Add(daoBlock, params.DAOForkExtraRange)
//			if b.header.Number.Cmp(daoBlock) >= 0 && b.header.Number.Cmp(limit) < 0 {
//				if config.DAOForkSupport {
//					b.header.Extra = common.CopyBytes(params.DAOForkBlockExtra)
//				}
//			}
//		}
//		if config.DAOForkSupport && config.DAOForkBlock != nil && config.DAOForkBlock.Cmp(b.header.Number) == 0 {
//			misc.ApplyDAOHardFork(statedb)
//		}
//		// Execute any user modifications to the block
//		if gen != nil {
//			gen(i, b)
//		}
//
//		body := types.Body{Transactions: b.txs, Uncles: b.uncles, Withdrawals: b.withdrawals}
//		block, err := b.engine.FinalizeAndAssemble(cm, b.header, statedb, &body, b.receipts)
//		if err != nil {
//			panic(err)
//		}
//
//		// Write state changes to db
//		root, err := statedb.Commit(b.header.Number.Uint64(), config.IsEIP158(b.header.Number))
//		if err != nil {
//			panic(fmt.Sprintf("state write error: %v", err))
//		}
//		if err = triedb.Commit(root, false); err != nil {
//			panic(fmt.Sprintf("trie write error: %v", err))
//		}
//		return block, b.receipts
//	}
//
//	// Forcibly use hash-based state scheme for retaining all nodes in disk.
//	triedb := triedb.NewDatabase(db, triedb.HashDefaults)
//	defer triedb.Close()
//
//	for i := 0; i < n; i++ {
//		statedb, err := state.New(parent.Root(), state.NewDatabaseWithNodeDB(db, triedb), nil)
//		if err != nil {
//			panic(err)
//		}
//		block, receipts := genblock(i, parent, triedb, statedb)
//
//		// Post-process the receipts.
//		// Here we assign the final block hash and other info into the receipt.
//		// In order for DeriveFields to work, the transaction and receipt lists need to be
//		// of equal length. If AddUncheckedTx or AddUncheckedReceipt are used, there will be
//		// extra ones, so we just trim the lists here.
//		receiptsCount := len(receipts)
//		txs := block.Transactions()
//		if len(receipts) > len(txs) {
//			receipts = receipts[:len(txs)]
//		} else if len(receipts) < len(txs) {
//			txs = txs[:len(receipts)]
//		}
//		var blobGasPrice *big.Int
//		if block.ExcessBlobGas() != nil {
//			blobGasPrice = eip4844.CalcBlobFee(*block.ExcessBlobGas())
//		}
//		if err := receipts.DeriveFields(config, block.Hash(), block.NumberU64(), block.Time(), block.BaseFee(), blobGasPrice, txs); err != nil {
//			panic(err)
//		}
//
//		// Re-expand to ensure all receipts are returned.
//		receipts = receipts[:receiptsCount]
//
//		// Advance the chain.
//		cm.add(block, receipts)
//		parent = block
//	}
//	return cm.chain, cm.receipts
//}
