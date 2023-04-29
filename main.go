package main

import (
	"bufio"
	"strconv"
	"time"

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
	"gorm.io/gorm/logger"
)

var (
	blockOffset      int64
	startIndex       *big.Int
	client           *ethclient.Client
	err              error
	db               *gorm.DB
	lastestDbBlockId int
	lastestDbTxId    int
)

func main() {
	// 輸入參數判斷
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("請輸入一正整數 n ，程式將從最新區塊的前 n 個區塊開始獲取 (不輸入則預設 n = 10):")
	fmt.Print("-> ")
	text, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	if len(text) == 1 {
		// 沒輸入，預設前10個區塊
		blockOffset = 10
	} else {
		// 輸入n，則從最新區塊-n個區塊開始跑
		content := text[:len(text)-1]
		blockOffset, err = strconv.ParseInt(content, 10, 64)
		if err != nil {
			log.Fatal("錯誤的輸入: ", err)
		}
	}

	// Get latest block header and caculate the start block
	client, err = ethclient.Dial("https://mainnet.infura.io/v3/" + os.Getenv("INFURA_ETH_MAIN_KEY"))
	if err != nil {
		fmt.Println("json-rpc server connection failed")
		return
	}
	latestHeader, err := client.HeaderByNumber(context.Background(), nil)
	fmt.Println("鏈上最新區塊:", latestHeader.Number.String())
	if err != nil {
		log.Fatal(err)
	}
	sLatestHeader := latestHeader.Number.String()
	iLatestHeader, _ := strconv.ParseInt(sLatestHeader, 10, 64)
	iStartIndex := iLatestHeader - blockOffset
	startIndex = big.NewInt(iStartIndex)
	fmt.Println("程式起始區塊:", startIndex)

	// Get block by header number
	block, err := client.BlockByNumber(context.Background(), startIndex)
	if err != nil {
		log.Fatal(err)
	}

	initDb()

	for {
		// Get parent_hash by parent_num
		parentNumber := big.NewInt((block.Number().Int64() - 1))
		parentBlock, err := client.BlockByNumber(context.Background(), parentNumber)
		if err != nil {
			log.Fatal(err)
		}
		parentHash := parentBlock.Hash().Hex()

		// Arrange blocks data for insertion
		blockModel := Block{
			BlockNum:   block.Number().Uint64(),
			BlockHash:  block.Hash().Hex(),
			BlockTime:  block.Time(),
			ParentHash: parentHash,
		}
		db.Create(&blockModel)

		fmt.Println("新稱區塊中，block_num:", block.Number().Uint64())

		// 多線程執行新增交易紀錄
		for _, tx := range block.Transactions() {
			// Insert txs and logs
			go insertTxsAndLogs(tx, blockModel.ID)
		}
		time.Sleep(3 * time.Second)

		// 執行 or 等待，下一個區塊
		nextIndex := startIndex.Add(startIndex, big.NewInt(1))
		for {
			nextBlock, _ := client.BlockByNumber(context.Background(), nextIndex)
			if nextBlock != nil {
				// Next block round
				block = nextBlock
				break
			} else {
				// 等待下一區塊
				time.Sleep(6 * time.Second)
				fmt.Printf("等待下一個最新區塊: %d 產生中\n", block.Number().Uint64()+1)
				time.Sleep(6 * time.Second)
			}
		}
	}
}

// 連線Db & 初始化gorm
func initDb() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local", DB_USERNAME, DB_PWD, DB_HOST, DB_PORT, DB_NAME)
	db, _ = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}

// 插入Txs和Logs
func insertTxsAndLogs(tx *types.Transaction, blockId int) {
	// Prevent invalid memory address or nil pointer dereference
	var to string
	if tx.To() == nil {
		to = ""
	} else {
		to = tx.To().Hex()
	}

	// Insert txs
	from, _ := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
	transactionModel := Transaction{
		TxHash:  tx.Hash().Hex(),
		From:    from.Hex(),
		To:      to,
		Nonce:   tx.Nonce(),
		Data:    tx.Data(),
		Value:   tx.Value().String(),
		BlockID: blockId,
	}
	db.Create(&transactionModel)

	fmt.Println("新增Transaction中，tx_hash:", tx.Hash().Hex())

	// Get logs from TransactionReceipt
	receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		log.Fatal(err)
	}

	if len(receipt.Logs) == 0 {
		return
	}

	// Batch insert logs
	logModels := []Log{}
	for _, log := range receipt.Logs {
		logModel := Log{
			Index:         log.Index,
			Data:          log.Data,
			TransactionID: transactionModel.ID,
		}
		logModels = append(logModels, logModel)

		fmt.Println("新增Log中，log_index:", log.Index)
	}
	db.Create(&logModels)
}
