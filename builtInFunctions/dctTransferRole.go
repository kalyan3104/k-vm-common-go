package builtInFunctions

import (
	"bytes"
	"math/big"

	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/check"
	"github.com/kalyan3104/k-core/data/dct"
	"github.com/kalyan3104/k-core/marshal"
	vmcommon "github.com/kalyan3104/k-vm-common-go"
)

const transfer = "transfer"

var transferAddressesKeyPrefix = []byte(core.ProtectedKeyPrefix + transfer + core.DCTKeyIdentifier)

type dctTransferAddress struct {
	baseActiveHandler
	set             bool
	marshaller      vmcommon.Marshalizer
	accounts        vmcommon.AccountsAdapter
	maxNumAddresses uint32
}

// NewDCTTransferRoleAddressFunc returns the dct transfer role address handler built-in function component
func NewDCTTransferRoleAddressFunc(
	accounts vmcommon.AccountsAdapter,
	marshaller marshal.Marshalizer,
	maxNumAddresses uint32,
	set bool,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*dctTransferAddress, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if maxNumAddresses < 1 {
		return nil, ErrInvalidMaxNumAddresses
	}
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}

	e := &dctTransferAddress{
		accounts:        accounts,
		marshaller:      marshaller,
		maxNumAddresses: maxNumAddresses,
		set:             set,
	}

	e.baseActiveHandler.activeHandler = enableEpochsHandler.IsSendAlwaysFlagEnabled

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dctTransferAddress) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves DCT change roles function call
func (e *dctTransferAddress) ProcessBuiltinFunction(
	_, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	err := checkBasicDCTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(vmInput.CallerAddr, core.DCTSCAddress) {
		return nil, ErrAddressIsNotDCTSystemSC
	}
	if !vmcommon.IsSystemAccountAddress(vmInput.RecipientAddr) {
		return nil, ErrOnlySystemAccountAccepted
	}

	systemAcc, err := e.getSystemAccount()
	if err != nil {
		return nil, err
	}

	dctTokenTransferRoleKey := append(transferAddressesKeyPrefix, vmInput.Arguments[0]...)
	addresses, _, err := getDCTRolesForAcnt(e.marshaller, systemAcc, dctTokenTransferRoleKey)
	if err != nil {
		return nil, err
	}

	if e.set {
		err = e.addNewAddresses(vmInput, addresses)
		if err != nil {
			return nil, err
		}
	} else {
		deleteRoles(addresses, vmInput.Arguments[1:])
	}

	err = saveRolesToAccount(systemAcc, dctTokenTransferRoleKey, addresses, e.marshaller)
	if err != nil {
		return nil, err
	}

	err = e.accounts.SaveAccount(systemAcc)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}

	logData := append([][]byte{systemAcc.AddressBytes()}, vmInput.Arguments[1:]...)
	addDCTEntryInVMOutput(vmOutput, []byte(vmInput.Function), vmInput.Arguments[0], 0, big.NewInt(0), logData...)

	return vmOutput, nil
}

func (e *dctTransferAddress) addNewAddresses(vmInput *vmcommon.ContractCallInput, addresses *dct.DCTRoles) error {
	for _, newAddress := range vmInput.Arguments[1:] {
		isNew := true
		for _, address := range addresses.Roles {
			if bytes.Equal(newAddress, address) {
				isNew = false
				break
			}
		}
		if isNew {
			addresses.Roles = append(addresses.Roles, newAddress)
		}
	}

	if uint32(len(addresses.Roles)) > e.maxNumAddresses {
		return ErrTooManyTransferAddresses
	}

	return nil
}

func (e *dctTransferAddress) getSystemAccount() (vmcommon.UserAccountHandler, error) {
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

// IsInterfaceNil returns true if underlying object in nil
func (e *dctTransferAddress) IsInterfaceNil() bool {
	return e == nil
}
