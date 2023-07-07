package datafield

import (
	"github.com/kalyan3104/k-core/core"
	vmcommon "github.com/kalyan3104/k-vm-common-go"
)

func (odp *operationDataFieldParser) parseSingleDCTTransfer(args [][]byte, function string, sender, receiver []byte) *ResponseParseData {
	responseParse, parsedDCTTransfers, ok := odp.extractDCTData(args, function, sender, receiver)
	if !ok {
		return responseParse
	}

	if core.IsSmartContractAddress(receiver) && isASCIIString(parsedDCTTransfers.CallFunction) {
		responseParse.Function = parsedDCTTransfers.CallFunction
	}

	if len(parsedDCTTransfers.DCTTransfers) == 0 || !isASCIIString(string(parsedDCTTransfers.DCTTransfers[0].DCTTokenName)) {
		return responseParse
	}

	firstTransfer := parsedDCTTransfers.DCTTransfers[0]
	responseParse.Tokens = append(responseParse.Tokens, string(firstTransfer.DCTTokenName))
	responseParse.DCTValues = append(responseParse.DCTValues, firstTransfer.DCTValue.String())

	return responseParse
}

func (odp *operationDataFieldParser) extractDCTData(args [][]byte, function string, sender, receiver []byte) (*ResponseParseData, *vmcommon.ParsedDCTTransfers, bool) {
	responseParse := &ResponseParseData{
		Operation: function,
	}

	parsedDCTTransfers, err := odp.dctTransferParser.ParseDCTTransfers(sender, receiver, function, args)
	if err != nil {
		return responseParse, nil, false
	}

	return responseParse, parsedDCTTransfers, true
}
