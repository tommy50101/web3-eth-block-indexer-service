# Ehh-Block-Indexer-Service

## Instructions

透過 go-ethereum 將區塊內的資料掃進 db

## Setup

```zsh
$ go get .
```

## Start

```zsh
$ go run .
```

## Usage

- 啟動後，要輸入兩次參數:

1. 第一個參數，值為1或2，指定於哪條鏈上作業。 1:BSC testnet &nbsp;&nbsp; 2:Ethereum testnet(Goerli) &nbsp;&nbsp;&nbsp;&nbsp; 3:Ethereum mainnet

2. 第二個參數，值為任意正整數，決定程式從最新區塊的前面第 n 個區塊開始獲取資料。 若沒輸入則預設 n = 10<br/>
<br/>
- 輸入完參數後，即會依照你指定的鏈，從最新前n個區塊開始，掃描鏈上的blocks、transactions、logs的資訊，並存入資料庫，一直掃到最新區塊，並持續獲取新產出的區塊。<br/>
<br/>
- 不同鏈的資料會記錄到不同的庫，但都存在於同一個已架好的AWS RDS實例裡，連線資訊都紀錄於config.go，也可透過任何GUI直接連線過去確認數據。<br/>
<br/>
- 若遇到分叉現象，會記錄目前分支上的區塊為不穩定，直到後續接在同一分支的區塊數量大於6個，才會轉換為穩定狀態。<br/>
<br/>
