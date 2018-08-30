package main

import (
	"bytes"
	"crypto/sha256"
	"strconv"
	"time"
)

// 块结构
type Block struct {
	Timestamp     int64  // 当前时间戳，也就是区块创建的时间
	Data          []byte // 区块存储的实际有效信息，也就是交易信息
	PrevBlockHash []byte // 前一个块的哈希，即父哈希
	Hash          []byte // 当前块的哈希
}

// SetHash 用来计算和设置区块的哈希
func (b *Block) SetHash() {
	timestamp := []byte(strconv.FormatInt(b.Timestamp, 10))
	headers := bytes.Join([][]byte{b.PrevBlockHash, b.Data, timestamp}, []byte{})
	hash := sha256.Sum256(headers)
	b.Hash = hash[:]
}

// 新块创建，返回创建后的区块
func NewBlock(data string, prevBlockHash []byte) *Block {
	block := &Block{
		time.Now().Unix(),
		[]byte(data),
		prevBlockHash,
		[]byte{},
	}
	block.SetHash()
	return block
}

// 创世快的创建，返回第一个创世块
func NewGenesisBlock() *Block {
	return NewBlock("Genesis Block", []byte{})
}
