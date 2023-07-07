package builtInFunctions

import (
	"bytes"
	"math/big"

	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/check"
	vmcommon "github.com/kalyan3104/k-vm-common-go"
)

type dctFreezeWipe struct {
	baseAlwaysActiveHandler
	dctStorageHandler   vmcommon.DCTNFTStorageHandler
	enableEpochsHandler vmcommon.EnableEpochsHandler
	marshaller          vmcommon.Marshalizer
	keyPrefix           []byte
	wipe                bool
	freeze              bool
}

// NewDCTFreezeWipeFunc returns the dct freeze/un-freeze/wipe built-in function component
func NewDCTFreezeWipeFunc(
	dctStorageHandler vmcommon.DCTNFTStorageHandler,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
	marshaller vmcommon.Marshalizer,
	freeze bool,
	wipe bool,
) (*dctFreezeWipe, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(dctStorageHandler) {
		return nil, ErrNilDCTNFTStorageHandler
	}
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}

	e := &dctFreezeWipe{
		dctStorageHandler:   dctStorageHandler,
		enableEpochsHandler: enableEpochsHandler,
		marshaller:          marshaller,
		keyPrefix:           []byte(baseDCTKeyPrefix),
		freeze:              freeze,
		wipe:                wipe,
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dctFreezeWipe) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves DCT transfer function call
func (e *dctFreezeWipe) ProcessBuiltinFunction(
	_, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	if vmInput == nil {
		return nil, ErrNilVmInput
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return nil, ErrBuiltInFunctionCalledWithValue
	}
	if len(vmInput.Arguments) != 1 {
		return nil, ErrInvalidArguments
	}
	if !bytes.Equal(vmInput.CallerAddr, core.DCTSCAddress) {
		return nil, ErrAddressIsNotDCTSystemSC
	}
	if check.IfNil(acntDst) {
		return nil, ErrNilUserAccount
	}

	dctTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	identifier, nonce := extractTokenIdentifierAndNonceDCTWipe(vmInput.Arguments[0])

	var amount *big.Int
	var err error

	if e.wipe {
		amount, err = e.wipeIfApplicable(acntDst, dctTokenKey, identifier, nonce)
		if err != nil {
			return nil, err
		}

	} else {
		amount, err = e.toggleFreeze(acntDst, dctTokenKey)
		if err != nil {
			return nil, err
		}
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
	addDCTEntryInVMOutput(vmOutput, []byte(vmInput.Function), identifier, nonce, amount, vmInput.CallerAddr, acntDst.AddressBytes())

	return vmOutput, nil
}

func (e *dctFreezeWipe) wipeIfApplicable(acntDst vmcommon.UserAccountHandler, tokenKey []byte, identifier []byte, nonce uint64) (*big.Int, error) {
	tokenData, err := getDCTDataFromKey(acntDst, tokenKey, e.marshaller)
	if err != nil {
		return nil, err
	}

	dctUserMetadata := DCTUserMetadataFromBytes(tokenData.Properties)
	if !dctUserMetadata.Frozen {
		return nil, ErrCannotWipeAccountNotFrozen
	}

	err = acntDst.AccountDataHandler().SaveKeyValue(tokenKey, nil)
	if err != nil {
		return nil, err
	}

	err = e.removeLiquidity(identifier, nonce, tokenData.Value)
	if err != nil {
		return nil, err
	}

	wipedAmount := vmcommon.ZeroValueIfNil(tokenData.Value)
	return wipedAmount, nil
}

func (e *dctFreezeWipe) removeLiquidity(tokenIdentifier []byte, nonce uint64, value *big.Int) error {
	if !e.enableEpochsHandler.IsWipeSingleNFTLiquidityDecreaseEnabled() {
		return nil
	}

	tokenIDKey := append(e.keyPrefix, tokenIdentifier...)
	return e.dctStorageHandler.AddToLiquiditySystemAcc(tokenIDKey, nonce, big.NewInt(0).Neg(value))
}

func (e *dctFreezeWipe) toggleFreeze(acntDst vmcommon.UserAccountHandler, tokenKey []byte) (*big.Int, error) {
	tokenData, err := getDCTDataFromKey(acntDst, tokenKey, e.marshaller)
	if err != nil {
		return nil, err
	}

	dctUserMetadata := DCTUserMetadataFromBytes(tokenData.Properties)
	dctUserMetadata.Frozen = e.freeze
	tokenData.Properties = dctUserMetadata.ToBytes()

	err = saveDCTData(acntDst, tokenData, tokenKey, e.marshaller)
	if err != nil {
		return nil, err
	}

	frozenAmount := vmcommon.ZeroValueIfNil(tokenData.Value)
	return frozenAmount, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *dctFreezeWipe) IsInterfaceNil() bool {
	return e == nil
}
