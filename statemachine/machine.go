package statemachine

import (
	"cxchain223/crypto/sha3"
	"cxchain223/statdb"
	"cxchain223/trie"
	"cxchain223/types"
	"cxchain223/utils/rlp"
)

type IMachine interface {
	Execute(state trie.ITrie, text types.Transaction)
	Execute1(state statdb.StatDB, tx types.Transaction1) *types.Receiption
}

type StateMachine struct{}

func (m StateMachine) Execute1(state statdb.StatDB, tx types.Transaction1) *types.Receiption {
	//TODO implement me
	from := tx.From()
	to := tx.To
	value := tx.Value
	gasUsed := tx.Gas
	if tx.Gas < 21000 {
		return nil
	} else {
		gasUsed = 21000
	}
	gasUsed = gasUsed * tx.GasPrice //total gas
	cost := value + gasUsed
	account := state.Load(from) //the sender account
	if account == nil {
		return nil
	}
	if account.Amount < gasUsed {
		return nil
	}
	account.Amount -= cost
	state.Store(from, *account)
	_to := state.Load(to) //to_account
	if _to == nil {
		_to = &types.Account{} //new an to_account
	}
	_to.Amount += value // transfer
	state.Store(to, *_to)
	toSign, err := rlp.EncodeToBytes(tx) //encode
	if err != nil {
		return &types.Receiption{
			Status: 0,
		}
	}
	txHash := sha3.Keccak256(toSign) //sign
	receipt := &types.Receiption{
		TxHash:  &txHash,
		Status:  1,
		GasUsed: gasUsed,
	}
	return receipt
}

func (m StateMachine) Execute(state trie.ITrie, tx types.Transaction) {
	// 获取交易的发送方地址
	from := tx.From()

	// 获取交易的接收方地址和交易金额
	to := tx.To
	value := tx.Value

	// 获取交易的 gas 限额，并计算实际使用的 gas
	gasUsed := tx.Gas
	if tx.Gas < 21000 {
		return
	} else {
		gasUsed = 21000
	}
	gasUsed = gasUsed * tx.GasPrice

	// 计算交易的总成本（价值 + gas 消耗）
	cost := value + gasUsed

	// 从状态 trie 中加载发送方账户数据
	data, err := state.Load(from[:])
	if err != nil {
		return
	}

	// 解码发送方账户数据
	var account types.Account
	_ = rlp.DecodeBytes(data, &account)

	// 检查发送方账户余额是否足够支付交易成本
	if account.Amount < cost {
		return
	}

	// 扣除发送方账户中的交易成本
	account.Amount = account.Amount - cost

	// 将更新后的账户数据编码为字节流
	data, err = rlp.EncodeToBytes(account)

	// 存储更新后的发送方账户数据到状态 trie 中
	state.Store(from[:], data)

	// 加载接收方账户数据，如果不存在则创建新账户
	data, err = state.Load(to[:])
	var toAccount types.Account
	if err != nil {
		toAccount = types.Account{}
	} else {
		rlp.DecodeBytes(data, &toAccount)
	}

	// 增加接收方账户中的交易金额
	toAccount.Amount = toAccount.Amount + value

	// 将更新后的接收方账户数据编码为字节流
	data, err = rlp.EncodeToBytes(toAccount)

	// 存储更新后的接收方账户数据到状态 trie 中
	state.Store(to[:], data)
}
