package main

import "bytes"

// 输入结构
type TXInput struct {
	Txid      []byte
	Vout      int
	Signature []byte //
	PubKey    []byte //
}

// UsesKey 方法检查输入使用了指定密钥来解锁一个输出
func (in *TXInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := HashPubKey(in.PubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}
