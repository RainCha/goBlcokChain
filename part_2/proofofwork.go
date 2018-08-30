package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"
)

var (
	maxNonce = math.MaxInt64
)

// 前24位必须是6, 用16位表示即 前6位必须是0
const targetBits = 24

// ProofOfWork 结构
type ProofOfWork struct {
	block  *Block
	target *big.Int //
}

// 创建工作量证明
func NewProofOfWork(b *Block) *ProofOfWork {

	//  big.Int 初始化为 1，然后左移 256 - targetBits 位
	//  然后，target（目标） 的 16 进制形式为：
	//  0x10000000000000000000000000000000000000000000000000000000000
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))

	pow := &ProofOfWork{b, target}

	return pow
}

// 准备用来计算哈希的数据
func (pow *ProofOfWork) prepareData(nonce int) []byte {
	data := bytes.Join(
		[][]byte{
			pow.block.PrevBlockHash,
			pow.block.Data,
			IntToHex(pow.block.Timestamp), // 把 int64的数据 转为 byte 数组
			IntToHex(int64(targetBits)), 
			IntToHex(int64(nonce)), 
		},
		[]byte{},
	)

	return data
}

// 开始工作，找到正确的哈希
func (pow *ProofOfWork) Run() (int, []byte) {
	var hashInt big.Int
	var hash [32]byte
	nonce := 0

	fmt.Printf("挖矿--计算满足条件的哈希 \"%s\"\n", pow.block.Data)
	for nonce < maxNonce {
		data := pow.prepareData(nonce)

		// 得到一个hash, 并转为大整数
		hash = sha256.Sum256(data)
		fmt.Printf("\r%x", hash)
		hashInt.SetBytes(hash[:])

		// 如果大整数小于目标（即前6位为0），则这个hash是有效的
		if hashInt.Cmp(pow.target) == -1 {
			break
		} else {
			nonce++
		}
	}
	fmt.Print("\n\n")

	return nonce, hash[:]
}

// Validate validates block's PoW
func (pow *ProofOfWork) Validate() bool {
	var hashInt big.Int

	data := pow.prepareData(pow.block.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])

	isValid := hashInt.Cmp(pow.target) == -1

	return isValid
}
