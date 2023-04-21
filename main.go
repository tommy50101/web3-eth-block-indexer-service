package main

import (
	_ "github.com/joho/godotenv/autoload"
	"os"
    "fmt"
    "github.com/ethereum/go-ethereum/ethclient"
)

func main() {
	key := os.Getenv("INFURA_ETH_MAIN_KEY")
    client, err := ethclient.Dial("https://mainnet.infura.io/v3/"+key )
    if err != nil {
        fmt.Println("connection failed")
		return
    }

	fmt.Println(*client, err);
}
