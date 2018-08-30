package main

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
)

const protocol = "tcp"
const nodeVersion = 1
const commandLength = 12

var nodeAddress string
var miningAddress string

// 对中心节点的地址进行硬编码：因为每个节点必须知道从何处开始初始化
var knownNodes = []string{"localhost:3000"}
var blocksInTransit = [][]byte{}
var mempool = make(map[string]Transaction)

// 允许节点来互相发现彼此
type addr struct {
	AddrList []string
}

// 为完成数据转移
type block struct {
	AddrFrom string
	Block    []byte
}

// getblocks 意为 “给我看一下你有什么区块,
type getblocks struct {
	AddrFrom string
}

// getdata 用于某个块或交易的请求，它可以仅包含一个块或交易的 ID。
type getdata struct {
	AddrFrom string
	Type     string
	ID       []byte
}

// inv 来向其他节点展示当前节点有什么块和交易, 它没有包含完整的区块链和交易，仅仅是哈希而已
type inv struct {
	AddrFrom string
	Type     string // 表明了这是块还是交易
	Items    [][]byte
}

// 为完成数据转移
type tx struct {
	AddFrom     string
	Transaction []byte
}

// 版本信息 --- 用于在新节点运行时，向其他节点发送消息时使用
type verzion struct {
	Version    int    // 仅有一个区块链版本,Version 并不会存储什么信息
	BestHeight int    // 存储区块链中节点的高度
	AddrFrom   string // 存储发送节点的地址
}

// commandToBytes 创建一个 12 字节的缓冲区，并用命令名进行填充，将剩下的字节置为空
func commandToBytes(command string) []byte {
	var bytes [commandLength]byte

	for i, c := range command {
		bytes[i] = byte(c)
	}

	return bytes[:]
}

// 与 commandToBytes 功能相反
func bytesToCommand(bytes []byte) string {
	var command []byte

	for _, b := range bytes {
		if b != 0x0 {
			command = append(command, b)
		}
	}

	return fmt.Sprintf("%s", command)
}

func extractCommand(request []byte) []byte {
	return request[:commandLength]
}

func requestBlocks() {
	for _, node := range knownNodes {
		sendGetBlocks(node)
	}
}

func sendAddr(address string) {
	nodes := addr{knownNodes}
	nodes.AddrList = append(nodes.AddrList, nodeAddress)
	payload := gobEncode(nodes)
	request := append(commandToBytes("addr"), payload...)

	sendData(address, request)
}

func sendBlock(addr string, b *Block) {
	data := block{nodeAddress, b.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes("block"), payload...)

	sendData(addr, request)
}

func sendData(addr string, data []byte) {
	conn, err := net.Dial(protocol, addr)
	if err != nil {
		fmt.Printf("%s is not available\n", addr)
		var updatedNodes []string

		for _, node := range knownNodes {
			if node != addr {
				updatedNodes = append(updatedNodes, node)
			}
		}

		knownNodes = updatedNodes

		return
	}
	defer conn.Close()

	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		log.Panic(err)
	}
}

func sendInv(address, kind string, items [][]byte) {
	inventory := inv{nodeAddress, kind, items}
	payload := gobEncode(inventory)
	request := append(commandToBytes("inv"), payload...)

	sendData(address, request)
}

// getblocks 意为 “给我看一下你有什么区块”
func sendGetBlocks(address string) {
	payload := gobEncode(getblocks{nodeAddress})
	request := append(commandToBytes("getblocks"), payload...)

	sendData(address, request)
}

func sendGetData(address, kind string, id []byte) {
	payload := gobEncode(getdata{nodeAddress, kind, id})
	request := append(commandToBytes("getdata"), payload...)

	sendData(address, request)
}

func sendTx(addr string, tnx *Transaction) {
	data := tx{nodeAddress, tnx.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes("tx"), payload...)

	sendData(addr, request)
}

// 发送版本消息 -- 消息，在底层就是字节序列
func sendVersion(addr string, bc *Blockchain) {
	bestHeight := bc.GetBestHeight()
	payload := gobEncode(verzion{nodeVersion, bestHeight, nodeAddress})

	//前 12 个字节指定了命令名（比如这里的 version），后面的字节会包含 gob 编码的消息结构
	request := append(commandToBytes("version"), payload...)

	sendData(addr, request)
}

func handleAddr(request []byte) {
	var buff bytes.Buffer
	var payload addr

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	knownNodes = append(knownNodes, payload.AddrList...)
	fmt.Printf("There are %d known nodes now!\n", len(knownNodes))
	requestBlocks()
}

func handleBlock(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload block

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	blockData := payload.Block
	block := DeserializeBlock(blockData)

	// 当接收到一个新块时，我们把它放到区块链里面。TODO 应该对块做校验
	fmt.Println("Recevied a new block!")
	bc.AddBlock(block)

	fmt.Printf("Added block %x\n", block.Hash)

	//  如果还有更多的区块需要下载，我们继续从上一个下载的块的那个节点继续请求
	if len(blocksInTransit) > 0 {
		blockHash := blocksInTransit[0]
		sendGetData(payload.AddrFrom, "block", blockHash)

		blocksInTransit = blocksInTransit[1:]
	} else {
		// 当最后把所有块都下载完后，对 UTXO 集进行重新索引。
		UTXOSet := UTXOSet{bc}
		UTXOSet.Reindex()
	}
}

// 处理 Inv 消息
func handleInv(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload inv

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("Recevied inventory with %d %s\n", len(payload.Items), payload.Type)

	// 如果收到块哈希，我们想要将它们保存在 blocksInTransit 变量来跟踪已下载的块。这能够让我们从不同的节点下载块。
	if payload.Type == "block" {
		blocksInTransit = payload.Items

		// 在将块置于传送状态时，我们给 inv 消息的发送者发送 getdata 命令。
		blockHash := payload.Items[0]
		sendGetData(payload.AddrFrom, "block", blockHash)

		newInTransit := [][]byte{}
		for _, b := range blocksInTransit {
			if bytes.Compare(b, blockHash) != 0 {
				newInTransit = append(newInTransit, b)
			}
		}
		// 更新 blocksInTransit
		blocksInTransit = newInTransit
	}

	if payload.Type == "tx" {
		txID := payload.Items[0]

		if mempool[hex.EncodeToString(txID)].ID == nil {
			sendGetData(payload.AddrFrom, "tx", txID)
		}
	}
}

// handleGetBlocks 并不是“把你全部的区块给我”，而是请求了一个块哈希的列表。
// 这是为了减轻网络负载，因为区块可以从不同的节点下载，并且我们不想从一个单一节点下载数十 GB 的数据。
func handleGetBlocks(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload getblocks

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}
	// 请求区块的hash列表
	blocks := bc.GetBlockHashes()
	sendInv(payload.AddrFrom, "block", blocks)
}

func handleGetData(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload getdata

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	// 如果它们请求一个块，则返回块
	if payload.Type == "block" {
		block, err := bc.GetBlock([]byte(payload.ID))
		if err != nil {
			return
		}

		sendBlock(payload.AddrFrom, &block)
	}

	// 如果它们请求一笔交易，则返回交易
	if payload.Type == "tx" {
		txID := hex.EncodeToString(payload.ID)
		tx := mempool[txID]

		sendTx(payload.AddrFrom, &tx)
		// delete(mempool, txID)
	}
}

// 处理交易
func handleTx(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload tx

	// 首先要做的事情是将新交易放到内存池中（再次提醒，在将交易放到内存池之前，必要对其进行验证）
	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	txData := payload.Transaction
	tx := DeserializeTransaction(txData)
	mempool[hex.EncodeToString(tx.ID)] = tx

	// 检查当前节点是否是中心节点。
	// 在我们的实现中，中心节点并不会挖矿。它只会将新的交易推送给网络中的其他节点。
	if nodeAddress == knownNodes[0] {
		for _, node := range knownNodes {
			if node != nodeAddress && node != payload.AddFrom {
				sendInv(node, "tx", [][]byte{tx.ID})
			}
		}
	} else {
		// 矿工节点 -- miningAddress 只会在矿工节点上设置。
		if len(mempool) >= 2 && len(miningAddress) > 0 {
			// 如果当前节点（矿工）的内存池中有两笔或更多的交易，开始挖矿
		MineTransactions:
			var txs []*Transaction

			// 首先，内存池中所有交易都是要通过验证的
			for id := range mempool {
				tx := mempool[id]
				// 	// 无效的交易会被忽略，
				if bc.VerifyTransaction(&tx) {
					txs = append(txs, &tx)
				}
			}
			//	如果没有有效交易，则挖矿中断
			if len(txs) == 0 {
				fmt.Println("All transactions are invalid! Waiting for new ones...")
				return
			}

			// 验证后的交易被放到一个块里，同时还有附带奖励的 coinbase 交易
			cbTx := NewCoinbaseTX(miningAddress, "")
			txs = append(txs, cbTx)

			// 当块被挖出来以后，UTXO 集会被重新索引
			newBlock := bc.MineBlock(txs)
			UTXOSet := UTXOSet{bc}
			UTXOSet.Reindex()

			fmt.Println("New block is mined!")

			// 当一笔交易被挖出来以后，就会被从内存池中移除。
			for _, tx := range txs {
				txID := hex.EncodeToString(tx.ID)
				delete(mempool, txID)
			}

			// 当前节点所连接到的所有其他节点，接收带有新块哈希的 inv 消息。
			// 在处理完消息后，它们可以对块进行请求。
			for _, node := range knownNodes {
				if node != nodeAddress {
					sendInv(node, "block", [][]byte{newBlock.Hash})
				}
			}

			if len(mempool) > 0 {
				goto MineTransactions
			}
		}
	}
}

// 处理版本消息连接
func handleVersion(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload verzion

	// 首先，我们需要对请求进行解码，提取有效信息
	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	myBestHeight := bc.GetBestHeight()
	foreignerBestHeight := payload.BestHeight

	// 然后节点将从消息中提取的 BestHeight 与自身进行比较。
	// 如果自身节点的区块链更长，它会回复 version 消息；否则，它会发送 getblocks 消息。
	if myBestHeight < foreignerBestHeight {
		sendGetBlocks(payload.AddrFrom)
	} else if myBestHeight > foreignerBestHeight {
		sendVersion(payload.AddrFrom, bc)
	}

	// sendAddr(payload.AddrFrom)
	if !nodeIsKnown(payload.AddrFrom) {
		knownNodes = append(knownNodes, payload.AddrFrom)
	}
}

// 处理连接消息
func handleConnection(conn net.Conn, bc *Blockchain) {
	request, err := ioutil.ReadAll(conn)
	if err != nil {
		log.Panic(err)
	}
	// 运行 bytesToCommand 来提取命令名
	command := bytesToCommand(request[:commandLength])
	fmt.Printf("Received %s command\n", command)

	// 选择正确的处理器处理命令主体
	switch command {
	case "addr":
		handleAddr(request)
	case "block":
		handleBlock(request, bc)
	case "inv":
		handleInv(request, bc)
	case "getblocks":
		handleGetBlocks(request, bc)
	case "getdata":
		handleGetData(request, bc)
	case "tx":
		handleTx(request, bc)
	case "version":
		handleVersion(request, bc)
	default:
		fmt.Println("Unknown command!")
	}

	conn.Close()
}

// StartServer 启动一个新节点
func StartServer(nodeID, minerAddress string) {
	nodeAddress = fmt.Sprintf("localhost:%s", nodeID)
	// minerAddress 参数指定了接收挖矿奖励的地址
	miningAddress = minerAddress
	ln, err := net.Listen(protocol, nodeAddress)
	if err != nil {
		log.Panic(err)
	}
	defer ln.Close()

	bc := NewBlockchain(nodeID)

	// 如果当前节点不是中心节点，它必须向中心节点发送 version 消息来查询是否自己的区块链已过时
	if nodeAddress != knownNodes[0] {
		sendVersion(knownNodes[0], bc)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Panic(err)
		}

		// 一个节点接收到消息后的处理
		go handleConnection(conn, bc)
	}
}

func gobEncode(data interface{}) []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

func nodeIsKnown(addr string) bool {
	for _, node := range knownNodes {
		if node == addr {
			return true
		}
	}

	return false
}
