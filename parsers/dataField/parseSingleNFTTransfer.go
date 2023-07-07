package datafield

import (
	"bytes"

	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/sharding"
)

func (odp *operationDataFieldParser) parseSingleDCTNFTTransfer(args [][]byte, function string, sender, receiver []byte, numOfShards uint32) *ResponseParseData {
	responseParse, parsedDCTTransfers, ok := odp.extractDCTData(args, function, sender, receiver)
	if !ok {
		return responseParse
	}

	if core.IsSmartContractAddress(parsedDCTTransfers.RcvAddr) && isASCIIString(parsedDCTTransfers.CallFunction) {
		responseParse.Function = parsedDCTTransfers.CallFunction
	}

	if len(parsedDCTTransfers.DCTTransfers) == 0 || !isASCIIString(string(parsedDCTTransfers.DCTTransfers[0].DCTTokenName)) {
		return responseParse
	}

	rcvAddr := receiver
	if bytes.Equal(sender, receiver) {
		rcvAddr = parsedDCTTransfers.RcvAddr
	}

	dctNFTTransfer := parsedDCTTransfers.DCTTransfers[0]
	receiverShardID := sharding.ComputeShardID(rcvAddr, numOfShards)
	token := computeTokenIdentifier(string(dctNFTTransfer.DCTTokenName), dctNFTTransfer.DCTTokenNonce)

	responseParse.Tokens = append(responseParse.Tokens, token)
	responseParse.DCTValues = append(responseParse.DCTValues, dctNFTTransfer.DCTValue.String())

	if len(rcvAddr) != len(sender) {
		return responseParse
	}

	responseParse.Receivers = append(responseParse.Receivers, rcvAddr)
	responseParse.ReceiversShardID = append(responseParse.ReceiversShardID, receiverShardID)

	return responseParse
}
