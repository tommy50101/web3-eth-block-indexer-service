# Ehh-Block-Indexer-Service

## Instructions

透過 go-ethereum 將區塊內的資料掃進 db

## Setup

```zsh
$ go get .
```

## Start

```zsh
1. $ cd web3-eth-block-indexer-service/

2. $ go run .

3. 輸入第一個參數，值為1或2，指定於哪條鏈上作業。 1:BSC testnet 2:Ethereum testnet(Goerli)

4. 輸入第二個參數，值為任意正整數，決定程式從最新區塊的前面第 n 個區塊開始獲取資料。 若沒輸入則預設 n = 10
```

## Usage

啟動後即會依照你指定的鏈，從最新前n個區塊開始，掃描鏈上block、transactions、logs的資訊，並存入資料庫，一直掃到最新區塊，並繼續持續獲取新產出的區塊。

不同鏈的資料會記錄到不同的庫，但都存在於同一個已架好的AWS RDS實例裡，連線資訊都紀錄於config.go，也可透過任何GUI直接連線過去確認數據
