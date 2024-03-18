package main

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/incognitochain/bridge-eth/rpccaller"
)

type rpcTransaction struct {
	tx *types.Transaction
	txExtraInfo
}

type txExtraInfo struct {
	BlockNumber *string         `json:"blockNumber,omitempty"`
	BlockHash   *common.Hash    `json:"blockHash,omitempty"`
	From        *common.Address `json:"from,omitempty"`
}

func (tx *rpcTransaction) UnmarshalJSON(msg []byte) error {
	if err := json.Unmarshal(msg, &tx.tx); err != nil {
		return err
	}
	return json.Unmarshal(msg, &tx.txExtraInfo)
}

type RpcBlock struct {
	Hash         common.Hash              `json:"hash"`
	Transactions []map[string]interface{} `json:"transactions"`
	UncleHashes  []common.Hash            `json:"uncles"`
}

type GetEVMBlockByNumberRes struct {
	rpccaller.RPCBaseRes
	Result *RpcBlock `json:"result"`
}
