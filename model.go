package main

import "gorm.io/gorm"

type Block struct {
	// gorm.Model
	ID         int
	BlockNum   uint64
	BlockHash  string
	BlockTime  uint64
	ParentHash string
}

type Transaction struct {
	gorm.Model
	ID      int
	TxHash  string
	From    string
	To      string
	Nonce   uint64
	Data    []byte
	Value   string
	BlockID int
}

type Log struct {
	gorm.Model
	Index         uint
	Data          []byte
	TransactionID int
}

func (block Block) TableName() string {
	// 绑定MYSQL表名為block
	return "block"
}

func (transaction Transaction) TableName() string {
	// 绑定MYSQL表名為transaction
	return "transaction"
}

func (log Log) TableName() string {
	// 绑定MYSQL表名為log
	return "log"
}
