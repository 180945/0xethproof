package main

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/incognitochain/bridge-eth/rpccaller"
	"github.com/pkg/errors"
	"math/big"
	"strconv"
	"strings"
)

func main() {

	//txHash := common.HexToHash("0xb23bd58950e36cc6505749d15a0bb043c3623bf85bec1e4a424283fb03b5b90f")
	//
	rpc := "https://1rpc.io/eth"
	//
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
	//
	//fmt.Println(res)
	//
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

	blockByNum, err := GetEVMHeaderByNumber(big.NewInt(19435758), rpc)
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
		var tx *rpcTransaction
		err = json.Unmarshal(jsonData, &tx)
		if err != nil {
			panic(err)
		}
		fmt.Println("--------------------------------")
		fmt.Println(tx.tx.Hash().String())
		fmt.Println(tx.tx.To())
		fmt.Println(tx.From)
	}

	client, err := ethclient.Dial(rpc)
	if err != nil {
		panic(err)
	}

	newOwner := []common.Address{
		common.HexToAddress("0xb1C398DDf9f45eAcdB42487347950a54Cb0fB02F"),
		common.HexToAddress("0x3fC91A3afd70395Cd496C647d5a6CC9D4B2b7FAD"),
		common.HexToAddress("0xF3fDcfbfdB96315FC628854627BDd5e363b3aDE4"),
		common.HexToAddress("0x3F93129a61c0c60759a9A871DC683AE58E55209F"),
		common.HexToAddress("0x70e8df8e5887b6Ca5A118B06E132fbBE69f0f736"),
		common.HexToAddress("0xb1C398DDf9f45eAcdB42487347950a54Cb0fB02F"),
		common.HexToAddress("0x073E9D8d85179d573b7C2fa99770b5eADDCD92a0"),
	}

	swapOwner(newOwner, common.HexToAddress("0x5b6e24479811E7edac7A5dBbE115E5c0b5D8effB"), common.HexToAddress("0x38869bf66a61cF6bDB996A6aE40D5853Fd43B526"), client)
}

func swapOwner(newOwners []common.Address, safeAddr common.Address, multisendAddr common.Address, client *ethclient.Client) {
	prevOwner := common.HexToAddress("0x0000000000000000000000000000000000000001")
	safeAbi, _ := abi.JSON(strings.NewReader(SafeMetaData.ABI))
	multiSendAbi, _ := abi.JSON(strings.NewReader(MultisendMetaData.ABI))

	safeContract, _ := NewSafe(safeAddr, client)
	oldOwners, _ := safeContract.GetOwners(nil)

	var swapOwnerCallData []byte
	for i, v := range newOwners {
		swapOwnerTmp, err := safeAbi.Pack("swapOwner", prevOwner, oldOwners[i], v)
		if err != nil {
			panic(err.Error())
		}

		txData := []byte{}
		txData = append(txData, byte(0))
		txData = append(txData, safeAddr.Bytes()...)
		temp := toByte32(big.NewInt(0).Bytes())
		txData = append(txData, temp[:]...)
		temp2 := make([]byte, 32)
		temp3 := big.NewInt(int64(len(swapOwnerTmp))).Bytes()
		temp2 = append(temp2, temp3...)
		txData = append(txData, temp2[(len(temp3)):]...)
		txData = append(txData, swapOwnerTmp...)

		if err != nil {
			panic(err)
		}

		swapOwnerCallData = append(swapOwnerCallData, txData...)
	}

	swapOwnerCallData, err := multiSendAbi.Pack("multiSend", swapOwnerCallData)
	if err != nil {
		panic(err)
	}

	nonce, err := safeContract.Nonce(nil)
	if err != nil {
		panic(err)
	}

	encodeTx, err := safeContract.EncodeTransactionData(
		nil,
		multisendAddr,
		big.NewInt(0),
		swapOwnerCallData,
		1,
		big.NewInt(0),
		big.NewInt(0),
		big.NewInt(0),
		common.Address{},
		common.Address{},
		nonce,
	)

	if err != nil {
		panic(err)
	}

	//fmt.Println(hex.EncodeToString(swapOwnerCallData))
	fmt.Println("--------------------------------")
	fmt.Println(hex.EncodeToString(encodeTx))

	// sign tx
	// execute transaction
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

func toByte32(s []byte) [32]byte {
	a := [32]byte{}
	copy(a[:], s)
	return a
}
