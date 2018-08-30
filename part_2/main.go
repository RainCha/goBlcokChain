package main

import (
	"fmt"
	"strconv"
)

func main() {
	bc := NewBlockchain()

	bc.AddBlock("发送 1 BTC 给 Ivan")
	bc.AddBlock("发送 2 BTC 给 Ivan")

	for _, block := range bc.blocks {
		fmt.Printf("前一个区块的哈希: %x\n", block.PrevBlockHash)
		fmt.Printf("数据: %s\n", block.Data)
		fmt.Printf("哈希: %x\n", block.Hash)

		// 验证块的hash是否满足条件
		pow := NewProofOfWork(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		
		fmt.Println()
	}
}
