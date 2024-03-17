package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/pkg/errors"
)

func main() {

	txHash := common.HexToHash("0xb23bd58950e36cc6505749d15a0bb043c3623bf85bec1e4a424283fb03b5b90f")

	rpc := "https://eth.llamarpc.com"

	_, ethBlockHash, ethTxIdx, ethDepositProof, err := getETHDepositProof(rpc, txHash)
	if err != nil {
		panic(err.Error())
	}

	fmt.Println("depositProof ---- : ", ethBlockHash, ethTxIdx, ethDepositProof)

	// verify
	res, err := VerifyProofAndParseEVMReceipt(ethBlockHash, ethTxIdx, ethDepositProof, rpc, true)
	if err != nil {
		panic(err.Error())
	}

	fmt.Println(res)
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
