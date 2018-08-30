package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
)

// subsidy 是挖出新块的奖励金
// 在比特币中，实际并没有存储这个数字，而是基于区块总数进行计算而得：区块总数除以 210000 就是 subsidy
const subsidy = 10

// 交易结构
type Transaction struct {
	ID   []byte     // 交易ID
	Vin  []TXInput  // 输入
	Vout []TXOutput // 输出
}

// 检查当前的交易是否是 Coinbase 交易
func (tx Transaction) IsCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

// SetID 设置 一笔交易的 ID
func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	var hash [32]byte

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}
	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}

// 输入结构 -- 作为一笔交易的输入
type TXInput struct {
	Txid      []byte
	Vout      int
	ScriptSig string
}

// 输出结构 -- 作为一笔交易的输出
type TXOutput struct {
	Value        int    // 比特币数量
	ScriptPubKey string // 锁定脚本 -- 要花这笔钱，必须要解锁该脚本。
}

// CanUnlockOutputWith checks whether the address initiated the transaction
func (in *TXInput) CanUnlockOutputWith(unlockingData string) bool {
	return in.ScriptSig == unlockingData
}

// CanBeUnlockedWith checks if the output can be unlocked with the provided data
func (out *TXOutput) CanBeUnlockedWith(unlockingData string) bool {
	return out.ScriptPubKey == unlockingData
}

// NewCoinbaseTX 创建一个新的 coinbase 交易
func NewCoinbaseTX(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("奖励给 '%s'", to)
	}

	txin := TXInput{[]byte{}, -1, data}
	txout := TXOutput{subsidy, to}
	tx := Transaction{nil, []TXInput{txin}, []TXOutput{txout}}
	tx.SetID()

	return &tx
}

// NewUTXOTransaction 创建一笔新的交易，并存储到一个块中
func NewUTXOTransaction(from, to string, amount int, bc *Blockchain) *Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	// 找到所有的未花费输出，并且确保它们有足够的 value（可理解为有足够的钱进行交易）
	acc, validOutputs := bc.FindSpendableOutputs(from, amount)

	if acc < amount {
		log.Panic("错误：余额不足")
	}

	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)
		if err != nil {
			log.Panic(err)
		}
		// 对于每个找到的输出，会创建一个引用该输出的输入
		for _, out := range outs {
			input := TXInput{txID, out, from}
			inputs = append(inputs, input)
		}
	}

	// 创建一个交易输出。由接收者地址锁定。这是一个付钱操作。
	outputs = append(outputs, TXOutput{amount, to})

	// 输出额度多于要付的钱
	if acc > amount {
		// 再创建一个找余额的交易输出。由发送者地址锁定。这是一个找零操作
		outputs = append(outputs, TXOutput{acc - amount, from})
	}

	// 创建交易
	tx := Transaction{nil, inputs, outputs}
	tx.SetID()

	return &tx
}
