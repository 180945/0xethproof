package main

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/incognitochain/bridge-eth/rpccaller"
)

type RpcBlock struct {
	Hash         common.Hash              `json:"hash"`
	Transactions []map[string]interface{} `json:"transactions"`
	UncleHashes  []common.Hash            `json:"uncles"`
}

type GetEVMBlockByNumberRes struct {
	rpccaller.RPCBaseRes
	Result *RpcBlock `json:"result"`
}
