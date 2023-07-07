package builtInFunctions

import (
	"bytes"
	"math/big"

	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/check"
	"github.com/kalyan3104/k-core/data/dct"
	vmcommon "github.com/kalyan3104/k-vm-common-go"
)

const numArgsPerAdd = 3

type dctDeleteMetaData struct {
	baseActiveHandler
	allowedAddress []byte
	delete         bool
	accounts       vmcommon.AccountsAdapter
	keyPrefix      []byte
	marshaller     vmcommon.Marshalizer
	funcGasCost    uint64
	function       string
}

// ArgsNewDCTDeleteMetadata defines the argument list for new dct delete metadata built in function
type ArgsNewDCTDeleteMetadata struct {
	FuncGasCost         uint64
	Marshalizer         vmcommon.Marshalizer
	Accounts            vmcommon.AccountsAdapter
	AllowedAddress      []byte
	Delete              bool
	EnableEpochsHandler vmcommon.EnableEpochsHandler
}

// NewDCTDeleteMetadataFunc returns the dct metadata deletion built-in function component
func NewDCTDeleteMetadataFunc(
	args ArgsNewDCTDeleteMetadata,
) (*dctDeleteMetaData, error) {
	if check.IfNil(args.Marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(args.Accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(args.EnableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}

	e := &dctDeleteMetaData{
		keyPrefix:      []byte(baseDCTKeyPrefix),
		marshaller:     args.Marshalizer,
		funcGasCost:    args.FuncGasCost,
		accounts:       args.Accounts,
		allowedAddress: args.AllowedAddress,
		delete:         args.Delete,
		function:       core.BuiltInFunctionMultiDCTNFTTransfer,
	}

	e.baseActiveHandler.activeHandler = args.EnableEpochsHandler.IsSendAlwaysFlagEnabled

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dctDeleteMetaData) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves DCT delete and add metadata function call
func (e *dctDeleteMetaData) ProcessBuiltinFunction(
	_, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	if vmInput == nil {
		return nil, ErrNilVmInput
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return nil, ErrBuiltInFunctionCalledWithValue
	}
	if !bytes.Equal(vmInput.CallerAddr, e.allowedAddress) {
		return nil, ErrAddressIsNotAllowed
	}
	if !bytes.Equal(vmInput.CallerAddr, vmInput.RecipientAddr) {
		return nil, ErrInvalidRcvAddr
	}

	if e.delete {
		err := e.deleteMetadata(vmInput.Arguments)
		if err != nil {
			return nil, err
		}
	} else {
		err := e.addMetadata(vmInput.Arguments)
		if err != nil {
			return nil, err
		}
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}

	return vmOutput, nil
}

// input is list(tokenID-numIntervals-list(start,end))
func (e *dctDeleteMetaData) deleteMetadata(args [][]byte) error {
	lenArgs := uint64(len(args))
	if lenArgs < 4 {
		return ErrInvalidNumOfArgs
	}

	systemAcc, err := e.getSystemAccount()
	if err != nil {
		return err
	}

	for i := uint64(0); i+1 < uint64(len(args)); {
		tokenID := args[i]
		numIntervals := big.NewInt(0).SetBytes(args[i+1]).Uint64()
		i += 2

		if !vmcommon.ValidateToken(tokenID) {
			return ErrInvalidTokenID
		}

		if i >= lenArgs {
			return ErrInvalidNumOfArgs
		}

		err = e.deleteMetadataForListIntervals(systemAcc, tokenID, args, i, numIntervals)
		if err != nil {
			return err
		}

		i += numIntervals * 2
	}

	err = e.accounts.SaveAccount(systemAcc)
	if err != nil {
		return err
	}

	return nil
}

func (e *dctDeleteMetaData) deleteMetadataForListIntervals(
	systemAcc vmcommon.UserAccountHandler,
	tokenID []byte,
	args [][]byte,
	index, numIntervals uint64,
) error {
	lenArgs := uint64(len(args))
	for j := index; j < index+numIntervals*2; j += 2 {
		if j > lenArgs-2 {
			return ErrInvalidNumOfArgs
		}

		startIndex := big.NewInt(0).SetBytes(args[j]).Uint64()
		endIndex := big.NewInt(0).SetBytes(args[j+1]).Uint64()

		err := e.deleteMetadataForInterval(systemAcc, tokenID, startIndex, endIndex)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *dctDeleteMetaData) deleteMetadataForInterval(
	systemAcc vmcommon.UserAccountHandler,
	tokenID []byte,
	startIndex, endIndex uint64,
) error {
	if endIndex < startIndex {
		return ErrInvalidArguments
	}
	if startIndex == 0 {
		return ErrInvalidNonce
	}

	dctTokenKey := append(e.keyPrefix, tokenID...)
	for nonce := startIndex; nonce <= endIndex; nonce++ {
		dctNFTTokenKey := computeDCTNFTTokenKey(dctTokenKey, nonce)

		err := systemAcc.AccountDataHandler().SaveKeyValue(dctNFTTokenKey, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

// input is list(tokenID-nonce-metadata)
func (e *dctDeleteMetaData) addMetadata(args [][]byte) error {
	if len(args)%numArgsPerAdd != 0 || len(args) < numArgsPerAdd {
		return ErrInvalidNumOfArgs
	}

	systemAcc, err := e.getSystemAccount()
	if err != nil {
		return err
	}

	for i := 0; i < len(args); i += numArgsPerAdd {
		tokenID := args[i]
		nonce := big.NewInt(0).SetBytes(args[i+1]).Uint64()
		if nonce == 0 {
			return ErrInvalidNonce
		}

		if !vmcommon.ValidateToken(tokenID) {
			return ErrInvalidTokenID
		}

		dctTokenKey := append(e.keyPrefix, tokenID...)
		dctNFTTokenKey := computeDCTNFTTokenKey(dctTokenKey, nonce)
		metaData := &dct.MetaData{}
		err = e.marshaller.Unmarshal(metaData, args[i+2])
		if err != nil {
			return err
		}
		if metaData.Nonce != nonce {
			return ErrInvalidMetadata
		}

		var tokenFromSystemSC *dct.DCToken
		tokenFromSystemSC, err = e.getDCTDigitalTokenDataFromSystemAccount(systemAcc, dctNFTTokenKey)
		if err != nil {
			return err
		}

		if tokenFromSystemSC != nil && tokenFromSystemSC.TokenMetaData != nil {
			return ErrTokenHasValidMetadata
		}

		if tokenFromSystemSC == nil {
			tokenFromSystemSC = &dct.DCToken{
				Value: big.NewInt(0),
				Type:  uint32(core.NonFungible),
			}
		}
		tokenFromSystemSC.TokenMetaData = metaData
		err = e.marshalAndSaveData(systemAcc, tokenFromSystemSC, dctNFTTokenKey)
		if err != nil {
			return err
		}
	}

	err = e.accounts.SaveAccount(systemAcc)
	if err != nil {
		return err
	}

	return nil
}

func (e *dctDeleteMetaData) getDCTDigitalTokenDataFromSystemAccount(
	systemAcc vmcommon.UserAccountHandler,
	dctNFTTokenKey []byte,
) (*dct.DCToken, error) {
	marshaledData, _, err := systemAcc.AccountDataHandler().RetrieveValue(dctNFTTokenKey)
	if err != nil || len(marshaledData) == 0 {
		return nil, nil
	}

	dctData := &dct.DCToken{}
	err = e.marshaller.Unmarshal(dctData, marshaledData)
	if err != nil {
		return nil, err
	}

	return dctData, nil
}

func (e *dctDeleteMetaData) getSystemAccount() (vmcommon.UserAccountHandler, error) {
	systemSCAccount, err := e.accounts.LoadAccount(vmcommon.SystemAccountAddress)
	if err != nil {
		return nil, err
	}

	userAcc, ok := systemSCAccount.(vmcommon.UserAccountHandler)
	if !ok {
		return nil, ErrWrongTypeAssertion
	}

	return userAcc, nil
}

func (e *dctDeleteMetaData) marshalAndSaveData(
	systemAcc vmcommon.UserAccountHandler,
	dctData *dct.DCToken,
	dctNFTTokenKey []byte,
) error {
	marshaledData, err := e.marshaller.Marshal(dctData)
	if err != nil {
		return err
	}

	err = systemAcc.AccountDataHandler().SaveKeyValue(dctNFTTokenKey, marshaledData)
	if err != nil {
		return err
	}

	return nil
}

// IsInterfaceNil returns true if underlying object is nil
func (e *dctDeleteMetaData) IsInterfaceNil() bool {
	return e == nil
}
