package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"math/big"

	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/check"
	"github.com/kalyan3104/k-core/data/dct"
	vmcommon "github.com/kalyan3104/k-vm-common-go"
)

type dctNFTCreateRoleTransfer struct {
	baseAlwaysActiveHandler
	keyPrefix        []byte
	marshaller       vmcommon.Marshalizer
	accounts         vmcommon.AccountsAdapter
	shardCoordinator vmcommon.Coordinator
}

// NewDCTNFTCreateRoleTransfer returns the dct NFT create role transfer built-in function component
func NewDCTNFTCreateRoleTransfer(
	marshaller vmcommon.Marshalizer,
	accounts vmcommon.AccountsAdapter,
	shardCoordinator vmcommon.Coordinator,
) (*dctNFTCreateRoleTransfer, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(shardCoordinator) {
		return nil, ErrNilShardCoordinator
	}

	e := &dctNFTCreateRoleTransfer{
		keyPrefix:        []byte(baseDCTKeyPrefix),
		marshaller:       marshaller,
		accounts:         accounts,
		shardCoordinator: shardCoordinator,
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dctNFTCreateRoleTransfer) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves DCT create role transfer function call
func (e *dctNFTCreateRoleTransfer) ProcessBuiltinFunction(
	acntSnd, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {

	err := checkBasicDCTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	if !check.IfNil(acntSnd) {
		return nil, ErrInvalidArguments
	}
	if check.IfNil(acntDst) {
		return nil, ErrNilUserAccount
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
	if bytes.Equal(vmInput.CallerAddr, core.DCTSCAddress) {
		outAcc, errExec := e.executeTransferNFTCreateChangeAtCurrentOwner(vmOutput, acntDst, vmInput)
		if errExec != nil {
			return nil, errExec
		}
		vmOutput.OutputAccounts = make(map[string]*vmcommon.OutputAccount)
		vmOutput.OutputAccounts[string(outAcc.Address)] = outAcc
	} else {
		err = e.executeTransferNFTCreateChangeAtNextOwner(vmOutput, acntDst, vmInput)
		if err != nil {
			return nil, err
		}
	}

	return vmOutput, nil
}

func (e *dctNFTCreateRoleTransfer) executeTransferNFTCreateChangeAtCurrentOwner(
	vmOutput *vmcommon.VMOutput,
	acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.OutputAccount, error) {
	if len(vmInput.Arguments) != 2 {
		return nil, ErrInvalidArguments
	}
	if len(vmInput.Arguments[1]) != len(vmInput.CallerAddr) {
		return nil, ErrInvalidArguments
	}

	tokenID := vmInput.Arguments[0]
	nonce, err := getLatestNonce(acntDst, tokenID)
	if err != nil {
		return nil, err
	}

	err = saveLatestNonce(acntDst, tokenID, 0)
	if err != nil {
		return nil, err
	}

	dctTokenRoleKey := append(roleKeyPrefix, tokenID...)
	err = e.deleteCreateRoleFromAccount(acntDst, dctTokenRoleKey)
	if err != nil {
		return nil, err
	}

	logData := [][]byte{acntDst.AddressBytes(), boolToSlice(false)}
	addDCTEntryInVMOutput(vmOutput, []byte(vmInput.Function), tokenID, 0, big.NewInt(0), logData...)

	destAddress := vmInput.Arguments[1]
	if e.shardCoordinator.ComputeId(destAddress) == e.shardCoordinator.SelfId() {
		newDestinationAcc, errLoad := e.accounts.LoadAccount(destAddress)
		if errLoad != nil {
			return nil, errLoad
		}
		newDestUserAcc, ok := newDestinationAcc.(vmcommon.UserAccountHandler)
		if !ok {
			return nil, ErrWrongTypeAssertion
		}

		err = saveLatestNonce(newDestUserAcc, tokenID, nonce)
		if err != nil {
			return nil, err
		}

		err = e.addCreateRoleToAccount(newDestUserAcc, dctTokenRoleKey)
		if err != nil {
			return nil, err
		}

		err = e.accounts.SaveAccount(newDestUserAcc)
		if err != nil {
			return nil, err
		}

		logData = [][]byte{destAddress, boolToSlice(true)}
		addDCTEntryInVMOutput(vmOutput, []byte(vmInput.Function), tokenID, 0, big.NewInt(0), logData...)
	}

	outAcc := &vmcommon.OutputAccount{
		Address:         destAddress,
		Balance:         big.NewInt(0),
		BalanceDelta:    big.NewInt(0),
		OutputTransfers: make([]vmcommon.OutputTransfer, 0),
	}
	outTransfer := vmcommon.OutputTransfer{
		Value: big.NewInt(0),
		Data: []byte(core.BuiltInFunctionDCTNFTCreateRoleTransfer + "@" +
			hex.EncodeToString(tokenID) + "@" + hex.EncodeToString(big.NewInt(0).SetUint64(nonce).Bytes())),
		SenderAddress: vmInput.CallerAddr,
	}
	outAcc.OutputTransfers = append(outAcc.OutputTransfers, outTransfer)

	return outAcc, nil
}

func (e *dctNFTCreateRoleTransfer) deleteCreateRoleFromAccount(
	acntDst vmcommon.UserAccountHandler,
	dctTokenRoleKey []byte,
) error {
	roles, _, err := getDCTRolesForAcnt(e.marshaller, acntDst, dctTokenRoleKey)
	if err != nil {
		return err
	}

	deleteRoles(roles, [][]byte{[]byte(core.DCTRoleNFTCreate)})
	return saveRolesToAccount(acntDst, dctTokenRoleKey, roles, e.marshaller)
}

func (e *dctNFTCreateRoleTransfer) addCreateRoleToAccount(
	acntDst vmcommon.UserAccountHandler,
	dctTokenRoleKey []byte,
) error {
	roles, _, err := getDCTRolesForAcnt(e.marshaller, acntDst, dctTokenRoleKey)
	if err != nil {
		return err
	}

	for _, role := range roles.Roles {
		if bytes.Equal(role, []byte(core.DCTRoleNFTCreate)) {
			return nil
		}
	}

	roles.Roles = append(roles.Roles, []byte(core.DCTRoleNFTCreate))
	return saveRolesToAccount(acntDst, dctTokenRoleKey, roles, e.marshaller)
}

func saveRolesToAccount(
	acntDst vmcommon.UserAccountHandler,
	dctTokenRoleKey []byte,
	roles *dct.DCTRoles,
	marshaller vmcommon.Marshalizer,
) error {
	marshaledData, err := marshaller.Marshal(roles)
	if err != nil {
		return err
	}
	err = acntDst.AccountDataHandler().SaveKeyValue(dctTokenRoleKey, marshaledData)
	if err != nil {
		return err
	}

	return nil
}

func (e *dctNFTCreateRoleTransfer) executeTransferNFTCreateChangeAtNextOwner(
	vmOutput *vmcommon.VMOutput,
	acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) error {
	if len(vmInput.Arguments) != 2 {
		return ErrInvalidArguments
	}

	tokenID := vmInput.Arguments[0]
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()

	err := saveLatestNonce(acntDst, tokenID, nonce)
	if err != nil {
		return err
	}

	dctTokenRoleKey := append(roleKeyPrefix, tokenID...)
	err = e.addCreateRoleToAccount(acntDst, dctTokenRoleKey)
	if err != nil {
		return err
	}

	logData := [][]byte{acntDst.AddressBytes(), boolToSlice(true)}
	addDCTEntryInVMOutput(vmOutput, []byte(vmInput.Function), tokenID, 0, big.NewInt(0), logData...)

	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *dctNFTCreateRoleTransfer) IsInterfaceNil() bool {
	return e == nil
}
