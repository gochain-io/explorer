package models

import "time"

type Transaction struct {
	TxHash      string    `json:"tx_hash" bson:"tx_hash"`
	To          string    `json:"to" bson:"to"`
	From        string    `json:"from" bson:"from"`
	Value       string    `json:"value" bson:"value"`
	GasPrice    string    `json:"gas_price" bson:"gas_price"`
	GasFee      string    `json:"gas_fee" bson:"gas_fee"`
	GasLimit    uint64    `json:"gas_limit" bson:"gas_limit"`
	BlockNumber int64     `json:"block_number" bson:"block_number"`
	Nonce       string    `json:"nonce" bson:"nonce"`
	BlockHash   string    `json:"block_hash" bson:"hash"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
	InputData   string    `json:"input_data" bson:"input_data"`
}

type TransactionList struct {
	Transactions []*Transaction `json:"transactions"`
}
