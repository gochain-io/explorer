package backend

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/gochain-io/explorer/server/models"
	"github.com/gochain-io/gochain/common"
	"github.com/gochain-io/gochain/core/types"
	"github.com/gochain-io/gochain/ethclient"
)

var wei = big.NewInt(1000000000000000000)

func appendIfMissing(slice []string, i string) []string {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}

type MongoBackend struct {
	host      string
	mongo     *mgo.Database
	ethClient *ethclient.Client
}

// New create new rpc client with given url
func NewMongoClient(host, rpcUrl string) *MongoBackend {
	client, err := ethclient.Dial(rpcUrl)
	if err != nil {
		log.Fatal().Err(err).Msg("main")
	}
	Host := []string{
		host,
	}
	session, err := mgo.DialWithInfo(&mgo.DialInfo{
		Addrs: Host,
	})
	if err != nil {
		panic(err)
	}

	importer := new(MongoBackend)

	importer.mongo = session.DB("blocks")
	importer.ethClient = client
	importer.createIndexes()

	return importer

}
func (self *MongoBackend) parseTx(tx *types.Transaction, block *types.Block) *models.Transaction {
	from, err := self.ethClient.TransactionSender(context.Background(), tx, block.Header().Hash(), 0)
	if err != nil {
		log.Fatal().Err(err).Msg("parseTx")
	}
	gas := tx.Gas()
	to := ""
	if tx.To() != nil {
		to = tx.To().Hex()
	}
	log.Debug().Interface("TX:", tx).Msg("parseTx")
	return &models.Transaction{TxHash: tx.Hash().Hex(),
		To:          to,
		From:        from.Hex(),
		Value:       tx.Value().String(),
		GasPrice:    tx.GasPrice().String(),
		GasLimit:    tx.Gas(),
		BlockNumber: block.Number().Int64(),
		GasFee:      new(big.Int).Mul(tx.GasPrice(), big.NewInt(int64(gas))).String(),
		Nonce:       strconv.Itoa(int(tx.Nonce())),
		BlockHash:   block.Hash().Hex(),
		CreatedAt:   time.Unix(block.Time().Int64(), 0),
		InputData:   string(tx.Data()[:]),
	}
}
func (self *MongoBackend) parseBlock(block *types.Block) *models.Block {
	var transactions []string
	for _, tx := range block.Transactions() {
		transactions = append(transactions, tx.Hash().Hex())
	}
	return &models.Block{Number: block.Header().Number.Int64(),
		GasLimit:   int(block.Header().GasLimit),
		BlockHash:  block.Hash().Hex(),
		CreatedAt:  time.Unix(block.Time().Int64(), 0),
		ParentHash: block.ParentHash().Hex(),
		TxHash:     block.Header().TxHash.Hex(),
		GasUsed:    strconv.Itoa(int(block.Header().GasUsed)),
		Nonce:      strconv.Itoa(int((block.Nonce()))),
		Miner:      block.Coinbase().Hex(),
		TxCount:    int(uint64(len(block.Transactions()))),
		Difficulty: block.Difficulty().Int64(),
		// TotalDifficulty: block.DeprecatedTd().Int64(), # deprecated https://github.com/ethereum/go-ethereum/blob/master/core/types/block.go#L154
		Sha3Uncles: block.UncleHash().Hex(),
		ExtraData:  string(block.Extra()[:]),
		// Transactions: transactions,
	}
}

func (self *MongoBackend) createIndexes() {
	err := self.mongo.C("Transactions").EnsureIndex(mgo.Index{Key: []string{"from"}, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}
	err = self.mongo.C("Transactions").EnsureIndex(mgo.Index{Key: []string{"to"}, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}
	err = self.mongo.C("Transactions").EnsureIndex(mgo.Index{Key: []string{"tx_hash"}, Unique: true, DropDups: true, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("Transactions").EnsureIndex(mgo.Index{Key: []string{"block_number"}, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("Blocks").EnsureIndex(mgo.Index{Key: []string{"number"}, Unique: true, DropDups: true, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("Blocks").EnsureIndex(mgo.Index{Key: []string{"-number"}, Background: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("Blocks").EnsureIndex(mgo.Index{Key: []string{"miner"}, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("Blocks").EnsureIndex(mgo.Index{Key: []string{"hash"}, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("ActiveAddress").EnsureIndex(mgo.Index{Key: []string{"updated_at"}, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}
	err = self.mongo.C("ActiveAddress").EnsureIndex(mgo.Index{Key: []string{"address"}, Unique: true, DropDups: true, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}
	err = self.mongo.C("Addresses").EnsureIndex(mgo.Index{Key: []string{"address"}, Unique: true, DropDups: true, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("Addresses").EnsureIndex(mgo.Index{Key: []string{"-balance_float"}, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("TokensHolders").EnsureIndex(mgo.Index{Key: []string{"contract_address", "token_holder_address"}, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("TokensHolders").EnsureIndex(mgo.Index{Key: []string{"balance_int"}, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("InternalTransactions").EnsureIndex(mgo.Index{Key: []string{"contract_address", "from_address", "to_address"}, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("InternalTransactions").EnsureIndex(mgo.Index{Key: []string{"block_number"}, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}
}

func (self *MongoBackend) importBlock(block *types.Block) *models.Block {
	log.Debug().Str("BlockNumber", block.Header().Number.String()).Str("Hash", block.Hash().Hex()).Str("ParentHash", block.ParentHash().Hex()).Msg("Importing block")
	b := self.parseBlock(block)
	log.Debug().Interface("Block", b)
	_, err := self.mongo.C("Blocks").Upsert(bson.M{"number": b.Number}, b)
	if err != nil {
		log.Fatal().Err(err).Msg("importBlock")
	}
	_, err = self.mongo.C("Transactions").RemoveAll(bson.M{"block_number": b.Number}) //deleting all txs belong to this block if any exist
	if err != nil {
		log.Fatal().Err(err).Msg("importBlock")
	}
	for _, tx := range block.Transactions() {
		self.importTx(tx, block)
	}
	_, err = self.mongo.C("ActiveAddress").Upsert(bson.M{"address": block.Coinbase().Hex()}, &models.ActiveAddress{Address: block.Coinbase().Hex(), UpdatedAt: time.Now()})
	if err != nil {
		log.Fatal().Err(err).Msg("importBlock")
	}
	return b

}
func (self *MongoBackend) importTx(tx *types.Transaction, block *types.Block) {
	log.Debug().Msg("Importing tx" + tx.Hash().Hex())
	transaction := self.parseTx(tx, block)
	_, err := self.mongo.C("Transactions").Upsert(bson.M{"tx_hash": tx.Hash().String()}, transaction)
	if err != nil {
		log.Fatal().Err(err).Msg("importTx")
	}
	_, err = self.mongo.C("ActiveAddress").Upsert(bson.M{"address": transaction.From}, &models.ActiveAddress{Address: transaction.From, UpdatedAt: time.Now()})
	if err != nil {
		log.Fatal().Err(err).Msg("importBlock")
	}

	_, err = self.mongo.C("ActiveAddress").Upsert(bson.M{"address": transaction.To}, &models.ActiveAddress{Address: transaction.To, UpdatedAt: time.Now()})
	if err != nil {
		log.Fatal().Err(err).Msg("importBlock")
	}
}
func (self *MongoBackend) needReloadBlock(blockNumber int64) bool {
	block := self.getBlockByNumber(blockNumber)
	if block == nil {
		log.Debug().Msg("Checking parent - main block not found")
		return true
	}
	parentBlockNumber := (block.Number - 1)
	parentBlock := self.getBlockByNumber(parentBlockNumber)
	if parentBlock != nil {
		log.Debug().Str("ParentHash", block.ParentHash).Str("Hash from parent", parentBlock.BlockHash).Int64("BlockNumber", block.Number).Int64("ParentNumber", parentBlock.Number).Msg("Checking parent")
	}
	return parentBlock == nil || (block.ParentHash != parentBlock.BlockHash)

}

func (self *MongoBackend) transactionsConsistent(blockNumber int64) bool {
	block := self.getBlockByNumber(blockNumber)
	if block != nil {
		transactionCounter, err := self.mongo.C("Transactions").Find(bson.M{"block_number": blockNumber}).Count()
		log.Debug().Int("Transactions in block", block.TxCount).Int("Num of transactions in db", transactionCounter).Msg("TransactionsConsistent")
		if err != nil {
			log.Fatal().Err(err).Msg("TransactionsConsistent")
		}
		return transactionCounter == block.TxCount
	}
	return true
}

func (self *MongoBackend) importAddress(address string, balance *big.Int, tokenName, tokenSymbol string, contract, go20 bool) *models.Address {
	balanceGoFloat, _ := new(big.Float).SetPrec(100).Quo(new(big.Float).SetInt(balance), new(big.Float).SetInt(wei)).Float64() //converting to GO from wei
	balanceGoString := new(big.Rat).SetFrac(balance, wei).FloatString(18)
	log.Info().Str("address", address).Str("precise balance", balanceGoString).Float64("balance float", balanceGoFloat).Msg("Updating address")
	tokenHoldersCounter, err := self.mongo.C("TokensHolders").Find(bson.M{"contract_address": address}).Count()
	if err != nil {
		log.Fatal().Err(err).Msg("importAddress")
	}

	internalTransactionsCounter, err := self.mongo.C("InternalTransactions").Find(bson.M{"contract_address": address}).Count()
	if err != nil {
		log.Fatal().Err(err).Msg("importAddress")
	}

	addressM := &models.Address{Address: address,
		BalanceWei:    balance.String(),
		LastUpdatedAt: time.Now(),
		TokenName:     tokenName,
		TokenSymbol:   tokenSymbol,
		Contract:      contract,
		GO20:          go20,
		BalanceFloat:  balanceGoFloat,
		BalanceString: balanceGoString,
		// NumberOfTransactions:         transactionCounter,
		NumberOfTokenHolders:         tokenHoldersCounter,
		NumberOfInternalTransactions: internalTransactionsCounter,
	}
	_, err = self.mongo.C("Addresses").Upsert(bson.M{"address": address}, addressM)
	if err != nil {
		log.Fatal().Err(err).Msg("importAddress")
	}
	return addressM

}

func (self *MongoBackend) importTokenHolder(contractAddress, tokenHolderAddress string, balance *big.Int, tokenName, tokenSymbol string) *models.TokenHolder {
	balanceInt := new(big.Int).Div(balance, wei) //converting to GO from wei
	log.Info().Str("contractAddress", contractAddress).Str("balance", balance.String()).Str("Balance int", balanceInt.String()).Msg("Updating token holder")
	tokenHolder := &models.TokenHolder{
		ContractAddress:    contractAddress,
		TokenHolderAddress: tokenHolderAddress,
		Balance:            balance.String(),
		UpdatedAt:          time.Now(),
		TokenName:          tokenName,
		TokenSymbol:        tokenSymbol,
		BalanceInt:         balanceInt.Int64()}
	_, err := self.mongo.C("TokensHolders").Upsert(bson.M{"contract_address": contractAddress, "token_holder_address": tokenHolderAddress}, tokenHolder)
	if err != nil {
		log.Fatal().Err(err).Msg("importTokenHolder")
	}
	return tokenHolder

}

func (self *MongoBackend) importInternalTransaction(contractAddress string, transferEvent TransferEvent) *models.InternalTransaction {

	internalTransaction := &models.InternalTransaction{
		Contract:        contractAddress,
		From:            transferEvent.From.String(),
		To:              transferEvent.To.String(),
		Value:           transferEvent.Value.String(),
		BlockNumber:     transferEvent.BlockNumber,
		TransactionHash: transferEvent.TransactionHash,
		UpdatedAt:       time.Now(),
	}
	_, err := self.mongo.C("InternalTransactions").Upsert(bson.M{"contract_address": contractAddress, "from_address": internalTransaction.From, "to_address": internalTransaction.To}, internalTransaction)
	if err != nil {
		log.Fatal().Err(err).Msg("importInternalTransaction")
	}
	return internalTransaction
}

func (self *MongoBackend) getBlockByNumber(blockNumber int64) *models.Block {
	var c models.Block
	err := self.mongo.C("Blocks").Find(bson.M{"number": blockNumber}).Select(bson.M{"transactions": 0}).One(&c)
	if err != nil {
		log.Debug().Int64("Block", blockNumber).Err(err).Msg("GetBlockByNumber")
		return nil
	}
	return &c
}

func (self *MongoBackend) getBlockByHash(blockHash string) *models.Block {
	var c models.Block
	err := self.mongo.C("Blocks").Find(bson.M{"hash": blockHash}).Select(bson.M{"transactions": 0}).One(&c)
	if err != nil {
		log.Debug().Str("Block", blockHash).Err(err).Msg("GetBlockByNumber")
		return nil
	}
	return &c
}

func (self *MongoBackend) getBlockTransactionsByNumber(blockNumber int64, skip, limit int) []*models.Transaction {
	var transactions []*models.Transaction
	err := self.mongo.C("Transactions").Find(bson.M{"block_number": blockNumber}).Skip(skip).Limit(limit).All(&transactions)
	if err != nil {
		log.Debug().Int64("block", blockNumber).Err(err).Msg("getBlockTransactions")
	}
	return transactions
}

func (self *MongoBackend) getLatestsBlocks(skip, limit int) []*models.LightBlock {
	var blocks []*models.LightBlock
	err := self.mongo.C("Blocks").Find(nil).Sort("-number").Select(bson.M{"number": 1, "created_at": 1, "miner": 1, "tx_count": 1}).Skip(skip).Limit(limit).All(&blocks)
	if err != nil {
		log.Debug().Int("Block", limit).Err(err).Msg("GetLatestsBlocks")
		return nil
	}
	return blocks
}

func (self *MongoBackend) getActiveAdresses(fromDate time.Time) []*models.ActiveAddress {
	var addresses []*models.ActiveAddress
	err := self.mongo.C("ActiveAddress").Find(bson.M{"updated_at": bson.M{"$gte": fromDate}}).All(&addresses)
	if err != nil {
		log.Debug().Err(err).Msg("GetActiveAdresses")
	}
	return addresses
}

func (self *MongoBackend) getAddressByHash(address string) *models.Address {
	var c models.Address
	err := self.mongo.C("Addresses").Find(bson.M{"address": address}).One(&c)
	if err != nil {
		log.Debug().Str("Address", address).Err(err).Msg("GetAddressByHash")
		return nil
	}
	//lazy calculation for number of transactions
	transactionCounter, err := self.mongo.C("Transactions").Find(bson.M{"$or": []bson.M{bson.M{"from": address}, bson.M{"to": address}}}).Count()
	if err != nil {
		log.Fatal().Err(err).Msg("importAddress")
	}
	c.NumberOfTransactions = transactionCounter
	return &c
}

func (self *MongoBackend) getTransactionByHash(transactionHash string) *models.Transaction {
	var c models.Transaction
	err := self.mongo.C("Transactions").Find(bson.M{"tx_hash": transactionHash}).One(&c)
	if err != nil {
		log.Debug().Str("Transaction", transactionHash).Err(err).Msg("GetTransactionByHash")
		return nil
	}
	//lazy calculation for receipt
	receipt, err := self.ethClient.TransactionReceipt(context.Background(), common.HexToHash(transactionHash))
	if err != nil {
		log.Warn().Err(err).Str("TX hash", common.HexToHash(transactionHash).String()).Msg("TransactionReceipt")
	} else {
		gasPrice := new(big.Int)
		_, err := fmt.Sscan(c.GasPrice, gasPrice)
		if err != nil {
			log.Error().Str("Cannot convert to bigint", c.GasPrice).Err(err).Msg("getTransactionByHash")
		}
		c.GasFee = new(big.Int).Mul(gasPrice, big.NewInt(int64(receipt.GasUsed))).String()
		log.Info().Str("Transaction", transactionHash).Uint64("Got new gas used", receipt.GasUsed).Uint64("Old gas", c.GasLimit).Msg("GetTransactionByHash")
	}
	return &c
}

func (self *MongoBackend) getTransactionList(address string, skip, limit int) []*models.Transaction {
	var transactions []*models.Transaction
	err := self.mongo.C("Transactions").Find(bson.M{"$or": []bson.M{bson.M{"from": address}, bson.M{"to": address}}}).Skip(skip).Limit(limit).All(&transactions)
	if err != nil {
		log.Debug().Str("address", address).Err(err).Msg("getAddressTransactions")
	}
	return transactions
}

func (self *MongoBackend) getTokenHoldersList(contractAddress string, skip, limit int) []*models.TokenHolder {
	var tokenHoldersList []*models.TokenHolder
	err := self.mongo.C("TokensHolders").Find(bson.M{"contract_address": contractAddress}).Sort("-balance_int").Skip(skip).Limit(limit).All(&tokenHoldersList)
	if err != nil {
		log.Debug().Str("contractAddress", contractAddress).Err(err).Msg("getTokenHoldersList")
	}
	return tokenHoldersList
}

func (self *MongoBackend) getInternalTransactionsList(contractAddress string, skip, limit int) []*models.InternalTransaction {
	var internalTransactionsList []*models.InternalTransaction
	err := self.mongo.C("InternalTransactions").Find(bson.M{"contract_address": contractAddress}).Sort("-block_number").Skip(skip).Limit(limit).All(&internalTransactionsList)
	if err != nil {
		log.Debug().Str("contractAddress", contractAddress).Err(err).Msg("getInternalTransactionsList")
	}
	return internalTransactionsList
}

func (self *MongoBackend) getRichlist(skip, limit int) []*models.Address {
	var addresses []*models.Address
	err := self.mongo.C("Addresses").Find(bson.M{"balance_float": bson.M{"$gt": 0}}).Sort("-balance_float").Skip(skip).Limit(limit).All(&addresses)
	if err != nil {
		log.Debug().Err(err).Msg("GetRichlist")
	}
	return addresses
}
func (self *MongoBackend) getStats() *models.Stats {
	numOfBlocks, err := self.mongo.C("Blocks").Find(nil).Count()
	if err != nil {
		log.Debug().Err(err).Msg("GetStats num of Blocks")
	}
	numOfTransactions, err := self.mongo.C("Transactions").Find(nil).Count()
	if err != nil {
		log.Debug().Err(err).Msg("GetStats num of Transactions")
	}
	return &models.Stats{NumberOfBlocks: int64(numOfBlocks), NumberOfTransactions: int64(numOfTransactions)}
}
