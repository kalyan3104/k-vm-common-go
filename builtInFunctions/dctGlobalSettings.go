package builtInFunctions

import (
	"bytes"

	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/check"
	"github.com/kalyan3104/k-core/marshal"
	vmcommon "github.com/kalyan3104/k-vm-common-go"
)

type dctGlobalSettings struct {
	baseActiveHandler
	keyPrefix  []byte
	set        bool
	accounts   vmcommon.AccountsAdapter
	marshaller marshal.Marshalizer
	function   string
}

// NewDCTGlobalSettingsFunc returns the dct pause/un-pause built-in function component
func NewDCTGlobalSettingsFunc(
	accounts vmcommon.AccountsAdapter,
	marshaller marshal.Marshalizer,
	set bool,
	function string,
	activeHandler func() bool,
) (*dctGlobalSettings, error) {
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if activeHandler == nil {
		return nil, ErrNilActiveHandler
	}
	if !isCorrectFunction(function) {
		return nil, ErrInvalidArguments
	}

	e := &dctGlobalSettings{
		keyPrefix:  []byte(baseDCTKeyPrefix),
		set:        set,
		accounts:   accounts,
		marshaller: marshaller,
		function:   function,
	}

	e.baseActiveHandler.activeHandler = activeHandler

	return e, nil
}

func isCorrectFunction(function string) bool {
	switch function {
	case core.BuiltInFunctionDCTPause, core.BuiltInFunctionDCTUnPause, core.BuiltInFunctionDCTSetLimitedTransfer, core.BuiltInFunctionDCTUnSetLimitedTransfer:
		return true
	case vmcommon.BuiltInFunctionDCTSetBurnRoleForAll, vmcommon.BuiltInFunctionDCTUnSetBurnRoleForAll:
		return true
	default:
		return false
	}
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dctGlobalSettings) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves DCT pause function call
func (e *dctGlobalSettings) ProcessBuiltinFunction(
	_, _ vmcommon.UserAccountHandler,
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
	if !vmcommon.IsSystemAccountAddress(vmInput.RecipientAddr) {
		return nil, ErrOnlySystemAccountAccepted
	}

	dctTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)

	err := e.toggleSetting(dctTokenKey)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
	return vmOutput, nil
}

func (e *dctGlobalSettings) toggleSetting(dctTokenKey []byte) error {
	systemSCAccount, err := e.getSystemAccount()
	if err != nil {
		return err
	}

	dctMetaData, err := e.getGlobalMetadata(dctTokenKey)
	if err != nil {
		return err
	}

	switch e.function {
	case core.BuiltInFunctionDCTSetLimitedTransfer, core.BuiltInFunctionDCTUnSetLimitedTransfer:
		dctMetaData.LimitedTransfer = e.set
		break
	case core.BuiltInFunctionDCTPause, core.BuiltInFunctionDCTUnPause:
		dctMetaData.Paused = e.set
		break
	case vmcommon.BuiltInFunctionDCTUnSetBurnRoleForAll, vmcommon.BuiltInFunctionDCTSetBurnRoleForAll:
		dctMetaData.BurnRoleForAll = e.set
		break
	}

	err = systemSCAccount.AccountDataHandler().SaveKeyValue(dctTokenKey, dctMetaData.ToBytes())
	if err != nil {
		return err
	}

	return e.accounts.SaveAccount(systemSCAccount)
}

func (e *dctGlobalSettings) getSystemAccount() (vmcommon.UserAccountHandler, error) {
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

// IsPaused returns true if the dctTokenKey (prefixed) is paused
func (e *dctGlobalSettings) IsPaused(dctTokenKey []byte) bool {
	dctMetadata, err := e.getGlobalMetadata(dctTokenKey)
	if err != nil {
		return false
	}

	return dctMetadata.Paused
}

// IsLimitedTransfer returns true if the dctTokenKey (prefixed) is with limited transfer
func (e *dctGlobalSettings) IsLimitedTransfer(dctTokenKey []byte) bool {
	dctMetadata, err := e.getGlobalMetadata(dctTokenKey)
	if err != nil {
		return false
	}

	return dctMetadata.LimitedTransfer
}

// IsBurnForAll returns true if the dctTokenKey (prefixed) is with burn for all
func (e *dctGlobalSettings) IsBurnForAll(dctTokenKey []byte) bool {
	dctMetadata, err := e.getGlobalMetadata(dctTokenKey)
	if err != nil {
		return false
	}

	return dctMetadata.BurnRoleForAll
}

// IsSenderOrDestinationWithTransferRole returns true if we have transfer role on the system account
func (e *dctGlobalSettings) IsSenderOrDestinationWithTransferRole(sender, destination, tokenID []byte) bool {
	if !e.activeHandler() {
		return false
	}

	systemAcc, err := e.getSystemAccount()
	if err != nil {
		return false
	}

	dctTokenTransferRoleKey := append(transferAddressesKeyPrefix, tokenID...)
	addresses, _, err := getDCTRolesForAcnt(e.marshaller, systemAcc, dctTokenTransferRoleKey)
	if err != nil {
		return false
	}

	for _, address := range addresses.Roles {
		if bytes.Equal(address, sender) || bytes.Equal(address, destination) {
			return true
		}
	}

	return false
}

func (e *dctGlobalSettings) getGlobalMetadata(dctTokenKey []byte) (*DCTGlobalMetadata, error) {
	systemSCAccount, err := e.getSystemAccount()
	if err != nil {
		return nil, err
	}

	val, _, _ := systemSCAccount.AccountDataHandler().RetrieveValue(dctTokenKey)
	dctMetaData := DCTGlobalMetadataFromBytes(val)
	return &dctMetaData, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *dctGlobalSettings) IsInterfaceNil() bool {
	return e == nil
}
