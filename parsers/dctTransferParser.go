package parsers

import (
	"bytes"
	"math/big"

	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/check"
	"github.com/kalyan3104/k-core/data/dct"
	vmcommon "github.com/kalyan3104/k-vm-common-go"
)

// MinArgsForDCTTransfer defines the minimum arguments needed for an dct transfer
const MinArgsForDCTTransfer = 2

// MinArgsForDCTNFTTransfer defines the minimum arguments needed for an nft transfer
const MinArgsForDCTNFTTransfer = 4

// MinArgsForMultiDCTNFTTransfer defines the minimum arguments needed for a multi transfer
const MinArgsForMultiDCTNFTTransfer = 4

// ArgsPerTransfer defines the number of arguments per transfer in multi transfer
const ArgsPerTransfer = 3

type dctTransferParser struct {
	marshaller vmcommon.Marshalizer
}

// NewDCTTransferParser creates a new dct transfer parser
func NewDCTTransferParser(
	marshaller vmcommon.Marshalizer,
) (*dctTransferParser, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}

	return &dctTransferParser{marshaller: marshaller}, nil
}

// ParseDCTTransfers returns the list of dct transfers, the callFunction and callArgs from the given arguments
func (e *dctTransferParser) ParseDCTTransfers(
	sndAddr []byte,
	rcvAddr []byte,
	function string,
	args [][]byte,
) (*vmcommon.ParsedDCTTransfers, error) {
	switch function {
	case core.BuiltInFunctionDCTTransfer:
		return e.parseSingleDCTTransfer(rcvAddr, args)
	case core.BuiltInFunctionDCTNFTTransfer:
		return e.parseSingleDCTNFTTransfer(sndAddr, rcvAddr, args)
	case core.BuiltInFunctionMultiDCTNFTTransfer:
		return e.parseMultiDCTNFTTransfer(rcvAddr, args)
	default:
		return nil, ErrNotDCTTransferInput
	}
}

func (e *dctTransferParser) parseSingleDCTTransfer(rcvAddr []byte, args [][]byte) (*vmcommon.ParsedDCTTransfers, error) {
	if len(args) < MinArgsForDCTTransfer {
		return nil, ErrNotEnoughArguments
	}
	dctTransfers := &vmcommon.ParsedDCTTransfers{
		DCTTransfers: make([]*vmcommon.DCTTransfer, 1),
		RcvAddr:      rcvAddr,
		CallArgs:     make([][]byte, 0),
		CallFunction: "",
	}
	if len(args) > MinArgsForDCTTransfer {
		dctTransfers.CallFunction = string(args[MinArgsForDCTTransfer])
	}
	if len(args) > MinArgsForDCTTransfer+1 {
		dctTransfers.CallArgs = append(dctTransfers.CallArgs, args[MinArgsForDCTTransfer+1:]...)
	}
	dctTransfers.DCTTransfers[0] = &vmcommon.DCTTransfer{
		DCTValue:      big.NewInt(0).SetBytes(args[1]),
		DCTTokenName:  args[0],
		DCTTokenType:  uint32(core.Fungible),
		DCTTokenNonce: 0,
	}

	return dctTransfers, nil
}

func (e *dctTransferParser) parseSingleDCTNFTTransfer(sndAddr, rcvAddr []byte, args [][]byte) (*vmcommon.ParsedDCTTransfers, error) {
	if len(args) < MinArgsForDCTNFTTransfer {
		return nil, ErrNotEnoughArguments
	}
	dctTransfers := &vmcommon.ParsedDCTTransfers{
		DCTTransfers: make([]*vmcommon.DCTTransfer, 1),
		RcvAddr:      rcvAddr,
		CallArgs:     make([][]byte, 0),
		CallFunction: "",
	}

	if bytes.Equal(sndAddr, rcvAddr) {
		dctTransfers.RcvAddr = args[3]
	}
	if len(args) > MinArgsForDCTNFTTransfer {
		dctTransfers.CallFunction = string(args[MinArgsForDCTNFTTransfer])
	}
	if len(args) > MinArgsForDCTNFTTransfer+1 {
		dctTransfers.CallArgs = append(dctTransfers.CallArgs, args[MinArgsForDCTNFTTransfer+1:]...)
	}
	dctTransfers.DCTTransfers[0] = &vmcommon.DCTTransfer{
		DCTValue:      big.NewInt(0).SetBytes(args[2]),
		DCTTokenName:  args[0],
		DCTTokenType:  uint32(core.NonFungible),
		DCTTokenNonce: big.NewInt(0).SetBytes(args[1]).Uint64(),
	}

	return dctTransfers, nil
}

func (e *dctTransferParser) parseMultiDCTNFTTransfer(rcvAddr []byte, args [][]byte) (*vmcommon.ParsedDCTTransfers, error) {
	if len(args) < MinArgsForMultiDCTNFTTransfer {
		return nil, ErrNotEnoughArguments
	}
	dctTransfers := &vmcommon.ParsedDCTTransfers{
		RcvAddr:      rcvAddr,
		CallArgs:     make([][]byte, 0),
		CallFunction: "",
	}

	numOfTransfer := big.NewInt(0).SetBytes(args[0])
	startIndex := uint64(1)
	isTxAtSender := false

	isFirstArgumentAnAddress := len(args[0]) == len(rcvAddr) && !numOfTransfer.IsUint64()
	if isFirstArgumentAnAddress {
		dctTransfers.RcvAddr = args[0]
		numOfTransfer.SetBytes(args[1])
		startIndex = 2
		isTxAtSender = true
	}

	minLenArgs := ArgsPerTransfer*numOfTransfer.Uint64() + startIndex
	if uint64(len(args)) < minLenArgs {
		return nil, ErrNotEnoughArguments
	}

	if uint64(len(args)) > minLenArgs {
		dctTransfers.CallFunction = string(args[minLenArgs])
	}
	if uint64(len(args)) > minLenArgs+1 {
		dctTransfers.CallArgs = append(dctTransfers.CallArgs, args[minLenArgs+1:]...)
	}

	var err error
	dctTransfers.DCTTransfers = make([]*vmcommon.DCTTransfer, numOfTransfer.Uint64())
	for i := uint64(0); i < numOfTransfer.Uint64(); i++ {
		tokenStartIndex := startIndex + i*ArgsPerTransfer
		dctTransfers.DCTTransfers[i], err = e.createNewDCTTransfer(tokenStartIndex, args, isTxAtSender)
		if err != nil {
			return nil, err
		}
	}

	return dctTransfers, nil
}

func (e *dctTransferParser) createNewDCTTransfer(
	tokenStartIndex uint64,
	args [][]byte,
	isTxAtSender bool,
) (*vmcommon.DCTTransfer, error) {
	dctTransfer := &vmcommon.DCTTransfer{
		DCTValue:      big.NewInt(0).SetBytes(args[tokenStartIndex+2]),
		DCTTokenName:  args[tokenStartIndex],
		DCTTokenType:  uint32(core.Fungible),
		DCTTokenNonce: big.NewInt(0).SetBytes(args[tokenStartIndex+1]).Uint64(),
	}
	if dctTransfer.DCTTokenNonce > 0 {
		dctTransfer.DCTTokenType = uint32(core.NonFungible)

		if !isTxAtSender && len(args[tokenStartIndex+2]) > vmcommon.MaxLengthForValueToOptTransfer {
			transferDCTData := &dct.DCToken{}
			err := e.marshaller.Unmarshal(transferDCTData, args[tokenStartIndex+2])
			if err != nil {
				return nil, err
			}
			dctTransfer.DCTValue.Set(transferDCTData.Value)
		}
	}

	return dctTransfer, nil
}

// IsInterfaceNil returns true if underlying object is nil
func (e *dctTransferParser) IsInterfaceNil() bool {
	return e == nil
}
