#!/bin/bash

set -e

ip=172.10.10.246
port=5000

swarm_key=$(cat ~/.gdc/stacks/c/stack.json  | jq -r .swarmKey)
contract_address=$(cat ~/.gdc/stacks/c/runtime/stackState.json | jq -r ".deployedContracts[1].location.address")

#http://10.1.120.43:26657/status

node_bootnodes=$(curl -s -X POST -H "Content-Type: application/json"  --data '{"jsonrpc":"2.0","method":"admin_nodeInfo","params":[],"id":1}' http://${ip}:$((${port}+98)) | jq .result.enode -r)
node_bootnodes=$(echo $node_bootnodes | sed "s/127\.0\.0\.1/${ip}/g" | sed "s/:30311/:$((${port}+99))/g" | sed 's/?discport=0//')
echo $node_bootnodes

ipfs_bootnodes=$(curl -s -X POST http://${ip}:$((${port}+18))/api/v0/id | jq -r '.Addresses[0]')
ipfs_bootnodes=$(echo $ipfs_bootnodes | sed "s/127\.0\.0\.1/${ip}/g" | sed "s/4001/$((${port}+17))/g")
echo $ipfs_bootnodes


#cp ~/.gdc/stacks/c/init/blockchain/genesis.json ~/.gdc/stacks/f/init/blockchain/genesis.json
