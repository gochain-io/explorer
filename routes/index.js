var mongoose = require('mongoose');

var Block = mongoose.model('Block');
var Transaction = mongoose.model('Transaction');
var Address = mongoose.model('Address');

var filters = require('./filters')
var web3relay = require('./web3relay');


var async = require('async');

module.exports = function (app) {

  var DAO = require('./dao');
  var Token = require('./token');

  var compile = require('./compiler');
  var stats = require('./stats');

  /*
    Local DB: data request format
    { "address": "0x1234blah", "txin": true }
    { "tx": "0x1234blah" }
    { "block": "1234" }
  */
  app.post('/addr', getAddr);
  app.post('/tx', getTx);
  app.post('/block', getBlock);
  app.post('/data', getData);

  app.post('/daorelay', DAO);
  app.post('/tokenrelay', Token);
  app.post('/web3relay', web3relay.data);
  app.post('/compile', compile);

  app.post('/stats', stats);

  app.post('/signed', blocksSignedByAddr);

  app.get('/config', getConfig);
  app.get('/api/richlist', getRichList);
  app.get('/totalSupply', getTotalSupply);
  app.get('/circulatingSupply', getCirculating);
}

function getConfig(req, res) {
  const cfg = require('../config');

  res.send(cfg);
}

function getTotalSupply(req, res) {
  web3relay.totalSupply(function (supply) {
    res.send(supply.toString(10));
  });
}

function getCirculating(req, res) {
  web3relay.circulatingSupply(function (supply) {
    res.send(supply.toString(10));
  });
}

var getRichList = function (req, res) {
  var limit = parseInt(req.query.limit) || 100;
  var start = parseInt(req.query.start) || 0;
  console.log("limit:", limit);
  console.log("start:", start);
  var data = {};
  web3relay.totalSupply(function (totalSupply) {
    web3relay.circulatingSupply(function (circulatingSupply) {
      Address.find({ "balanceDecimal": { "$gt": 0 } }).lean(true).sort('-balanceDecimal').skip(start).limit(limit)
        .exec("find", function (err, docs) {
          data.circulatingSupply = circulatingSupply.toString(10);
          data.totalSupply = totalSupply.toString(10);
          if (docs)
            data.rankings = filters.filterAddresses(docs, start);
          else
            data.rankings = [];
          res.write(JSON.stringify(data));
          res.end();
        });
    });
  });
}

var getAddr = function (req, res) {
  // TODO: validate addr and tx
  var addr = req.body.addr.toLowerCase();
  var count = parseInt(req.body.count);

  var limit = parseInt(req.body.length);
  var start = parseInt(req.body.start);

  var data = { draw: parseInt(req.body.draw), recordsFiltered: count, recordsTotal: count };

  var addrFind = Transaction.find({ $or: [{ "to": addr }, { "from": addr }] })

  addrFind.lean(true).sort('-blockNumber').skip(start).limit(limit)
    .exec("find", function (err, docs) {
      if (docs)
        data.data = filters.filterTX(docs, addr);
      else
        data.data = [];
      res.write(JSON.stringify(data));
      res.end();
    });

};



var getBlock = function (req, res) {

  // TODO: support queries for block hash
  var txQuery = "number";
  var number = parseInt(req.body.block);

  var blockFind = Block.findOne({ number: number }).lean(true);
  blockFind.exec(function (err, doc) {
    if (err || !doc) {
      console.error("BlockFind error: " + err)
      console.error(req.body);
      res.write(JSON.stringify({ "error": true }));
    } else {
      var block = filters.filterBlocks([doc]);
      res.write(JSON.stringify(block[0]));
    }
    res.end();
  });

};

var getTx = function (req, res) {

  var tx = req.body.tx.toLowerCase();

  var txFind = Block.findOne({ "transactions.hash": tx }, "transactions timestamp")
    .lean(true);
  txFind.exec(function (err, doc) {
    if (!doc) {
      console.log("missing: " + tx)
      res.write(JSON.stringify({}));
      res.end();
    } else {
      // filter transactions
      var txDocs = filters.filterBlock(doc, "hash", tx)
      res.write(JSON.stringify(txDocs));
      res.end();
    }
  });

};


/*
  Fetch data from DB
*/
var getData = function (req, res) {

  // TODO: error handling for invalid calls
  var action = req.body.action.toLowerCase();
  var limit = req.body.limit

  if (action in DATA_ACTIONS) {
    if (isNaN(limit))
      var lim = MAX_ENTRIES;
    else
      var lim = parseInt(limit);

    DATA_ACTIONS[action](lim, res);

  } else {

    console.error("Invalid Request: " + action)
    res.status(400).send();
  }

};

/*
  temporary blockstats here
*/
var blocksSignedByAddr = function (req, res) {
  var addr = req.body.addr.toLowerCase();
  Block.count({ miner: addr }, function (err, blockResult) {
    if (err) {
      console.error(err);
      res.status(500).send();
    } else {
      console.log("Number of blocks mined by specific address in DB:", blockResult);
      res.write(JSON.stringify(
        {
          signed: blockResult
        }
      ));
      res.end();
    }
  });
}

var latestBlock = function (req, res) {
  var block = Block.findOne({}, "totalDifficulty")
    .lean(true).sort('-number');
  block.exec(function (err, doc) {
    res.write(JSON.stringify(doc));
    res.end();
  });
}


var getLatest = function (lim, res, callback) {
  var blockFind = Block.find({}, "number transactions timestamp miner extraData transactionsCount")
    .lean(true).sort('-number').limit(lim);
  blockFind.exec(function (err, docs) {
    callback(docs, res);
  });
}

/* get blocks from db */
var sendBlocks = function (lim, res) {
  var blockFind = Block.find({}, "number transactions timestamp miner extraData transactionsCount")
    .lean(true).sort('-number').limit(lim);
  blockFind.exec(function (err, docs) {
    res.write(JSON.stringify({ "blocks": filters.filterBlocks(docs) }));
    res.end();
  });
}

var sendTxs = function (lim, res) {
  Transaction.find({}).lean(true).sort('-blockNumber').limit(lim)
    .exec(function (err, txs) {
      res.write(JSON.stringify({ "txs": txs }));
      res.end();
    });
}

const MAX_ENTRIES = 10;

const DATA_ACTIONS = {
  "latest_blocks": sendBlocks,
  "latest_txs": sendTxs
}
