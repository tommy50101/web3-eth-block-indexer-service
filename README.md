# Ehh-Block-Indexer-Service

## Instructions

透過 go-ethereum 將區塊內的資料掃進 db

## Setup

```zsh
$ go get .
```

## Start

```zsh
// 1. 指定開始區塊
$ go run . <block＿number>

// example:
$ go run . 17114151

// 2. 不指定開始區塊(預設最新區塊的前第10個開始)
$ go run .
```

## Usage

啟動後即會開始從指定起始block_number位置，依序掃到最新區塊，掃描鏈上block、transactions、logs的資訊，並存入資料庫
