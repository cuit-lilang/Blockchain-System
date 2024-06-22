package blockchain

import (
	"cxchain223/crypto/sha3"
	"cxchain223/trie"
	"cxchain223/txpool"
	"cxchain223/types"
	"cxchain223/utils/hash"
	"cxchain223/utils/rlp"
)

type Header struct {
	Root       hash.Hash
	ParentHash hash.Hash
	Height     uint64
	Coinbase   types.Address
	Timestamp  uint64

	Nonce uint64
}

type Body struct {
	Transactions []types.Transaction1
	Receiptions  []types.Receiption
}

func (header Header) Hash() hash.Hash {
	data, _ := rlp.EncodeToBytes(header)
	return sha3.Keccak256(data)
}

// NewHeader new a header of block
func NewHeader(parent Header) *Header {
	return &Header{
		Root:       parent.Root,
		ParentHash: parent.Hash(),
		Height:     parent.Height + 1,
	}
}

// NewBlock new a body of block
func NewBlock() *Body {
	return &Body{
		Transactions: make([]types.Transaction1, 0),
		Receiptions:  make([]types.Receiption, 0),
	}
}

type Blockchain struct {
	CurrentHeader Header
	Statedb       trie.ITrie
	Txpool        txpool.TxPool
}
