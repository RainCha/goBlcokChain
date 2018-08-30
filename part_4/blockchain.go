package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/boltdb/bolt"
)

// 定义数据库本地存储文件 和 bucket 名称
const dbFile = "blockchain.db"
const blocksBucket = "blocks"

// 创始块交易信息
const genesisCoinbaseData = "第一个创始块生成"

// 区块链
type Blockchain struct {
	tip []byte // 数据库中存储的最后一个区块的hash
	db  *bolt.DB
}

// 区块链迭代器 -- 用来按顺序，一个个打印区块链中的区块
type BlockchainIterator struct {
	currentHash []byte   // 当前迭代的块hash
	db          *bolt.DB // 数据库的连接
}

//使用交易信息 新建区块
func (bc *Blockchain) MineBlock(transactions []*Transaction) {
	var lastHash []byte

	err := bc.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte("l"))

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	// 使用这个hash来新建一个区块
	newBlock := NewBlock(transactions, lastHash)

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

// 查询未被花费的交易
func (bc *Blockchain) FindUnspentTransactions(address string) []Transaction {
	var unspentTXs []Transaction
	spentTXOs := make(map[string][]int)
	bci := bc.Iterator()

	for {
		// 由于交易被存储在区块里，所以我们不得不检查区块链里的每一笔交易
		block := bci.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Vout {

				// 检查该输出是否已经被包含在一个交易的输入中, 也就是检查它是否已经被花费了
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}

				// 如果一个输出被一个地址锁定，并且这个地址恰好是我们要找的地址，那么这个输出就是我们想要的
				if out.CanBeUnlockedWith(address) {
					unspentTXs = append(unspentTXs, *tx)
				}
			}

			// coinbase 交易不解锁输出, 所以先排除coinbase交易，再将给定地址所有能够解锁输出的输入聚集起来
			if tx.IsCoinbase() == false {
				for _, in := range tx.Vin {
					if in.CanUnlockOutputWith(address) {
						inTxID := hex.EncodeToString(in.Txid)
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout)
					}
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	// 返回了一个交易列表，里面包含了未花费输出
	return unspentTXs
}

// FindUTXO 查找地址中所有未被花费的交易输出汇总
func (bc *Blockchain) FindUTXO(address string) []TXOutput {
	var UTXOs []TXOutput
	unspentTransactions := bc.FindUnspentTransactions(address)

	for _, tx := range unspentTransactions {
		for _, out := range tx.Vout {
			if out.CanBeUnlockedWith(address) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs
}

// 找到所有的未花费输出
func (bc *Blockchain) FindSpendableOutputs(address string, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	unspentTXs := bc.FindUnspentTransactions(address)
	accumulated := 0

Work:
	// 对所有的未花费交易进行迭代，并对它的值进行累加
	for _, tx := range unspentTXs {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Vout {
			if out.CanBeUnlockedWith(address) && accumulated < amount {
				// 累加输出余额（）
				accumulated += out.Value
				// 通过交易 ID 进行分组输出
				unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)

				// 当累加值大于或等于我们想要传送的值时，它就会停止
				if accumulated >= amount {
					break Work
				}
			}
		}
	}
	// 返回累加值，同时返回的还有通过交易 ID 进行分组的输出索引
	return accumulated, unspentOutputs
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

func dbExists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}

// 新建区块链
func NewBlockchain(address string) *Blockchain {
	if dbExists() == false {
		fmt.Println("暂未区块链，请先创建！")
		os.Exit(1)
	}

	var tip []byte
	// 打开一个 BoltDB 文件
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}
	// 开始一个读写事务
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		tip = b.Get([]byte("l"))

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	bc := Blockchain{tip, db}

	return &bc
}

// CreateBlockchain 在数据库上创建一个新的区块链
func CreateBlockchain(address string) *Blockchain {
	if dbExists() {
		fmt.Println("区块链已经存在！")
		os.Exit(1)
	}

	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		cbtx := NewCoinbaseTX(address, genesisCoinbaseData)
		genesis := NewGenesisBlock(cbtx)

		b, err := tx.CreateBucket([]byte(blocksBucket))
		if err != nil {
			log.Panic(err)
		}

		err = b.Put(genesis.Hash, genesis.Serialize())
		if err != nil {
			log.Panic(err)
		}

		err = b.Put([]byte("l"), genesis.Hash)
		if err != nil {
			log.Panic(err)
		}
		tip = genesis.Hash

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	bc := Blockchain{tip, db}

	return &bc
}
