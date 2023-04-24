# Ehh-Block-Indexer-Service

## Instructions

透過 go-ethereum 將區塊內的資料掃進 db

## Setup

```zsh
1. $ go get .
2. 在 .env 檔內加上 Infura 的 eth main net 的 key
```

## Start

```zsh
1. 不指定開始區塊(預設最新區塊的前第10個開始)
$ go run .

2. 指定開始區塊
$ go run . <block＿number>

example:
$ go run . 17114151
```

## Usage

啟動後即會開始從指定起始block_number位置，依序掃到最新區塊，掃描鏈上block、transactions、logs的資訊，並存入資料庫
