package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/incognitochain/bridge-eth/rpccaller"
	"github.com/pkg/errors"
	"math/big"
	"strconv"
)

func main() {

	//txHash := common.HexToHash("0xb23bd58950e36cc6505749d15a0bb043c3623bf85bec1e4a424283fb03b5b90f")

	rpc := "https://eth.llamarpc.com"

	//_, ethBlockHash, ethTxIdx, ethDepositProof, err := getETHDepositProof(rpc, txHash)
	//if err != nil {
	//	panic(err.Error())
	//}
	//
	//fmt.Println("depositProof ---- : ", ethBlockHash, ethTxIdx, ethDepositProof)
	//
	//// verify
	//res, err := VerifyProofAndParseEVMReceipt(ethBlockHash, ethTxIdx, ethDepositProof, rpc, true)
	//if err != nil {
	//	panic(err.Error())
	//}

	//fmt.Println(res)

	//client, err := ethclient.Dial(rpc)
	//if err != nil {
	//	panic(err.Error())
	//}
	//
	//block, err := client.BlockByNumber(context.Background(), big.NewInt(19428643))
	//if err != nil {
	//	panic(err.Error())
	//}
	//fmt.Println(block)

	blockByNum, err := GetEVMHeaderByNumber(big.NewInt(19428643), rpc)
	if err != nil {
		panic(err.Error())
	}

	for _, v := range blockByNum.Transactions {
		// check tx type
		txType, success := v["type"].(string)
		if !success {
			panic(errors.New("get type failed"))
		}
		// skip if tx type is 3
		decimal_num, err := strconv.ParseInt(txType[2:], 16, 64)
		// in case of any error
		if err != nil {
			panic(err)
		}

		if decimal_num == EIP4844 {
			continue
		}
		jsonData, _ := json.Marshal(v)
		var tx *types.Transaction
		err = json.Unmarshal(jsonData, &tx)
		if err != nil {
			panic(err)
		}
		fmt.Println("--------------------------------")
		fmt.Println(tx.To().String())
	}
}

func VerifyProofAndParseEVMReceipt(
	blockHash string,
	txIndex uint,
	proofStrs []string,
	url string,
	checkEVMHarkFork bool,
) (*types.Receipt, error) {
	// get evm header result
	blockHeader, err := getETHBlockByHash(url, blockHash)
	if err != nil {
		panic(err)
	}

	receiptHash, success := blockHeader["receiptsRoot"].(string)
	if !success {
		panic(errors.New("blockHeader[\"receiptsRoot\"].(string)"))
	}

	keybuf := new(bytes.Buffer)
	keybuf.Reset()
	rlp.Encode(keybuf, txIndex)

	nodeList := new(light.NodeList)
	for _, proofStr := range proofStrs {
		proofBytes, err := base64.StdEncoding.DecodeString(proofStr)
		if err != nil {
			return nil, err
		}
		nodeList.Put([]byte{}, proofBytes)
	}
	proof := nodeList.NodeSet()
	val, err := trie.VerifyProof(common.HexToHash(receiptHash), keybuf.Bytes(), proof)
	if err != nil {
		panic(err)
	}

	// if iReq.Type == IssuingETHRequestMeta || iReq.Type == IssuingPRVERC20RequestMeta || iReq.Type == IssuingPLGRequestMeta {
	if checkEVMHarkFork {
		if len(val) == 0 {
			panic(errors.New("the encoded receipt is empty"))
		}

		// hardfork london with new transaction type => 0x02 || RLP([...SenderPayload, ...SenderSignature, ...GasPayerPayload, ...GasPayerSignature])
		if val[0] == 1 || val[0] == 2 {
			val = val[1:]
		}
	}

	// Decode value from VerifyProof into Receipt
	constructedReceipt := new(types.Receipt)
	err = rlp.DecodeBytes(val, constructedReceipt)
	if err != nil {
		panic(err)
	}

	if constructedReceipt.Status != types.ReceiptStatusSuccessful {
		panic(errors.New("the constructedReceipt's status is not success"))
	}

	return constructedReceipt, nil
}

func GetEVMHeaderByNumber(blockNumber *big.Int, host string) (*RpcBlock, error) {
	rpcClient := rpccaller.NewRPCClient()
	getEVMHeaderByNumberParams := []interface{}{fmt.Sprintf("0x%x", blockNumber), true}
	var getEVMHeaderByNumberRes GetEVMBlockByNumberRes
	err := rpcClient.RPCCall(
		"",
		host,
		"",
		"eth_getBlockByNumber",
		getEVMHeaderByNumberParams,
		&getEVMHeaderByNumberRes,
	)
	if err != nil {
		return nil, err
	}
	if getEVMHeaderByNumberRes.RPCError != nil {
		return nil, errors.New(fmt.Sprintf("An error occured during calling eth_getBlockByNumber: %s", getEVMHeaderByNumberRes.RPCError.Message))
	}

	if getEVMHeaderByNumberRes.Result == nil {
		return nil, errors.New(fmt.Sprintf("An error occured during calling eth_getBlockByNumber: result is nil"))
	}

	return getEVMHeaderByNumberRes.Result, nil
}
