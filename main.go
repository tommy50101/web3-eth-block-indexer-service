package main

import (
	"strconv"

	"context"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	_ "github.com/joho/godotenv/autoload"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	client           *ethclient.Client
	err              error
	db               *gorm.DB
	lastestDbBlockId uint64
	blockId          uint64
)

func main() {
	initDb()
	client, err = ethclient.Dial("https://mainnet.infura.io/v3/" + os.Getenv("INFURA_ETH_MAIN_KEY"))
	if err != nil {
		fmt.Println("json-rpc server connection failed")
		return
	}

	// 參數代表從第幾個區塊開始，不給預設最新區塊的前10個開始
	arg := os.Args
	var startIndex *big.Int
	if len(arg) != 1 {
		i, _ := strconv.ParseInt(os.Args[1], 10, 64)
		startIndex = big.NewInt(i)
	} else {
		// Get latest block header
		latestHeader, err := client.HeaderByNumber(context.Background(), nil)
		if err != nil {
			log.Fatal(err)
		}
		sLatestHeader := latestHeader.Number.String()
		iLatestHeader, _ := strconv.ParseInt(sLatestHeader, 10, 64)
		iStartIndex := iLatestHeader - 10
		startIndex = big.NewInt(iStartIndex)
	}

	// Get block by header number
	block, err := client.BlockByNumber(context.Background(), startIndex)
	if err != nil {
		log.Fatal(err)
	}

	lastDbBlock := Block{}
	db.Order("id DESC").First(&lastDbBlock)
	lastestDbBlockId = lastDbBlock.ID
	blockId = lastestDbBlockId + 1

	blockModels := []Block{}
	transactionModels := []Transaction{}
	for {
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
			ID:         blockId,
			BlockNum:   block.Number().Uint64(),
			BlockHash:  block.Hash().Hex(),
			BlockTime:  block.Time(),
			ParentHash: parentHash,
		}
		blockModels = append(blockModels, blockModel)

		// Arrange txs data for insertion
		for _, tx := range block.Transactions() {
			fmt.Println("block_num:", block.Number().Uint64()) // 17102552
			fmt.Println("block_hash:", block.Hash().Hex())     // 0x9e8751ebb5069389b855bba72d94902cc385042661498a415979b7b6ee9ba4b9
			fmt.Println("block_time:", block.Time())           // 1527211625
			fmt.Println("parent_hash:", parentHash)            // 17102551

			if tx.To() == nil {
				continue
			}
			from, _ := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
			transactionModel := Transaction{
				TxHash:  tx.Hash().Hex(),
				From:    from.Hex(),
				To:      tx.To().Hex(),
				Nonce:   tx.Nonce(),
				Data:    tx.Data(),
				Value:   tx.Value().String(),
				BlockID: blockId,
			}
			transactionModels = append(transactionModels, transactionModel)
		}

		// Batch insert blocks, txs then exit when reached latest block
		nextIndex := startIndex.Add(startIndex, big.NewInt(1))
		nextBlock, _ := client.BlockByNumber(context.Background(), nextIndex)
		if nextBlock != nil {
			blockId = blockId + 1
			block = nextBlock
		} else {
			db.Create(&blockModels)
			insertTxs(transactionModels)

			log.Fatal("Has reached the latest block")
		}
	}
}

// 連線Db
func initDb() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local", DB_USERNAME, DB_PWD, DB_HOST, DB_PORT, DB_NAME)
	db, _ = gorm.Open(mysql.Open(dsn), &gorm.Config{})
}

// Insert txs
func insertTxs(transactionModels []Transaction) {
	db.Create(&transactionModels)

	// Get logs from tx receipt
	// receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// if len(receipt.Logs) == 0 {
	// 	continue
	// }

	// // Logs
	// for _, log := range receipt.Logs {
	// 	fmt.Println("log_index:", log.Index) // 1
	// 	fmt.Println("log_data:", log.Data)   // [0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 232 147 174 5 126 221 103 122 0]
	// 	// Insert log
	// 	logModel := Log{
	// 		Index:         log.Index,
	// 		Data:          log.Data,
	// 		TransactionID: transactionId,
	// 	}
	// 	db.Where(Log{Index: log.Index, Data: log.Data}).FirstOrCreate(&logModel)
	// }

}
