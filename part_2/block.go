package main

import (
	"time"
)

// 块结构
type Block struct {
	Timestamp     int64
	Data          []byte
	PrevBlockHash []byte
	Hash          []byte
	Nonce         int // 计数器
}

// 创建块 -- 增加工作量证明机制
func NewBlock(data string, prevBlockHash []byte) *Block {
	block := &Block{time.Now().Unix(), []byte(data), prevBlockHash, []byte{}, 0}

	// 创建区块，计算有效哈希
	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()

	// 存储 nonce ，方便之后验证
	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

// 创世块
func NewGenesisBlock() *Block {
	return NewBlock("Genesis Block", []byte{})
}
