交易2
------

### 挖矿奖励


### chainstate  --- 存储 UTXO集
1. chainstate 不存储交易。它所存储的是 UTXO 集，也就是未花费交易输出的集合。
2. 作用：由于交易被保存在区块中，要找到所有未花费输出的交易， 就需要对区块链里面的每一个区块进行迭代，检查里面的每一笔交易。但是区块数会越来越多，遍历整个链，将会十分耗费性能。因此，解决方案是要有一个仅有未花费输出的索引，这就是 UTXO 集要做的事情。

##### 经常需要迭代链的情况
1. Blockchain.FindUnspentTransactions - 找到有未花费输出交易的主要函数。也是在这个函数里面会对所有区块进行迭代。
2. Blockchain.FindSpendableOutputs - 这个函数用于当一个新的交易创建的时候。如果找到有所需数量的输出。使用 Blockchain.FindUnspentTransactions.
3. Blockchain.FindUTXO - 找到一个公钥哈希的未花费输出，然后用来获取余额。使用 Blockchain.FindUnspentTransactions.
4. Blockchain.FindTransation - 根据 ID 在区块链中找到一笔交易。它会在所有块上进行迭代直到找到它。

##### 优化方法
1. Blockchain.FindUTXO - 通过对区块进行迭代找到所有未花费输出。
2. UTXOSet.Reindex - 使用 UTXO 找到未花费输出，然后在数据库中进行存储。这里就是缓存的地方。
3. UTXOSet.FindSpendableOutputs - 类似 Blockchain.FindSpendableOutputs，但是使用 UTXO 集。
4. UTXOSet.FindUTXO - 类似 Blockchain.FindUTXO，但是使用 UTXO 集。
5. Blockchain.FindTransaction 跟之前一样。


##### 同步 UTXO集和数据库中的区块链
有了 UTXO 集，也就意味着我们的数据（交易）现在已经被分开存储：实际交易被存储在区块链中，未花费输出被存储在 UTXO 集中。想要 UTXO 集时刻处于最新状态，并且存储最新交易的输出，就需要一个良好的同步机制

###### 同步机制
1. 当挖出一个新块时，应该更新 UTXO 集（Update 方法）
2. 并移除已花费输出，且从新挖出来的交易中加入未花费输出
3. 如果一笔交易的输出被移除，并且不再包含任何输出，那么这笔交易也应该被移除


###  Merkle 树
比特币用 Merkle 树来获取交易哈希，哈希被保存在区块头中，并会用于工作量证明系统。

#### Merkle 树特点
1. 每个块都会有一个 Merkle 树，它从叶子节点（树的底部）开始，一个叶子节点就是一个交易哈希（比特币使用双 SHA256 哈希）
2. 叶子节点的数量必须是双数，但是并非每个块都包含了双数的交易。因为，如果一个块里面的交易数为单数，那么就将最后一个叶子节点（也就是 Merkle 树的最后一个交易，不是区块的最后一笔交易）复制一份凑成双数
3. 从下往上，两两成对，连接两个节点哈希，将组合哈希作为新的哈希。新的哈希就成为新的树节点。
3. 根哈希然后就会当做是整个块交易的唯一标示，将它保存到区块头，然后用于工作量证明。

#### Merkle 树的好处
Merkle 树的好处就是一个节点可以在不下载整个块的情况下，验证是否包含某笔交易。并且这些只需要一个交易哈希，一个 Merkle 树根哈希和一个 Merkle 路径。