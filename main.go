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
	sRpc                string
	blockOffset         int64
	startIndex          *big.Int
	client              *ethclient.Client
	err                 error
	dbName              string
	db                  *gorm.DB
	lastestDbBlockId    int
	lastestDbTxId       int
	waitTxGoroutineTime float64
	waitNextBlockTime   float64
)

func main() {
	checkParams()
	initDb()

	// Get latest block header and caculate the start block
	client, err = ethclient.Dial(sRpc)
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

	for {
		// Arrange blocks data for insertion
		fmt.Println("新增區塊中，block_num:", block.Number().Uint64())
		blockModel := Block{
			BlockNum:   block.Number().Uint64(),
			BlockHash:  block.Hash().Hex(),
			BlockTime:  block.Time(),
			ParentHash: block.ParentHash().Hex(),
		}
		db.Create(&blockModel)

		// 多線程執行新增交易紀錄
		for _, tx := range block.Transactions() {
			// Insert txs and logs
			go insertTxsAndLogs(tx, blockModel.ID)
		}
		time.Sleep(time.Duration(waitTxGoroutineTime) * time.Second)

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
				time.Sleep(time.Duration(waitNextBlockTime/2) * time.Second)
				fmt.Printf("等待下一個最新區塊: %d 產生中...\n", block.Number().Uint64()+1)
				time.Sleep(time.Duration(waitNextBlockTime/2) * time.Second)
			}
		}
	}
}

func checkParams() {
	// 輸入參數判斷
	// 判斷哪個鏈
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("請輸入要在哪個鏈上執行: 1.BSC testnet  2.Ethereum testnet(goerli) ")
	fmt.Print("-> ")
	text, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	if text[:len(text)-1] != "1" && text[:len(text)-1] != "2" {
		log.Fatal("不合法的輸入")
	}

	// 根據選擇不同的鏈，有不同的等待參數、數據庫、RPC節點
	if text[:len(text)-1] == "1" {
		sRpc = "https://data-seed-prebsc-2-s3.binance.org:8545/"
		dbName = "bsc_testnet"
		waitTxGoroutineTime = 0.5
		waitNextBlockTime = 1.0
	} else {
		sRpc = "https://goerli.infura.io/v3/84a99a188f8e4aaab60c45f9955c5d6b"
		dbName = "eth_testnet_goerli"
		waitTxGoroutineTime = 3.0
		waitNextBlockTime = 12.0
	}

	// 判斷從最新的前n個區塊開始
	fmt.Println("請輸入一正整數 n ，程式將從最新區塊的前 n 個區塊開始獲取 (不輸入則預設 n = 10):")
	fmt.Print("-> ")
	text2, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	if len(text2) == 1 {
		// 沒輸入，預設前10個區塊
		blockOffset = 10
	} else {
		// 輸入n，則從最新區塊-n個區塊開始跑
		content := text2[:len(text2)-1]
		blockOffset, err = strconv.ParseInt(content, 10, 64)
		if err != nil {
			log.Fatal("錯誤的輸入: ", err)
		}
	}
}

// 連線Db & 初始化gorm
func initDb() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local", DB_USERNAME, DB_PWD, DB_HOST, DB_PORT, dbName)
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
	// fmt.Println("新增Transaction中，tx_hash:", tx.Hash().Hex())
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

	// Get logs from TransactionReceipt
	receipt, err := client.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		return
	}

	if len(receipt.Logs) == 0 {
		return
	}

	// Batch insert logs
	logModels := []Log{}
	for _, log := range receipt.Logs {
		// fmt.Println("新增Log中，log_index:", log.Index)
		logModel := Log{
			Index:         log.Index,
			Data:          log.Data,
			TransactionID: transactionModel.ID,
		}
		logModels = append(logModels, logModel)
	}
	db.Create(&logModels)
}
