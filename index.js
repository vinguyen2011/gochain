'use strict';

var log4js = require('log4js');
var logger = log4js.getLogger('SERVER');

var express = require('express');
var app = express();
var bodyParser = require('body-parser');
app.use(bodyParser.json())

var hfc = require('fabric-client');
var utils = require('fabric-client/lib/utils.js');
var Peer = require('fabric-client/lib/Peer.js');
var Orderer = require('fabric-client/lib/Orderer.js');
var EventHub = require('fabric-client/lib/EventHub.js');

var config = require('./config.json');
var helper = require('./helper.js');

logger.setLevel('DEBUG');

var client = new hfc();
var chain;
var eventhub;
var tx_id = null;

init();

function init() {
	chain = client.newChain(config.chainName);
	chain.addOrderer(new Orderer(config.orderer.orderer_url));
  eventhub = new EventHub();
  eventhub.setPeerAddr(config.events[0].event_url);
  for (var i = 0; i < config.peers.length; i++) {
    chain.addPeer(new Peer(config.peers[i].peer_url));
  }
}

app.post('/invoke', function(req, res){
  eventhub.connect();

  hfc.newDefaultKeyValueStore({
    path: config.keyValueStore
  }).then(function (store) {
    client.setStateStore(store);
    return helper.getSubmitter(client);
  }).then(
    function (admin) {
      logger.info('Successfully obtained user to submit transaction');

      logger.info('Executing Invoke');
      tx_id = helper.getTxId();
      var nonce = utils.getNonce();
      var args = helper.getArgs(req.body.data);
      // send proposal to endorser
      var request = {
        chaincodeId: config.chaincodeID,
        fcn: config.invokeRequest.functionName,
        args: args,
        chainId: config.channelID,
        txId: tx_id,
        nonce: nonce
      };
      return chain.sendTransactionProposal(request);
    }
  ).then(
    function (results) {
      logger.info('Successfully obtained proposal responses from endorsers');

      return helper.processProposal(chain, results, 'move');
    }
  ).then(
    function (response) {
      if (response.status === 'SUCCESS') {
        var handle = setTimeout(() => {
            logger.error('Failed to receive transaction notification within the timeout period');
        res.sendStatus(500);
      }, parseInt(config.waitTime));
        eventhub.registerTxEvent(tx_id.toString(), (tx) => {
          logger.info('The chaincode transaction has been successfully committed');
        clearTimeout(handle);
        eventhub.disconnect();
        res.sendStatus(200);
      });
      }
    }
  ).catch(
    function (err) {
      eventhub.disconnect();
      logger.error('Failed to invoke transaction due to error: ' + err.stack ? err.stack : err);
      res.sendStatus(500);
    }
  );
});


app.post('/query', function(req, res){
  eventhub.connect();

  hfc.newDefaultKeyValueStore({
    path: config.keyValueStore
  }).then(function(store) {
    client.setStateStore(store);
    return helper.getSubmitter(client);
  }).then(
    function(admin) {
      logger.info('Successfully obtained enrolled user to perform query');

      logger.info('Executing Query');
      var targets = [];
      for (var i = 0; i < config.peers.length; i++) {
        targets.push(config.peers[i]);
      }
      var args = helper.getArgs(req.body.data);
      //chaincode query request
      var request = {
        targets: targets,
        chaincodeId: config.chaincodeID,
        chainId: config.channelID,
        txId: utils.buildTransactionID(),
        nonce: utils.getNonce(),
        fcn: config.queryRequest.functionName,
        args: args
      };
      // Query chaincode
      return chain.queryByChaincode(request);
    }
  ).then(
    function(response_payloads) {
      for (let i = 0; i < response_payloads.length; i++) {
        logger.info('############### Query results after the move on PEER%j, %j', i, response_payloads[i].toString('utf8'));
      }
      res.send(response_payloads[response_payloads.length - 1].toString('utf8'));
    }
  ).catch(
    function(err) {
      logger.error('Failed to end to end test with error:' + err.stack ? err.stack : err);
      res.sendStatus(500);
    }
  );
});

app.post('/deploy', function(req, res){
  eventhub.connect();
  console.log(req.body.data);

  hfc.newDefaultKeyValueStore({
    path: config.keyValueStore
  }).then(function(store) {
    client.setStateStore(store);
    return helper.getSubmitter(client);
  }).then(
    function(admin) {
      logger.info('Successfully obtained enrolled user to deploy the chaincode');

      logger.info('Executing Deploy');
      tx_id = helper.getTxId();
      var nonce = utils.getNonce();
      var args = helper.getArgs(req.body.data);
      // send proposal to endorser
      var request = {
        chaincodePath: config.chaincodePath,
        chaincodeId: config.chaincodeID,
        fcn: config.deployRequest.functionName,
        args: args,
        chainId: config.channelID,
        txId: tx_id,
        nonce: nonce,
        'dockerfile-contents': config.dockerfile_contents
      };

      return chain.sendDeploymentProposal(request);
    }
  ).then(
    function(results) {
      logger.info('Successfully obtained proposal responses from endorsers');
      return helper.processProposal(chain, results, 'deploy');
    }
  ).then(
    function(response) {
      if (response.status === 'SUCCESS') {
        logger.info('Successfully sent deployment transaction to the orderer.');
        var handle = setTimeout(() => {
            logger.error('Failed to receive transaction notification within the timeout period');
            res.sendStatus(500);
        }, parseInt(config.waitTime));

          eventhub.registerTxEvent(tx_id.toString(), (tx) => {
            logger.info('The chaincode transaction has been successfully committed');
          clearTimeout(handle);
          eventhub.disconnect();
          res.sendStatus(200);
        });
      } else {
        logger.error('Failed to order the deployment endorsement. Error code: ' + response.status);
        res.sendStatus(response.status);
      }
    }
  ).catch(
    function(err) {
      eventhub.disconnect();
      logger.error(err.stack ? err.stack : err);
      res.sendStatus(500);
    }
  );
});

app.listen(3000);