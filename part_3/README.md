## 持久化

#### 数据库 -- BoltDB

使用数据库 BoltDB 来完成持久化。BoltDB 特点：

1. Bolt 使用键值存储。数据被存储为键值对，存储在 bucket 中，类似于关系型数据库中的表。
2. Bolt 中 键和值都是字节数组（byte array）。在 Go 中，struct 数据需要转换为 byte array。这里使用 encoding/gob 来完成这一目标。

#### 这里使用两个 bucket 来存储数据

1. 一个 bucket 是 blocks，它存储了描述一条链中所有块的元数据
2. 另一个 bucket 是 chainstate，存储了一条链的状态，也就是当前所有的未花费的交易输出，和一些元数据

#### 这里会用到的键值对
1. 32 字节的 block-hash -> block 结构
2. l -> 链中最后一个块的 hash


#### 存储步骤 -- 见 blockchain.go
1. 打开一个数据库文件
2. 检查文件里面是否已经存储了一个区块链 
3.如果已经存储了一个区块链：
   1. 创建一个新的 Blockchain 实例
   2. 设置 Blockchain 实例的 tip 为数据库中存储的最后一个块的哈希
4. 如果没有区块链：
   1. 创建创世块
   2. 存储到数据库
   3. 将创世块哈希保存为最后一个块的哈希
   4. 创建一个新的 Blockchain 实例，初始时 tip 指向创世块（tip 有尾部，尖端的意思，在这里 tip 存储的是最后一个块的哈希）

## 命令行接口

简单实现还没有提供一个与程序交互的接口(CLI 程序)，可以使用命令去 新建区块链 和 添加区块。

#### CLI 使用

``` bash
# 打印区块链
./goBlockchain printchain

# 新增区块
./goBlockchain addblock -data "转给 1毛钱 给 李四"

```