package main

// 区块链结构 -- 存储区块的一个队列
type Blockchain struct {
	blocks []*Block
}

// 往区块链中添加一个区块
func (bc *Blockchain) AddBlock(data string) {
	prevBlock := bc.blocks[len(bc.blocks)-1]
	newBlock := NewBlock(data, prevBlock.Hash)
	bc.blocks = append(bc.blocks, newBlock)
}

// 新建一个区块链 并 创建第一个创世块
func NewBlockchain() *Blockchain {
	return &Blockchain{[]*Block{NewGenesisBlock()}}
}
