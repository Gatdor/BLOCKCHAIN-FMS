#!/bin/bash

# Clone Fabric samples if not present
if [ ! -d "fabric-samples" ]; then
    git clone https://github.com/hyperledger/fabric-samples.git
fi

# Navigate to test-network
cd fabric-samples/test-network

# Clean up previous network
./network.sh down

# Start the network with one organization
./network.sh up createChannel -c channel1 -ca

# Add environment variables for chaincode deployment
export PATH=${PWD}/../bin:$PATH
export FABRIC_CFG_PATH=${PWD}/../config/

echo "Fabric test network started. Use scripts/deploy_chaincode.sh to deploy chaincode."
