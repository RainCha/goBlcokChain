package main

import (
	"crypto/sha256"
)

// MerkleTree 结构
type MerkleTree struct {
	RootNode *MerkleNode
}

// MerkleNode 节点 结构
type MerkleNode struct {
	Left  *MerkleNode // 每个 MerkleNode 包含数据和指向左右分支的指针
	Right *MerkleNode
	Data  []byte // 每个节点包含一些数据
}

// NewMerkleTree 创建默克尔树。
func NewMerkleTree(data [][]byte) *MerkleTree {
	var nodes []MerkleNode

	// 当生成一棵新树时，要确保的第一件事就是叶子节点必须是双数
	if len(data)%2 != 0 {
		data = append(data, data[len(data)-1])
	}

	// 数据（也就是一个序列化后交易的数组）被转换成树的叶子，从这些叶子再慢慢形成一棵树
	for _, datum := range data {
		node := NewMerkleNode(nil, nil, datum)
		nodes = append(nodes, *node)
	}

	for i := 0; i < len(data)/2; i++ {
		var newLevel []MerkleNode

		for j := 0; j < len(nodes); j += 2 {
			node := NewMerkleNode(&nodes[j], &nodes[j+1], nil)
			newLevel = append(newLevel, *node)
		}

		nodes = newLevel
	}

	mTree := MerkleTree{&nodes[0]}

	return &mTree
}

// NewMerkleNode 创建 默克尔树的节点
func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	mNode := MerkleNode{}

	if left == nil && right == nil {
		// 当节点在叶子节点，数据从外界传入（在这里，也就是一个序列化后的交易）
		hash := sha256.Sum256(data)
		// 每个节点包含一些数据
		mNode.Data = hash[:]
	} else {
		// 当一个节点被关联到其他节点，它会将其他节点的数据取过来，连接后再哈希
		prevHashes := append(left.Data, right.Data...)
		hash := sha256.Sum256(prevHashes)
		// 每个节点包含一些数据
		mNode.Data = hash[:]
	}

	mNode.Left = left
	mNode.Right = right

	return &mNode
}
