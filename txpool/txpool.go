package txpool

import (
	"cxchain223/types"
	"cxchain223/utils/hash"
)

type TxPool interface {
	SetStatRoot(root hash.Hash)
	NewTx(tx *types.Transaction1)
	Pop() *types.Transaction1
	NotifyTxEvent(txs []*types.Transaction1)
}
