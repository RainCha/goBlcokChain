package main

import "fmt"

// 初始化 UTXO 集合
func (cli *CLI) reindexUTXO() {
	bc := NewBlockchain()
	UTXOSet := UTXOSet{bc}
	UTXOSet.Reindex()

	count := UTXOSet.CountTransactions()
	fmt.Printf("初始化完成！在 UTXO集合中，存在 %d 条交易.\n", count)
}
