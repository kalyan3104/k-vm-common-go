package datafield

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"unicode"

	"github.com/kalyan3104/k-core/core"
)

const (
	dctIdentifierSeparator  = "-"
	dctRandomSequenceLength = 6
)

func getAllBuiltInFunctions() []string {
	return []string{
		core.BuiltInFunctionClaimDeveloperRewards,
		core.BuiltInFunctionChangeOwnerAddress,
		core.BuiltInFunctionSetUserName,
		core.BuiltInFunctionSaveKeyValue,
		core.BuiltInFunctionDCTTransfer,
		core.BuiltInFunctionDCTBurn,
		core.BuiltInFunctionDCTFreeze,
		core.BuiltInFunctionDCTUnFreeze,
		core.BuiltInFunctionDCTWipe,
		core.BuiltInFunctionDCTPause,
		core.BuiltInFunctionDCTUnPause,
		core.BuiltInFunctionSetDCTRole,
		core.BuiltInFunctionUnSetDCTRole,
		core.BuiltInFunctionDCTSetLimitedTransfer,
		core.BuiltInFunctionDCTUnSetLimitedTransfer,
		core.BuiltInFunctionDCTLocalMint,
		core.BuiltInFunctionDCTLocalBurn,
		core.BuiltInFunctionDCTNFTTransfer,
		core.BuiltInFunctionDCTNFTCreate,
		core.BuiltInFunctionDCTNFTAddQuantity,
		core.BuiltInFunctionDCTNFTCreateRoleTransfer,
		core.BuiltInFunctionDCTNFTBurn,
		core.BuiltInFunctionDCTNFTAddURI,
		core.BuiltInFunctionDCTNFTUpdateAttributes,
		core.BuiltInFunctionMultiDCTNFTTransfer,
		core.DCTRoleLocalMint,
		core.DCTRoleLocalBurn,
		core.DCTRoleNFTCreate,
		core.DCTRoleNFTCreateMultiShard,
		core.DCTRoleNFTAddQuantity,
		core.DCTRoleNFTBurn,
		core.DCTRoleNFTAddURI,
		core.DCTRoleNFTUpdateAttributes,
		core.DCTRoleTransfer,
	}
}

func isBuiltInFunction(builtInFunctionsList []string, function string) bool {
	for _, builtInFunction := range builtInFunctionsList {
		if builtInFunction == function {
			return true
		}
	}

	return false
}

// EncodeBytesSlice will encode the provided bytes slice with a provided function
func EncodeBytesSlice(encodeFunc func(b []byte) string, rcvs [][]byte) []string {
	if encodeFunc == nil {
		return nil
	}

	encodedSlice := make([]string, 0, len(rcvs))
	for _, rcv := range rcvs {
		encodedSlice = append(encodedSlice, encodeFunc(rcv))
	}

	return encodedSlice
}

func computeTokenIdentifier(token string, nonce uint64) string {
	if token == "" || nonce == 0 {
		return ""
	}

	nonceBig := big.NewInt(0).SetUint64(nonce)
	hexEncodedNonce := hex.EncodeToString(nonceBig.Bytes())
	return fmt.Sprintf("%s-%s", token, hexEncodedNonce)
}

func extractTokenAndNonce(arg []byte) (string, uint64) {
	argsSplit := bytes.Split(arg, []byte(dctIdentifierSeparator))
	if len(argsSplit) < 2 {
		return string(arg), 0
	}

	if len(argsSplit[1]) <= dctRandomSequenceLength {
		return string(arg), 0
	}

	identifier := []byte(fmt.Sprintf("%s-%s", argsSplit[0], argsSplit[1][:dctRandomSequenceLength]))
	nonce := big.NewInt(0).SetBytes(argsSplit[1][dctRandomSequenceLength:])

	return string(identifier), nonce.Uint64()
}

func isEmptyAddr(addrLength int, address []byte) bool {
	emptyAddr := make([]byte, addrLength)

	return bytes.Equal(address, emptyAddr)
}

func isASCIIString(input string) bool {
	for i := 0; i < len(input); i++ {
		if input[i] > unicode.MaxASCII {
			return false
		}
	}

	return true
}
