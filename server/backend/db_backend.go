package backend

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/gochain-io/explorer/server/models"
	"github.com/gochain-io/gochain/v3/common"
	"github.com/gochain-io/gochain/v3/core/types"
	"github.com/gochain-io/gochain/v3/goclient"
)

var wei = big.NewInt(1000000000000000000)

type MongoBackend struct {
	host         string
	mongo        *mgo.Database
	mongoSession *mgo.Session
	goClient     *goclient.Client
}

// New create new rpc client with given url
func NewMongoClient(host, rpcUrl, dbName string) *MongoBackend {
	client, err := goclient.Dial(rpcUrl)
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
	importer.mongoSession = session
	importer.mongo = session.DB(dbName)
	importer.goClient = client
	importer.createIndexes()

	return importer

}
func (self *MongoBackend) PingDB() error {
	return self.mongoSession.Ping()
}
func (self *MongoBackend) parseTx(tx *types.Transaction, block *types.Block) *models.Transaction {
	from, err := self.goClient.TransactionSender(context.Background(), tx, block.Header().Hash(), 0)
	if err != nil {
		log.Fatal().Err(err).Msg("parseTx")
	}
	gas := tx.Gas()
	to := ""
	if tx.To() != nil {
		to = tx.To().Hex()
	}
	log.Debug().Interface("TX:", tx).Msg("parseTx")
	InputDataEmpty := hex.EncodeToString(tx.Data()[:]) == ""
	return &models.Transaction{TxHash: tx.Hash().Hex(),
		To:              to,
		From:            from.Hex(),
		Value:           tx.Value().String(),
		GasPrice:        tx.GasPrice().String(),
		ReceiptReceived: false,
		GasLimit:        tx.Gas(),
		BlockNumber:     block.Number().Int64(),
		GasFee:          new(big.Int).Mul(tx.GasPrice(), big.NewInt(int64(gas))).String(),
		Nonce:           tx.Nonce(),
		BlockHash:       block.Hash().Hex(),
		CreatedAt:       time.Unix(block.Time().Int64(), 0),
		InputData:       hex.EncodeToString(tx.Data()[:]),
		InputDataEmpty:  InputDataEmpty,
	}
}
func (self *MongoBackend) parseBlock(block *types.Block) *models.Block {
	var transactions []string
	for _, tx := range block.Transactions() {
		transactions = append(transactions, tx.Hash().Hex())
	}
	nonceBool := false
	if block.Nonce() == 0xffffffffffffffff {
		nonceBool = true
	}
	return &models.Block{Number: block.Header().Number.Int64(),
		GasLimit:   int(block.Header().GasLimit),
		BlockHash:  block.Hash().Hex(),
		CreatedAt:  time.Unix(block.Time().Int64(), 0),
		ParentHash: block.ParentHash().Hex(),
		TxHash:     block.Header().TxHash.Hex(),
		GasUsed:    strconv.Itoa(int(block.Header().GasUsed)),
		NonceBool:  &nonceBool,
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
	err := self.mongo.C("Transactions").EnsureIndex(mgo.Index{Key: []string{"tx_hash"}, Unique: true, DropDups: true, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("Transactions").EnsureIndex(mgo.Index{Key: []string{"block_number"}, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("Transactions").EnsureIndex(mgo.Index{Key: []string{"from", "created_at", "input_data_empty"}, Background: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("Transactions").EnsureIndex(mgo.Index{Key: []string{"to", "created_at", "input_data_empty"}, Background: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("Transactions").EnsureIndex(mgo.Index{Key: []string{"-created_at"}, Background: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("Transactions").EnsureIndex(mgo.Index{Key: []string{"contract_address"}, Background: true})
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

	err = self.mongo.C("Blocks").EnsureIndex(mgo.Index{Key: []string{"created_at", "miner"}, Background: true, Sparse: true})
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

	err = self.mongo.C("Addresses").EnsureIndex(mgo.Index{Key: []string{"-balance_float", "address"}, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("TokensHolders").EnsureIndex(mgo.Index{Key: []string{"contract_address", "token_holder_address"}, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("TokensHolders").EnsureIndex(mgo.Index{Key: []string{"token_holder_address"}, Background: true, Sparse: true})
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
	err = self.mongo.C("InternalTransactions").EnsureIndex(mgo.Index{Key: []string{"from_address", "block_number"}, Background: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("InternalTransactions").EnsureIndex(mgo.Index{Key: []string{"to_address", "block_number"}, Background: true})
	if err != nil {
		panic(err)
	}
	err = self.mongo.C("InternalTransactions").EnsureIndex(mgo.Index{Key: []string{"transaction_hash"}, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("InternalTransactions").EnsureIndex(mgo.Index{Key: []string{"block_number"}, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("Stats").EnsureIndex(mgo.Index{Key: []string{"-updated_at"}, Background: true, Sparse: true})
	if err != nil {
		panic(err)
	}

	err = self.mongo.C("Contracts").EnsureIndex(mgo.Index{Key: []string{"address"}, Unique: true, DropDups: true, Background: true, Sparse: true})
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
	self.UpdateActiveAddress(block.Coinbase().Hex())
	return b

}

func (self *MongoBackend) UpdateActiveAddress(address string) {
	_, err := self.mongo.C("ActiveAddress").Upsert(bson.M{"address": address}, &models.ActiveAddress{Address: address, UpdatedAt: time.Now()})
	if err != nil {
		log.Fatal().Err(err).Msg("UpdateActiveAddress")
	}
}

func (self *MongoBackend) importTx(tx *types.Transaction, block *types.Block) {
	log.Debug().Msg("Importing tx" + tx.Hash().Hex())
	transaction := self.parseTx(tx, block)

	toAddress := transaction.To
	if transaction.To == "" {
		log.Info().Str("hash", transaction.TxHash).Msg("Hash doesn't have an address")
		receipt, err := self.goClient.TransactionReceipt(context.Background(), tx.Hash())
		if err == nil {
			contractAddress := receipt.ContractAddress.String()
			if contractAddress != "0x0000000000000000000000000000000000000000" {
				transaction.ContractAddress = contractAddress
			}
			transaction.Status = false
			if receipt.Status == 1 {
				transaction.Status = true
			}
			toAddress = transaction.ContractAddress
		} else {
			log.Error().Err(err).Str("hash", transaction.TxHash).Msg("Cannot get a receipt in importTX")
		}
	}

	_, err := self.mongo.C("Transactions").Upsert(bson.M{"tx_hash": tx.Hash().String()}, transaction)
	if err != nil {
		log.Fatal().Err(err).Msg("importTx")
	}

	self.UpdateActiveAddress(toAddress)
	self.UpdateActiveAddress(transaction.From)
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

func (self *MongoBackend) importAddress(address string, balance *big.Int, token *TokenDetails, contract bool, updatedAtBlock int64) *models.Address {
	balanceGoFloat, _ := new(big.Float).SetPrec(100).Quo(new(big.Float).SetInt(balance), new(big.Float).SetInt(wei)).Float64() //converting to GO from wei
	balanceGoString := new(big.Rat).SetFrac(balance, wei).FloatString(18)
	log.Debug().Str("address", address).Str("precise balance", balanceGoString).Float64("balance float", balanceGoFloat).Msg("Updating address")
	tokenHoldersCounter, err := self.mongo.C("TokensHolders").Find(bson.M{"contract_address": address}).Count()
	if err != nil {
		log.Fatal().Err(err).Msg("importAddress")
	}

	internalTransactionsCounter, err := self.mongo.C("InternalTransactions").Find(bson.M{"contract_address": address}).Count()

	if err != nil {
		log.Fatal().Err(err).Msg("importAddress")
	}

	tokenTransactionsCounter, err := self.mongo.C("InternalTransactions").Find(bson.M{"$or": []bson.M{bson.M{"from_address": address}, bson.M{"to_address": address}}}).Count()
	if err != nil {
		log.Fatal().Err(err).Msg("importAddress")
	}

	addressM := &models.Address{Address: address,
		BalanceWei:     balance.String(),
		UpdatedAt:      time.Now(),
		UpdatedAtBlock: updatedAtBlock,
		TokenName:      token.Name,
		TokenSymbol:    token.Symbol,
		Decimals:       token.Decimals,
		TotalSupply:    token.TotalSupply.String(),
		Contract:       contract,
		ErcTypes:       token.Types,
		Interfaces:     token.Interfaces,
		BalanceFloat:   balanceGoFloat,
		BalanceString:  balanceGoString,
		// NumberOfTransactions:         transactionCounter,
		NumberOfTokenHolders:         tokenHoldersCounter,
		NumberOfInternalTransactions: internalTransactionsCounter,
		NumberOfTokenTransactions:    tokenTransactionsCounter,
	}
	_, err = self.mongo.C("Addresses").Upsert(bson.M{"address": address}, addressM)
	if err != nil {
		log.Fatal().Err(err).Msg("importAddress")
	}
	return addressM
}

func (self *MongoBackend) importTokenHolder(contractAddress, tokenHolderAddress string, token *TokenHolderDetails, address *models.Address) *models.TokenHolder {
	balanceInt := new(big.Int).Div(token.Balance, wei) //converting to GO from wei
	log.Info().Str("contractAddress", contractAddress).Str("tokenAddress", tokenHolderAddress).Str("balance", token.Balance.String()).Str("Balance int", balanceInt.String()).Msg("Updating token holder")
	tokenHolder := &models.TokenHolder{
		TokenName:          address.TokenName,
		TokenSymbol:        address.TokenSymbol,
		ContractAddress:    contractAddress,
		TokenHolderAddress: tokenHolderAddress,
		Balance:            token.Balance.String(),
		UpdatedAt:          time.Now(),
		BalanceInt:         balanceInt.Int64()}
	_, err := self.mongo.C("TokensHolders").Upsert(bson.M{"contract_address": contractAddress, "token_holder_address": tokenHolderAddress}, tokenHolder)
	if err != nil {
		log.Fatal().Err(err).Msg("importTokenHolder")
	}
	return tokenHolder

}

func (self *MongoBackend) importInternalTransaction(contractAddress string, transferEvent TransferEvent, createdAt time.Time) *models.InternalTransaction {

	internalTransaction := &models.InternalTransaction{
		Contract:        contractAddress,
		From:            transferEvent.From.String(),
		To:              transferEvent.To.String(),
		Value:           transferEvent.Value.String(),
		BlockNumber:     transferEvent.BlockNumber,
		TransactionHash: transferEvent.TransactionHash,
		CreatedAt:       createdAt,
		UpdatedAt:       time.Now(),
	}
	_, err := self.mongo.C("InternalTransactions").Upsert(bson.M{"transaction_hash": transferEvent.TransactionHash}, internalTransaction)
	if err != nil {
		log.Fatal().Err(err).Msg("importInternalTransaction")
	}
	return internalTransaction
}

func (self *MongoBackend) importContract(contractAddress string, byteCode string) *models.Contract {
	contract := &models.Contract{
		Address:   contractAddress,
		Bytecode:  byteCode,
		CreatedAt: time.Now(),
	}
	_, err := self.mongo.C("Contracts").Upsert(bson.M{"address": contract.Address}, contract)
	if err != nil {
		log.Fatal().Err(err).Msg("importContract")
	}

	return contract
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
	err := self.mongo.C("Blocks").Find(nil).Sort("-number").Select(bson.M{"number": 1, "created_at": 1, "miner": 1, "tx_count": 1, "extra_data": 1}).Skip(skip).Limit(limit).All(&blocks)
	if err != nil {
		log.Debug().Int("Block", limit).Err(err).Msg("GetLatestsBlocks")
		return nil
	}
	return blocks
}

func (self *MongoBackend) getActiveAdresses(fromDate time.Time) []*models.ActiveAddress {
	var addresses []*models.ActiveAddress
	err := self.mongo.C("ActiveAddress").Find(bson.M{"updated_at": bson.M{"$gte": fromDate}}).Select(bson.M{"address": 1}).Sort("-updated_at").All(&addresses)
	if err != nil {
		log.Debug().Err(err).Msg("GetActiveAdresses")
	}
	return addresses
}

func (self *MongoBackend) isContract(address string) bool {
	var c models.Address
	err := self.mongo.C("Addresses").Find(bson.M{"address": address}).Select(bson.M{"contract": 1}).One(&c)
	if err != nil {
		log.Debug().Str("Address", address).Err(err).Msg("isContract")
		return false
	}
	return c.Contract
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
	// lazy calculation for receipt
	if !c.ReceiptReceived {
		receipt, err := self.goClient.TransactionReceipt(context.Background(), common.HexToHash(transactionHash))
		if err != nil {
			log.Warn().Err(err).Str("TX hash", common.HexToHash(transactionHash).String()).Msg("TransactionReceipt")
		} else {
			gasPrice := new(big.Int)
			_, err := fmt.Sscan(c.GasPrice, gasPrice)
			if err != nil {
				log.Error().Str("Cannot convert to bigint", c.GasPrice).Err(err).Msg("getTransactionByHash")
			}
			c.GasFee = new(big.Int).Mul(gasPrice, big.NewInt(int64(receipt.GasUsed))).String()
			c.ContractAddress = receipt.ContractAddress.String()
			c.Status = false
			if receipt.Status == 1 {
				c.Status = true
			}
			c.ReceiptReceived = true
			jsonValue, err := json.Marshal(receipt.Logs)
			if err != nil {
				log.Error().Err(err).Msg("getTransactionByHash")
			}
			c.Logs = string(jsonValue)
			log.Info().Str("Transaction", transactionHash).Uint64("Got new gas used", receipt.GasUsed).Uint64("Old gas", c.GasLimit).Msg("GetTransactionByHash")
			_, err = self.mongo.C("Transactions").Upsert(bson.M{"tx_hash": c.TxHash}, c)
			if err != nil {
				log.Error().Err(err).Msg("getTransactionByHash")
			}
		}
	}
	return &c
}

func (self *MongoBackend) getTransactionList(address string, skip, limit int, fromTime, toTime time.Time, inputDataEmpty *bool) []*models.Transaction {
	var transactions []*models.Transaction
	var err error
	if inputDataEmpty != nil {
		err = self.mongo.C("Transactions").Find(bson.M{"$or": []bson.M{bson.M{"from": address}, bson.M{"to": address}}, "created_at": bson.M{"$gte": fromTime, "$lte": toTime}, "input_data_empty": *inputDataEmpty}).Sort("-created_at").Skip(skip).Limit(limit).All(&transactions)
	} else {
		err = self.mongo.C("Transactions").Find(bson.M{"$or": []bson.M{bson.M{"from": address}, bson.M{"to": address}}, "created_at": bson.M{"$gte": fromTime, "$lte": toTime}}).Sort("-created_at").Skip(skip).Limit(limit).All(&transactions)
	}
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
func (self *MongoBackend) getOwnedTokensList(ownerAddress string, skip, limit int) []*models.TokenHolder {
	var tokenHoldersList []*models.TokenHolder
	err := self.mongo.C("TokensHolders").Find(bson.M{"token_holder_address": ownerAddress}).Sort("-balance_int").Skip(skip).Limit(limit).All(&tokenHoldersList)
	if err != nil {
		log.Debug().Str("token_holder_address", ownerAddress).Err(err).Msg("getOwnedTokensList")
	}
	return tokenHoldersList
}

func (self *MongoBackend) getInternalTransactionsList(contractAddress string, tokenTransactions bool, skip, limit int) []*models.InternalTransaction {
	var internalTransactionsList []*models.InternalTransaction
	var query bson.M
	if tokenTransactions {
		query = bson.M{"$or": []bson.M{bson.M{"from_address": contractAddress}, bson.M{"to_address": contractAddress}}}
	} else {
		query = bson.M{"contract_address": contractAddress}
	}
	err := self.mongo.C("InternalTransactions").Find(query).Sort("-block_number").Skip(skip).Limit(limit).All(&internalTransactionsList)
	if err != nil {
		log.Debug().Str("contractAddress", contractAddress).Err(err).Msg("getInternalTransactionsList")
	}
	return internalTransactionsList
}

func (self *MongoBackend) getContract(contractAddress string) *models.Contract {
	var contract *models.Contract
	err := self.mongo.C("Contracts").Find(bson.M{"address": contractAddress}).One(&contract)
	if err != nil {
		log.Debug().Str("contractAddress", contractAddress).Err(err).Msg("getContract")
	}
	return contract
}

func (self *MongoBackend) getContractBlock(contractAddress string) int64 {
	var transaction *models.Transaction
	err := self.mongo.C("Transactions").Find(bson.M{"contract_address": contractAddress}).One(&transaction)
	if err != nil {
		log.Debug().Str("address", contractAddress).Err(err).Msg("getContractBlock")
	}
	if transaction != nil {
		return transaction.BlockNumber
	} else {
		return 0
	}

}

func (self *MongoBackend) updateContract(contract *models.Contract) bool {
	_, err := self.mongo.C("Contracts").Upsert(bson.M{"address": contract.Address}, contract)
	if err != nil {
		log.Fatal().Err(err).Msg("updateContract")
		return false
	}
	return true
}

func (self *MongoBackend) getRichlist(skip, limit int, lockedAddresses []string) []*models.Address {
	var addresses []*models.Address
	err := self.mongo.C("Addresses").Find(bson.M{"balance_float": bson.M{"$gt": 0}, "address": bson.M{"$nin": lockedAddresses}}).Sort("-balance_float").Skip(skip).Limit(limit).All(&addresses)
	if err != nil {
		log.Debug().Err(err).Msg("GetRichlist")
	}
	return addresses
}
func (self *MongoBackend) updateStats() {
	numOfTotalTransactions, err := self.mongo.C("Transactions").Find(nil).Count()
	if err != nil {
		log.Debug().Err(err).Msg("GetStats num of Total Transactions")
	}
	numOfLastWeekTransactions, err := self.mongo.C("Transactions").Find(bson.M{"created_at": bson.M{"$gte": time.Now().AddDate(0, 0, -7)}}).Count()
	if err != nil {
		log.Debug().Err(err).Msg("GetStats num of Last week Transactions")
	}
	numOfLastDayTransactions, err := self.mongo.C("Transactions").Find(bson.M{"created_at": bson.M{"$gte": time.Now().AddDate(0, 0, -1)}}).Count()
	if err != nil {
		log.Debug().Err(err).Msg("GetStats num of 24H Transactions")
	}
	stats := &models.Stats{
		NumberOfTotalTransactions:    int64(numOfTotalTransactions),
		NumberOfLastWeekTransactions: int64(numOfLastWeekTransactions),
		NumberOfLastDayTransactions:  int64(numOfLastDayTransactions),
		UpdatedAt:                    time.Now(),
	}
	err = self.mongo.C("Stats").Insert(stats)
	if err != nil {
		log.Debug().Err(err).Msg("Failes to update stats")
	}
}
func (self *MongoBackend) getStats() *models.Stats {
	var s *models.Stats
	err := self.mongo.C("Stats").Find(nil).Sort("-updated_at").One(&s)
	if err != nil {
		log.Debug().Err(err).Msg("Cannot get stats")
		s = &models.Stats{
			NumberOfTotalTransactions:    0,
			NumberOfLastWeekTransactions: 0,
			NumberOfLastDayTransactions:  0,
		}
	}
	return s
}
func (self *MongoBackend) getSignerStatsForRange(endTime time.Time, dur time.Duration, nodes map[common.Address]models.Node) []models.SignerStats {
	var resp []bson.M
	var stat []models.SignerStats
	queryDayStats := []bson.M{bson.M{"$match": bson.M{"created_at": bson.M{"$gte": endTime.Add(dur)}}}, bson.M{"$group": bson.M{"_id": "$miner", "count": bson.M{"$sum": 1}}}}
	pipe := self.mongo.C("Blocks").Pipe(queryDayStats)
	err := pipe.All(&resp)
	if err != nil {
		log.Info().Err(err).Msg("Cannot run pipe")
	}
	log.Info().Time("Date", endTime.Add(dur)).Msg("stats")
	for _, el := range resp {
		signerStats := models.SignerStats{Signer: common.HexToAddress(el["_id"].(string)), BlocksCount: el["count"].(int)}
		if val, ok := nodes[signerStats.Signer]; ok {
			signerStats.Name = val.Name
			signerStats.Region = val.Region
			signerStats.URL = val.URL
		}
		stat = append(stat, signerStats)
	}
	return stat
}

func (self *MongoBackend) getBlockRange(endTime time.Time, dur time.Duration) models.BlockRange {
	var startBlock, endBlock models.Block
	var resp models.BlockRange
	err := self.mongo.C("Blocks").Find(bson.M{"created_at": bson.M{"$gte": endTime.Add(dur)}}).Select(bson.M{"number": 1}).Sort("created_at").One(&startBlock)
	if err != nil {
		log.Info().Err(err).Msg("Cannot get block number")
	}
	log.Info().Interface("element", startBlock).Msg("startBlock")
	err = self.mongo.C("Blocks").Find(bson.M{"created_at": bson.M{"$gte": endTime.Add(dur)}}).Select(bson.M{"number": 1}).Sort("-created_at").One(&endBlock)
	if err != nil {
		log.Info().Err(err).Msg("Cannot get block number")
	}
	resp.StartBlock = startBlock.Number
	resp.EndBlock = endBlock.Number
	return resp
}

func (self *MongoBackend) getSignersStats(nodes map[common.Address]models.Node) []models.SignersStats {
	var stats []models.SignersStats
	kvs := map[string]time.Duration{"daily": -24 * time.Hour, "weekly": -7 * 24 * time.Hour, "monthly": -30 * 24 * time.Hour}
	endTime := time.Now()
	for k, v := range kvs {
		stats = append(stats, models.SignersStats{BlockRange: self.getBlockRange(endTime, v), SignerStats: self.getSignerStatsForRange(endTime, v, nodes), Range: k})
	}
	return stats
}

func (self *MongoBackend) cleanUp() {
	collectionNames, err := self.mongo.CollectionNames()
	if err != nil {
		log.Info().Err(err).Msg("Cannot get list of collections")
		return
	}
	for _, collectionName := range collectionNames {
		log.Info().Str("collection name", collectionName).Msg("cleanUp")
		self.mongo.C(collectionName).RemoveAll(nil)
	}
}
