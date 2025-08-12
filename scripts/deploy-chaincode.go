#!/bin/bash

# Package chaincode
peer lifecycle chaincode package getreech.tar.gz --path ./chaincode --lang golang --label getreech_1.0

# Install chaincode on peer
export CORE_PEER_ADDRESS=peer0.org1.example.com:7051
peer lifecycle chaincode install getreech.tar.gz

# Approve chaincode for organization
peer lifecycle chaincode approveformyorg -o orderer.example.com:7050 --channelID channel1 --name getreech --version 1.0 --package-id getreech_1.0 --sequence 1

# Commit chaincode
peer lifecycle chaincode commit -o orderer.example.com:7050 --channelID channel1 --name getreech --version 1.0 --sequence 1