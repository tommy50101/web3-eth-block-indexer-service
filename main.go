package main

import (
	"strconv"

	"github.com/ethereum/go-ethereum/core/types"

	// "math/big"
	"context"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/joho/godotenv/autoload"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Block struct {
	// gorm.Model
	ID         uint64
	BlockNum   uint64
	BlockHash  string
	BlockTime  uint64
	ParentHash string
}

type Transaction struct {
	gorm.Model
	ID      uint64
	TxHash  string
	From    string
	To      string
	Nonce   uint64
	Data    []byte
	Value   string
	BlockID uint64
}
type Log struct {
	gorm.Model
	Index         uint
	Data          []byte
	TransactionID uint64
}

var (
	db *gorm.DB
)

func (block Block) TableName() string {
	// 绑定MYSQL表名為block
	return "block"
}

func (transaction Transaction) TableName() string {
	// 绑定MYSQL表名為transaction
	return "transaction"
}

func (log Log) TableName() string {
	// 绑定MYSQL表名為log
	return "log"
}

func main() {
	initDb()
	client, err := ethclient.Dial("https://mainnet.infura.io/v3/" + os.Getenv("INFURA_ETH_MAIN_KEY"))
	if err != nil {
		fmt.Println("json-rpc server connection failed")
		return
	}

	// Get latest block header
	latestHeader, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(latestHeader.Number)

	// 參數代表從第幾個區塊開始，不給預設最新區塊的前10個開始
	arg := os.Args
	var startIndex *big.Int
	if len(arg) != 1 {
		i, _ := strconv.ParseInt(os.Args[1], 10, 64)
		startIndex = big.NewInt(i)
	} else {
		sLatestHeader := latestHeader.Number.String()
		iLatestHeader, _ := strconv.ParseInt(sLatestHeader, 10, 64)
		iStartIndex := iLatestHeader - 10
		startIndex = big.NewInt(iStartIndex)
	}

	// Get latest block by header number
	block, err := client.BlockByNumber(context.Background(), startIndex)
	if err != nil {
		log.Fatal(err)
	}

	// Get parent_hash by parent_num
	parentNumber := big.NewInt((block.Number().Int64() - 1))
	parentBlock, err := client.BlockByNumber(context.Background(), parentNumber)
	if err != nil {
		log.Fatal(err)
	}
	parentHash := parentBlock.Hash().Hex()

	fmt.Println("block_num:", block.Number().Uint64()) // 17102552
	fmt.Println("block_hash:", block.Hash().Hex())     // 0x9e8751ebb5069389b855bba72d94902cc385042661498a415979b7b6ee9ba4b9
	fmt.Println("block_time:", block.Time())           // 1527211625
	fmt.Println("parent_hash:", parentHash)            // 17102551

	// Insert block
	blockModel := Block{
		BlockNum:   block.Number().Uint64(),
		BlockHash:  block.Hash().Hex(),
		BlockTime:  block.Time(),
		ParentHash: parentHash,
	}
	db.Where(Block{BlockNum: block.Number().Uint64()}).FirstOrCreate(&blockModel)
	blockId := blockModel.ID

	// Txs
	for _, tx := range block.Transactions() {
		from, _ := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)

		fmt.Println("tx_hash:", tx.Hash().Hex())   // 0x7104e3d31ed4f82278b7b03a40661f289d95ad10385543c4cdf8755a678af8a2
		fmt.Println("from:", from.Hex())           // 0xB55Ff2eafcE9Efb883d0E6Ad27a286AA875dBED2
		fmt.Println("to:", tx.To().Hex())          // 0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D
		fmt.Println("nonce:", tx.Nonce())          // 79
		fmt.Println("data:", tx.Data())            // [24 203 175 229 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]
		fmt.Println("value:", tx.Value().String()) // 10000000000000000

		// Insert tx
		transactionModel := Transaction{
			TxHash:  tx.Hash().Hex(),
			From:    from.Hex(),
			To:      tx.To().Hex(),
			Nonce:   tx.Nonce(),
			Data:    tx.Data(),
			Value:   tx.Value().String(),
			BlockID: blockId,
		}
		db.Where(Transaction{TxHash: tx.Hash().Hex()}).FirstOrCreate(&transactionModel)
		transactionId := transactionModel.ID

		// Get logs from tx receipt
		receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
		if err != nil {
			log.Fatal(err)
		}

		if len(receipt.Logs) == 0 {
			continue
		}

		// Logs
		for _, log := range receipt.Logs {
			fmt.Println("log_index:", log.Index) // 1
			fmt.Println("log_data:", log.Data)   // [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 232 147 174 5 126 221 103 122 0]

			// Insert log
			logModel := Log{
				Index:         log.Index,
				Data:          log.Data,
				TransactionID: transactionId,
			}
			db.Where(Log{Index: log.Index, Data: log.Data}).FirstOrCreate(&logModel)
		}
	}
}

func initDb() {
	// 連線Db
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local", DB_USERNAME, DB_PWD, DB_HOST, DB_PORT, DB_NAME)
	db, _ = gorm.Open(mysql.Open(dsn), &gorm.Config{})
}
