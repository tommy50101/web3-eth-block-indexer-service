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
	lastestDbTxId    uint64
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
	db.Last(&lastDbBlock)
	lastestDbBlockId = lastDbBlock.ID
	blockId := lastestDbBlockId + 1

	lastDbTx := Transaction{}
	db.Last(&lastDbTx)
	lastestDbTxId = lastDbTx.ID
	txId := lastestDbTxId + 1

	blockModels := []Block{}
	transactionModels := []Transaction{}
	logModels := []Log{}

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
			ID:         blockId,
			BlockNum:   block.Number().Uint64(),
			BlockHash:  block.Hash().Hex(),
			BlockTime:  block.Time(),
			ParentHash: parentHash,
		}
		blockModels = append(blockModels, blockModel)

		fmt.Println("block_num:", block.Number().Uint64()) // 17102552
		fmt.Println("block_hash:", block.Hash().Hex())     // 0x9e8751ebb5069389b855bba72d94902cc385042661498a415979b7b6ee9ba4b9
		fmt.Println("block_time:", block.Time())           // 1527211625
		fmt.Println("parent_hash:", parentHash)            // 17102551

		for _, tx := range block.Transactions() {
			var to string
			if tx.To() == nil {
				to = ""
			} else {
				to = tx.To().Hex()
			}

			// Arrange txs data for insertion
			from, _ := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
			transactionModel := Transaction{
				ID:      txId,
				TxHash:  tx.Hash().Hex(),
				From:    from.Hex(),
				To:      to,
				Nonce:   tx.Nonce(),
				Data:    tx.Data(),
				Value:   tx.Value().String(),
				BlockID: blockId,
			}
			transactionModels = append(transactionModels, transactionModel)

			fmt.Println("tx_hash:", tx.Hash().Hex())   // 0x9c775841600aca0e9b4d8f1c87e3ae4fd52618d53bcc734558544c1165a6b416
			fmt.Println("from:", from.Hex())           // 0xae2Fc483527B8EF99EB5D9B44875F005ba1FaE13
			fmt.Println("to:", tx.To().Hex())          // 0x6b75d8AF000000e20B7a7DDf000Ba900b4009A80
			fmt.Println("nonce:", tx.Nonce())          // 248674
			fmt.Println("data:", tx.Data())            //
			fmt.Println("value:", tx.Value().String()) // [169 5 156 187 0 0 0 0 0 0 0 0 0 0 0 0]

			// Arrange logs data for insertion
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
					TransactionID: transactionModel.ID,
				}
				logModels = append(logModels, logModel)
			}
			txId = txId + 1
		}
		blockId = blockId + 1

		// Batch insert blocks, txs and logs, then exit when reached latest block
		nextIndex := startIndex.Add(startIndex, big.NewInt(1))
		nextBlock, _ := client.BlockByNumber(context.Background(), nextIndex)
		if nextBlock != nil {
			block = nextBlock
		} else {
			// Start to batch insert all data
			db.Create(&blockModels)
			db.Create(&transactionModels)
			db.Create(&logModels)

			log.Fatal("Has reached the latest block")
		}
	}
}

// 連線Db
func initDb() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local", DB_USERNAME, DB_PWD, DB_HOST, DB_PORT, DB_NAME)
	db, _ = gorm.Open(mysql.Open(dsn), &gorm.Config{})
}
