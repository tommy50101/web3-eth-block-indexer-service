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
	sRpc                 string
	err                  error
	blockOffset          int64
	startIndex           *big.Int
	client               *ethclient.Client
	block                *types.Block
	dbName               string
	db                   *gorm.DB
	lastestDbBlockId     int
	lastestDbTxId        int
	waitTxGoroutineTime  float64
	waitLogGoroutineTime float64
	waitNextBlockTime    float64
)

func main() {
	checkArgs()
	initDb()
	initRpc()
	initStartBlock()

	for {
		fmt.Printf("新增區塊 %d 中...\n", block.Number().Uint64())

		// 新增區塊
		blockId := insertBlock(block)

		// 多線程執行新增交易紀錄
		for _, tx := range block.Transactions() {
			go insertTxAndLogs(tx, blockId)
		}
		time.Sleep(time.Duration(waitTxGoroutineTime) * time.Second)

		fmt.Printf("新增區塊 %d 完畢\n\n", block.Number().Uint64())

		// 執行 or 等待，下一個區塊
		processOrWaitNextBlock()
	}
}

func checkArgs() {
	// 輸入參數判斷
	// 判斷哪個鏈
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("請輸入要在哪個鏈上執行: 1.BSC testnet  2.Ethereum testnet(goerli)  3.Ethereum mainnet")
	fmt.Print("-> ")
	text, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	if text[:len(text)-1] != "1" && text[:len(text)-1] != "2" && text[:len(text)-1] != "3" {
		log.Fatal("不合法的輸入")
	}

	// 根據選擇不同的鏈，有不同的等待參數、數據庫、RPC節點
	if text[:len(text)-1] == "1" {
		sRpc = "https://data-seed-prebsc-2-s3.binance.org:8545/"
		dbName = "bsc_testnet"
		waitTxGoroutineTime = 0.5
		waitLogGoroutineTime = 0.5
		waitNextBlockTime = 1.0
	} else if text[:len(text)-1] == "2" {
		sRpc = "https://goerli.infura.io/v3/84a99a188f8e4aaab60c45f9955c5d6b"
		dbName = "eth_testnet_goerli"
		waitTxGoroutineTime = 3.0
		waitLogGoroutineTime = 0.5
		waitNextBlockTime = 12.0
	} else {
		sRpc = "https://mainnet.infura.io/v3/84a99a188f8e4aaab60c45f9955c5d6b"
		dbName = "eth_mainnet"
		waitTxGoroutineTime = 3.0
		waitLogGoroutineTime = 0.5
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

func initRpc() {
	// Get latest block header and caculate the start block
	client, err = ethclient.Dial(sRpc)
	if err != nil {
		fmt.Println("json-rpc server connection failed")
		return
	}
}

func initStartBlock() {
	latestHeader, err := client.HeaderByNumber(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("鏈上最新區塊:", latestHeader.Number.String())

	sLatestHeader := latestHeader.Number.String()
	iLatestHeader, _ := strconv.ParseInt(sLatestHeader, 10, 64)
	iStartIndex := iLatestHeader - blockOffset
	startIndex = big.NewInt(iStartIndex)

	fmt.Printf("程式起始區塊: %d\n\n", startIndex)

	// Get start block by header number
	block, err = client.BlockByNumber(context.Background(), startIndex)
	if err != nil {
		log.Fatal(err)
	}
}

// 新增Block
func insertBlock(block *types.Block) int {
	blockModel := Block{
		BlockNum:   block.Number().Uint64(),
		BlockHash:  block.Hash().Hex(),
		BlockTime:  block.Time(),
		ParentHash: block.ParentHash().Hex(),
	}
	db.Create(&blockModel)
	return blockModel.ID
}

// 新增Tx及其Logs
func insertTxAndLogs(tx *types.Transaction, blockId int) {
	// 新增Tx
	from, _ := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
	var to string
	if tx.To() == nil {
		to = ""
	} else {
		to = tx.To().Hex()
	}
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
	if err != nil || len(receipt.Logs) == 0 {
		return
	}
	// 多線程執行新增Logs
	for _, log := range receipt.Logs {
		go insertLog(log, transactionModel.ID)
	}
	time.Sleep(time.Duration(waitLogGoroutineTime) * time.Second)
}

// 新增Log
func insertLog(log *types.Log, transactionId int) {
	logModel := Log{
		Index:         log.Index,
		Data:          log.Data,
		TransactionID: transactionId,
	}
	db.Create(&logModel)
}

func processOrWaitNextBlock() {
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
