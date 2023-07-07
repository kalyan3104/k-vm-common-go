package datafield

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/check"
	"github.com/kalyan3104/k-core/core/sharding"
	"github.com/kalyan3104/k-core/data/transaction"
	vmcommon "github.com/kalyan3104/k-vm-common-go"
	"github.com/kalyan3104/k-vm-common-go/parsers"
)

const (
	operationTransfer = `transfer`
	operationDeploy   = `scDeploy`

	minArgumentsQuantityOperationDCT = 2
	minArgumentsQuantityOperationNFT = 3
	numArgsRelayedV2                 = 4
	receiverAddressIndexRelayedV2    = 0
	dataFieldIndexRelayedV2          = 2

	argsTokenPosition                   = 0
	argsNoncePosition                   = 1
	argsValuePositionNonAndSemiFungible = 2
	argsValuePositionFungible           = 1
)

var errInvalidAddressLength = errors.New("invalid address length")

type operationDataFieldParser struct {
	builtInFunctionsList []string

	addressLength     int
	argsParser        vmcommon.CallArgsParser
	dctTransferParser vmcommon.DCTTransferParser
}

// NewOperationDataFieldParser will return a new instance of operationDataFieldParser
func NewOperationDataFieldParser(args *ArgsOperationDataFieldParser) (*operationDataFieldParser, error) {
	if check.IfNil(args.Marshalizer) {
		return nil, core.ErrNilMarshalizer
	}
	if args.AddressLength == 0 {
		return nil, errInvalidAddressLength
	}

	argsParser := parsers.NewCallArgsParser()
	dctTransferParser, err := parsers.NewDCTTransferParser(args.Marshalizer)
	if err != nil {
		return nil, err
	}

	return &operationDataFieldParser{
		argsParser:           argsParser,
		dctTransferParser:    dctTransferParser,
		addressLength:        args.AddressLength,
		builtInFunctionsList: getAllBuiltInFunctions(),
	}, nil
}

// Parse will parse the provided data field
func (odp *operationDataFieldParser) Parse(dataField []byte, sender, receiver []byte, numOfShards uint32) *ResponseParseData {
	return odp.parse(dataField, sender, receiver, false, numOfShards)
}

func (odp *operationDataFieldParser) parse(dataField []byte, sender, receiver []byte, ignoreRelayed bool, numOfShards uint32) *ResponseParseData {
	responseParse := &ResponseParseData{
		Operation: operationTransfer,
	}

	isSCDeploy := len(dataField) > 0 && isEmptyAddr(odp.addressLength, receiver)
	if isSCDeploy {
		responseParse.Operation = operationDeploy
		return responseParse
	}

	function, args, err := odp.argsParser.ParseData(string(dataField))
	if err != nil {
		return responseParse
	}

	switch function {
	case core.BuiltInFunctionDCTTransfer:
		return odp.parseSingleDCTTransfer(args, function, sender, receiver)
	case core.BuiltInFunctionDCTNFTTransfer:
		return odp.parseSingleDCTNFTTransfer(args, function, sender, receiver, numOfShards)
	case core.BuiltInFunctionMultiDCTNFTTransfer:
		return odp.parseMultiDCTNFTTransfer(args, function, sender, receiver, numOfShards)
	case core.BuiltInFunctionDCTLocalBurn, core.BuiltInFunctionDCTLocalMint:
		return parseQuantityOperationDCT(args, function)
	case core.BuiltInFunctionDCTWipe, core.BuiltInFunctionDCTFreeze, core.BuiltInFunctionDCTUnFreeze:
		return parseBlockingOperationDCT(args, function)
	case core.BuiltInFunctionDCTNFTCreate, core.BuiltInFunctionDCTNFTBurn, core.BuiltInFunctionDCTNFTAddQuantity:
		return parseQuantityOperationNFT(args, function)
	case core.RelayedTransaction, core.RelayedTransactionV2:
		if ignoreRelayed {
			return NewResponseParseDataAsRelayed()
		}
		return odp.parseRelayed(function, args, receiver, numOfShards)
	}

	isBuiltInFunc := isBuiltInFunction(odp.builtInFunctionsList, function)
	if isBuiltInFunc {
		responseParse.Operation = function
	}

	if function != "" && core.IsSmartContractAddress(receiver) && isASCIIString(function) {
		responseParse.Function = function
	}

	return responseParse
}

func (odp *operationDataFieldParser) parseRelayed(function string, args [][]byte, receiver []byte, numOfShards uint32) *ResponseParseData {
	if len(args) == 0 {
		return &ResponseParseData{
			IsRelayed: true,
		}
	}

	tx, ok := extractInnerTx(function, args, receiver)
	if !ok {
		return &ResponseParseData{
			IsRelayed: true,
		}
	}

	res := odp.parse(tx.Data, tx.SndAddr, tx.RcvAddr, true, numOfShards)
	if res.IsRelayed {
		return &ResponseParseData{
			IsRelayed: true,
		}
	}

	receivers := [][]byte{tx.RcvAddr}
	receiversShardID := []uint32{sharding.ComputeShardID(tx.RcvAddr, numOfShards)}
	if res.Operation == core.BuiltInFunctionMultiDCTNFTTransfer || res.Operation == core.BuiltInFunctionDCTNFTTransfer {
		receivers = res.Receivers
		receiversShardID = res.ReceiversShardID
	}

	return &ResponseParseData{
		Operation:        res.Operation,
		Function:         res.Function,
		DCTValues:        res.DCTValues,
		Tokens:           res.Tokens,
		Receivers:        receivers,
		ReceiversShardID: receiversShardID,
		IsRelayed:        true,
	}
}

func extractInnerTx(function string, args [][]byte, receiver []byte) (*transaction.Transaction, bool) {
	tx := &transaction.Transaction{}

	if function == core.RelayedTransaction {
		err := json.Unmarshal(args[0], &tx)

		return tx, err == nil
	}

	if len(args) != numArgsRelayedV2 {
		return nil, false
	}

	// sender of the inner tx is the receiver of the relayed tx
	tx.SndAddr = receiver
	tx.RcvAddr = args[receiverAddressIndexRelayedV2]
	tx.Data = args[dataFieldIndexRelayedV2]

	return tx, true
}

func parseBlockingOperationDCT(args [][]byte, funcName string) *ResponseParseData {
	responseData := &ResponseParseData{
		Operation: funcName,
	}

	if len(args) == 0 {
		return responseData
	}

	token, nonce := extractTokenAndNonce(args[argsTokenPosition])
	if !isASCIIString(token) {
		return responseData
	}

	if nonce != 0 {
		token = computeTokenIdentifier(token, nonce)
	}

	responseData.Tokens = append(responseData.Tokens, token)
	return responseData
}

func parseQuantityOperationDCT(args [][]byte, funcName string) *ResponseParseData {
	responseData := &ResponseParseData{
		Operation: funcName,
	}

	if len(args) < minArgumentsQuantityOperationDCT {
		return responseData
	}

	token := string(args[argsTokenPosition])
	if !isASCIIString(token) {
		return responseData
	}

	responseData.Tokens = append(responseData.Tokens, token)
	responseData.DCTValues = append(responseData.DCTValues, big.NewInt(0).SetBytes(args[argsValuePositionFungible]).String())

	return responseData
}

func parseQuantityOperationNFT(args [][]byte, funcName string) *ResponseParseData {
	responseData := &ResponseParseData{
		Operation: funcName,
	}

	if len(args) < minArgumentsQuantityOperationNFT {
		return responseData
	}

	token := string(args[argsTokenPosition])
	if !isASCIIString(token) {
		return responseData
	}

	nonce := big.NewInt(0).SetBytes(args[argsNoncePosition]).Uint64()
	tokenIdentifier := computeTokenIdentifier(token, nonce)

	value := big.NewInt(0).SetBytes(args[argsValuePositionNonAndSemiFungible]).String()
	if funcName == core.BuiltInFunctionDCTNFTCreate {
		value = big.NewInt(0).SetBytes(args[argsValuePositionNonAndSemiFungible-1]).String()
		tokenIdentifier = token
	}

	responseData.DCTValues = append(responseData.DCTValues, value)
	responseData.Tokens = append(responseData.Tokens, tokenIdentifier)

	return responseData
}
