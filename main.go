package main

import (
	"strconv"
	// "github.com/ethereum/go-ethereum/core/types"
	// "math/big"
	"log"
	_ "github.com/joho/godotenv/autoload"
	"os"
    "fmt"
    "github.com/ethereum/go-ethereum/ethclient"
	"context"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"math/big"
)

type Block struct {
	// gorm.Model
	ID           uint
	BlockNum     uint64
	BlockHash    string
	BlockTime    uint64
	ParentHash   string
}

var(
	db *gorm.DB
)

func (block Block) TableName() string {
	// 绑定MYSQL表名為block
	return "block"
}

func main() {
	initDb()
	client, err := ethclient.Dial("https://mainnet.infura.io/v3/" + os.Getenv("INFURA_ETH_MAIN_KEY") )
    if err != nil {
        fmt.Println("json-rpc server connection failed")
		return
    }

	// Get latest block header
	latestHeader, err := client.HeaderByNumber(context.Background(), nil)
    if err != nil {
        log.Fatal(err)
    }
    // fmt.Println("Latest block header:" + latestHeader.Number.String())

	// Get block by header number
    // blockNumber := big.NewInt(17102552)
    block, err := client.BlockByNumber(context.Background(), latestHeader.Number)
    if err != nil {
        log.Fatal(err)
    }
	fmt.Println("block_num:", block.Number().Uint64())     // 17102552
	fmt.Println("block_hash:", block.Hash().Hex())          // 0x9e8751ebb5069389b855bba72d94902cc385042661498a415979b7b6ee9ba4b9
    fmt.Println("block_time:", block.Time())       // 1527211625
	fmt.Println("parent_num:", block.Number().Uint64() - 1 )

	parentNumber := big.NewInt((block.Number().Int64() - 1))

	parentBlock, err := client.BlockByNumber(context.Background(), parentNumber)
    if err != nil {
        log.Fatal(err)
    }
	parentHash := parentBlock.Hash().Hex()

	blockModel := Block{
		BlockNum: block.Number().Uint64(),
		BlockHash: block.Hash().Hex(),
		BlockTime: block.Time(),
		ParentHash: parentHash,
	}
	db.Create(&blockModel)
}

func initDb() {
	username := os.Getenv("DB_USERNAME")
	password := os.Getenv("DB_PWD")
	host := os.Getenv("DB_HOST")
	port, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	Dbname := os.Getenv("DB_NAME")
	// 連線Db
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8&parseTime=True&loc=Local", username, password, host, port, Dbname)
	db, _ = gorm.Open(mysql.Open(dsn), &gorm.Config{})
}
