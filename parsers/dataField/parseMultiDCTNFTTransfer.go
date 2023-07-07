package datafield

import (
	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/sharding"
)

func (odp *operationDataFieldParser) parseMultiDCTNFTTransfer(args [][]byte, function string, sender, receiver []byte, numOfShards uint32) *ResponseParseData {
	responseParse, parsedDCTTransfers, ok := odp.extractDCTData(args, function, sender, receiver)
	if !ok {
		return responseParse
	}
	if core.IsSmartContractAddress(parsedDCTTransfers.RcvAddr) && isASCIIString(parsedDCTTransfers.CallFunction) {
		responseParse.Function = parsedDCTTransfers.CallFunction
	}

	receiverShardID := sharding.ComputeShardID(parsedDCTTransfers.RcvAddr, numOfShards)
	for _, dctTransferData := range parsedDCTTransfers.DCTTransfers {
		if !isASCIIString(string(dctTransferData.DCTTokenName)) {
			return &ResponseParseData{
				Operation: function,
			}
		}

		token := string(dctTransferData.DCTTokenName)
		if dctTransferData.DCTTokenNonce != 0 {
			token = computeTokenIdentifier(token, dctTransferData.DCTTokenNonce)
		}

		responseParse.Tokens = append(responseParse.Tokens, token)
		responseParse.DCTValues = append(responseParse.DCTValues, dctTransferData.DCTValue.String())
		responseParse.Receivers = append(responseParse.Receivers, parsedDCTTransfers.RcvAddr)
		responseParse.ReceiversShardID = append(responseParse.ReceiversShardID, receiverShardID)
	}

	return responseParse
}
