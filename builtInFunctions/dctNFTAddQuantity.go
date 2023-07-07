package builtInFunctions

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/check"
	vmcommon "github.com/kalyan3104/k-vm-common-go"
)

const maxLenForAddNFTQuantity = 32

type dctNFTAddQuantity struct {
	baseAlwaysActiveHandler
	keyPrefix             []byte
	globalSettingsHandler vmcommon.DCTGlobalSettingsHandler
	rolesHandler          vmcommon.DCTRoleHandler
	dctStorageHandler     vmcommon.DCTNFTStorageHandler
	enableEpochsHandler   vmcommon.EnableEpochsHandler
	funcGasCost           uint64
	mutExecution          sync.RWMutex
}

// NewDCTNFTAddQuantityFunc returns the dct NFT add quantity built-in function component
func NewDCTNFTAddQuantityFunc(
	funcGasCost uint64,
	dctStorageHandler vmcommon.DCTNFTStorageHandler,
	globalSettingsHandler vmcommon.DCTGlobalSettingsHandler,
	rolesHandler vmcommon.DCTRoleHandler,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*dctNFTAddQuantity, error) {
	if check.IfNil(dctStorageHandler) {
		return nil, ErrNilDCTNFTStorageHandler
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(rolesHandler) {
		return nil, ErrNilRolesHandler
	}
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}

	e := &dctNFTAddQuantity{
		keyPrefix:             []byte(baseDCTKeyPrefix),
		globalSettingsHandler: globalSettingsHandler,
		rolesHandler:          rolesHandler,
		funcGasCost:           funcGasCost,
		mutExecution:          sync.RWMutex{},
		dctStorageHandler:     dctStorageHandler,
		enableEpochsHandler:   enableEpochsHandler,
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dctNFTAddQuantity) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.DCTNFTAddQuantity
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves DCT NFT add quantity function call
// Requires 3 arguments:
// arg0 - token identifier
// arg1 - nonce
// arg2 - quantity to add
func (e *dctNFTAddQuantity) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkDCTNFTCreateBurnAddInput(acntSnd, vmInput, e.funcGasCost)
	if err != nil {
		return nil, err
	}
	if len(vmInput.Arguments) < 3 {
		return nil, ErrInvalidArguments
	}

	err = e.rolesHandler.CheckAllowedToExecute(acntSnd, vmInput.Arguments[0], []byte(core.DCTRoleNFTAddQuantity))
	if err != nil {
		return nil, err
	}

	dctTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	dctData, err := e.dctStorageHandler.GetDCTNFTTokenOnSender(acntSnd, dctTokenKey, nonce)
	if err != nil {
		return nil, err
	}
	if nonce == 0 {
		return nil, ErrNFTDoesNotHaveMetadata
	}

	isValueLengthCheckFlagEnabled := e.enableEpochsHandler.IsValueLengthCheckFlagEnabled()
	if isValueLengthCheckFlagEnabled && len(vmInput.Arguments[2]) > maxLenForAddNFTQuantity {
		return nil, fmt.Errorf("%w max length for add nft quantity is %d", ErrInvalidArguments, maxLenForAddNFTQuantity)
	}

	value := big.NewInt(0).SetBytes(vmInput.Arguments[2])
	dctData.Value.Add(dctData.Value, value)

	_, err = e.dctStorageHandler.SaveDCTNFTToken(acntSnd.AddressBytes(), acntSnd, dctTokenKey, nonce, dctData, false, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}
	err = e.dctStorageHandler.AddToLiquiditySystemAcc(dctTokenKey, nonce, value)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - e.funcGasCost,
	}

	addDCTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionDCTNFTAddQuantity), vmInput.Arguments[0], nonce, value, vmInput.CallerAddr)

	return vmOutput, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *dctNFTAddQuantity) IsInterfaceNil() bool {
	return e == nil
}
