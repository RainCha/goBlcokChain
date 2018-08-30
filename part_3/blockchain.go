package main

import (
	"fmt"
	"log"

	"github.com/boltdb/bolt"
)

// 定义数据库本地存储文件 和 bucket 名称
const dbFile = "blockchain.db"
const blocksBucket = "blocks"

// 区块链结构 -- 增加数据库
type Blockchain struct {
	tip []byte // 数据库中存储的最后一个区块的hash
	db  *bolt.DB
}

// 区块链迭代器 -- 用来按顺序，一个个打印区块链中的区块
type BlockchainIterator struct {
	currentHash []byte   // 当前迭代的块hash
	db          *bolt.DB // 数据库的连接
}

// 向区块链中添加区块的方法
func (bc *Blockchain) AddBlock(data string) {
	var lastHash []byte

	// 使用一个只读事务，获取数据库中保存的最后一区块的hash
	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte("l"))

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	// 使用这个hash来新建一个区块
	newBlock := NewBlock(data, lastHash)

	// 把新建的区块序列化后，保存到数据库中
	err = bc.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		err := b.Put(newBlock.Hash, newBlock.Serialize())
		if err != nil {
			log.Panic(err)
		}
		// 更新数据库的 l值
		err = b.Put([]byte("l"), newBlock.Hash)
		if err != nil {
			log.Panic(err)
		}

		bc.tip = newBlock.Hash

		return nil
	})
}

// 迭代区块链方法
func (bc *Blockchain) Iterator() *BlockchainIterator {
	bci := &BlockchainIterator{bc.tip, bc.db}

	return bci
}

// 迭代器方法 -- 返回链中的下一个区块
func (i *BlockchainIterator) Next() *Block {
	var block *Block

	err := i.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(i.currentHash)
		block = DeserializeBlock(encodedBlock)

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	i.currentHash = block.PrevBlockHash

	return block
}

// 新建区块链
func NewBlockchain() *Blockchain {
	var tip []byte
	// 打开一个 BoltDB 文件
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	// 开始一个读写事务
	err = db.Update(func(tx *bolt.Tx) error {

		// 获取存储区块的 bucket
		b := tx.Bucket([]byte(blocksBucket))

		if b == nil {
			fmt.Println("数据库中没有区块链. 正在新建...")

			// 新建创始块
			genesis := NewGenesisBlock()

			// 创建一个bucket
			b, err := tx.CreateBucket([]byte(blocksBucket))
			if err != nil {
				log.Panic(err)
			}

			// 把创始块存储下来，使用 创始块的hash 作为key, 序列化后的创始块作为 value
			err = b.Put(genesis.Hash, genesis.Serialize())
			if err != nil {
				log.Panic(err)
			}

			// 存储链中最后一个区块的 hash(此时为创始块的hash)
			err = b.Put([]byte("l"), genesis.Hash)
			if err != nil {
				log.Panic(err)
			}
			tip = genesis.Hash
		} else {
			// 从 bucket 获取链中最后一块的hash
			tip = b.Get([]byte("l"))
		}

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	// 创建区块链（现在区块链就存到的数据库里，不向之前都在程序运行的内存里）
	bc := Blockchain{tip, db}

	return &bc
}
