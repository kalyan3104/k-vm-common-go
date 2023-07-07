package parsers

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/data/dct"
	"github.com/kalyan3104/k-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
)

var sndAddr = bytes.Repeat([]byte{1}, 32)
var dstAddr = bytes.Repeat([]byte{1}, 32)

func TestNewDCTTransferParser(t *testing.T) {
	t.Parallel()

	dctParser, err := NewDCTTransferParser(nil)
	assert.Nil(t, dctParser)
	assert.Equal(t, err, ErrNilMarshalizer)

	dctParser, err = NewDCTTransferParser(&mock.MarshalizerMock{})
	assert.Nil(t, err)
	assert.False(t, dctParser.IsInterfaceNil())
}

func TestDctTransferParser_ParseDCTTransfersWrongFunction(t *testing.T) {
	t.Parallel()

	dctParser, _ := NewDCTTransferParser(&mock.MarshalizerMock{})
	parsedData, err := dctParser.ParseDCTTransfers(nil, nil, "some", nil)
	assert.Equal(t, err, ErrNotDCTTransferInput)
	assert.Nil(t, parsedData)
}

func TestDctTransferParser_ParseSingleDCTFunction(t *testing.T) {
	t.Parallel()

	dctParser, _ := NewDCTTransferParser(&mock.MarshalizerMock{})
	parsedData, err := dctParser.ParseDCTTransfers(
		nil,
		dstAddr,
		core.BuiltInFunctionDCTTransfer,
		[][]byte{[]byte("one")},
	)
	assert.Equal(t, err, ErrNotEnoughArguments)
	assert.Nil(t, parsedData)

	parsedData, err = dctParser.ParseDCTTransfers(
		nil,
		dstAddr,
		core.BuiltInFunctionDCTTransfer,
		[][]byte{[]byte("one"), big.NewInt(10).Bytes()},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 0)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.DCTTransfers[0].DCTValue.Uint64(), big.NewInt(10).Uint64())

	parsedData, err = dctParser.ParseDCTTransfers(
		nil,
		dstAddr,
		core.BuiltInFunctionDCTTransfer,
		[][]byte{[]byte("one"), big.NewInt(10).Bytes(), []byte("function"), []byte("arg")},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.CallFunction, "function")
}

func TestDctTransferParser_ParseSingleNFTTransfer(t *testing.T) {
	t.Parallel()

	dctParser, _ := NewDCTTransferParser(&mock.MarshalizerMock{})
	parsedData, err := dctParser.ParseDCTTransfers(
		nil,
		dstAddr,
		core.BuiltInFunctionDCTNFTTransfer,
		[][]byte{[]byte("one"), []byte("two")},
	)
	assert.Equal(t, err, ErrNotEnoughArguments)
	assert.Nil(t, parsedData)

	parsedData, err = dctParser.ParseDCTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionDCTNFTTransfer,
		[][]byte{[]byte("one"), big.NewInt(10).Bytes(), big.NewInt(10).Bytes(), dstAddr},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 0)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.DCTTransfers[0].DCTValue.Uint64(), big.NewInt(10).Uint64())
	assert.Equal(t, parsedData.DCTTransfers[0].DCTTokenNonce, big.NewInt(10).Uint64())

	parsedData, err = dctParser.ParseDCTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionDCTNFTTransfer,
		[][]byte{[]byte("one"), big.NewInt(10).Bytes(), big.NewInt(10).Bytes(), dstAddr, []byte("function"), []byte("arg")})
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.CallFunction, "function")

	parsedData, err = dctParser.ParseDCTTransfers(
		sndAddr,
		dstAddr,
		core.BuiltInFunctionDCTNFTTransfer,
		[][]byte{[]byte("one"), big.NewInt(10).Bytes(), big.NewInt(10).Bytes(), dstAddr, []byte("function"), []byte("arg")})
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.DCTTransfers[0].DCTValue.Uint64(), big.NewInt(10).Uint64())
	assert.Equal(t, parsedData.DCTTransfers[0].DCTTokenNonce, big.NewInt(10).Uint64())
}

func TestDctTransferParser_ParseMultiNFTTransferTransferOne(t *testing.T) {
	t.Parallel()

	dctParser, _ := NewDCTTransferParser(&mock.MarshalizerMock{})
	parsedData, err := dctParser.ParseDCTTransfers(
		nil,
		sndAddr,
		core.BuiltInFunctionMultiDCTNFTTransfer,
		[][]byte{[]byte("one"), []byte("two")},
	)
	assert.Equal(t, err, ErrNotEnoughArguments)
	assert.Nil(t, parsedData)

	parsedData, err = dctParser.ParseDCTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiDCTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(1).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes()},
	)
	assert.Equal(t, err, ErrNotEnoughArguments)
	assert.Nil(t, parsedData)

	parsedData, err = dctParser.ParseDCTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiDCTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(1).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), big.NewInt(20).Bytes()},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 0)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.DCTTransfers[0].DCTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.DCTTransfers[0].DCTTokenNonce, big.NewInt(10).Uint64())

	parsedData, err = dctParser.ParseDCTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiDCTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(1).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), big.NewInt(20).Bytes(), []byte("function"), []byte("arg")})
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.CallFunction, "function")

	dctData := &dct.DCToken{Value: big.NewInt(20)}
	marshaled, _ := dctParser.marshaller.Marshal(dctData)

	parsedData, err = dctParser.ParseDCTTransfers(
		sndAddr,
		dstAddr,
		core.BuiltInFunctionMultiDCTNFTTransfer,
		[][]byte{big.NewInt(1).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), marshaled, []byte("function"), []byte("arg")})
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.DCTTransfers[0].DCTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.DCTTransfers[0].DCTTokenNonce, big.NewInt(10).Uint64())
}

func TestDctTransferParser_ParseMultiNFTTransferTransferMore(t *testing.T) {
	t.Parallel()

	dctParser, _ := NewDCTTransferParser(&mock.MarshalizerMock{})
	parsedData, err := dctParser.ParseDCTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiDCTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(2).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), big.NewInt(20).Bytes()},
	)
	assert.Equal(t, err, ErrNotEnoughArguments)
	assert.Nil(t, parsedData)

	parsedData, err = dctParser.ParseDCTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiDCTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(2).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), big.NewInt(20).Bytes(), []byte("tokenID"), big.NewInt(0).Bytes(), big.NewInt(20).Bytes()},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCTTransfers), 2)
	assert.Equal(t, len(parsedData.CallArgs), 0)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.DCTTransfers[0].DCTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.DCTTransfers[0].DCTTokenNonce, big.NewInt(10).Uint64())
	assert.Equal(t, parsedData.DCTTransfers[1].DCTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.DCTTransfers[1].DCTTokenNonce, uint64(0))
	assert.Equal(t, parsedData.DCTTransfers[1].DCTTokenType, uint32(core.Fungible))

	parsedData, err = dctParser.ParseDCTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiDCTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(2).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), big.NewInt(20).Bytes(), []byte("tokenID"), big.NewInt(0).Bytes(), big.NewInt(20).Bytes(), []byte("function"), []byte("arg")},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCTTransfers), 2)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.CallFunction, "function")

	dctData := &dct.DCToken{Value: big.NewInt(20)}
	marshaled, _ := dctParser.marshaller.Marshal(dctData)
	parsedData, err = dctParser.ParseDCTTransfers(
		sndAddr,
		dstAddr,
		core.BuiltInFunctionMultiDCTNFTTransfer,
		[][]byte{big.NewInt(2).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), marshaled, []byte("tokenID"), big.NewInt(0).Bytes(), big.NewInt(20).Bytes()},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCTTransfers), 2)
	assert.Equal(t, len(parsedData.CallArgs), 0)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.DCTTransfers[0].DCTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.DCTTransfers[0].DCTTokenNonce, big.NewInt(10).Uint64())
	assert.Equal(t, parsedData.DCTTransfers[1].DCTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.DCTTransfers[1].DCTTokenNonce, uint64(0))
	assert.Equal(t, parsedData.DCTTransfers[1].DCTTokenType, uint32(core.Fungible))

	parsedData, err = dctParser.ParseDCTTransfers(
		sndAddr,
		dstAddr,
		core.BuiltInFunctionMultiDCTNFTTransfer,
		[][]byte{big.NewInt(2).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), marshaled, []byte("tokenID"), big.NewInt(0).Bytes(), big.NewInt(20).Bytes(), []byte("function"), []byte("arg")},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCTTransfers), 2)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.CallFunction, "function")
}
