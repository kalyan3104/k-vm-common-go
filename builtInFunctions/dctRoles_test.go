package builtInFunctions

import (
	"bytes"
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/data/dct"
	vmcommon "github.com/kalyan3104/k-vm-common-go"
	"github.com/kalyan3104/k-vm-common-go/mock"
	"github.com/stretchr/testify/require"
)

func TestNewDCTRolesFunc_NilMarshalizerShouldErr(t *testing.T) {
	t.Parallel()

	dctRolesF, err := NewDCTRolesFunc(nil, false)

	require.Equal(t, ErrNilMarshalizer, err)
	require.Nil(t, dctRolesF)
}

func TestDctRoles_ProcessBuiltinFunction_NilVMInputShouldErr(t *testing.T) {
	t.Parallel()

	dctRolesF, _ := NewDCTRolesFunc(nil, false)

	_, err := dctRolesF.ProcessBuiltinFunction(nil, &mock.UserAccountStub{}, nil)
	require.Equal(t, ErrNilVmInput, err)
}

func TestDctRoles_ProcessBuiltinFunction_WrongCalledShouldErr(t *testing.T) {
	t.Parallel()

	dctRolesF, _ := NewDCTRolesFunc(nil, false)

	_, err := dctRolesF.ProcessBuiltinFunction(nil, &mock.UserAccountStub{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: []byte{},
			Arguments:  [][]byte{[]byte("1"), []byte("2")},
		},
	})
	require.Equal(t, ErrAddressIsNotDCTSystemSC, err)
}

func TestDctRoles_ProcessBuiltinFunction_NilAccountDestShouldErr(t *testing.T) {
	t.Parallel()

	dctRolesF, _ := NewDCTRolesFunc(nil, false)

	_, err := dctRolesF.ProcessBuiltinFunction(nil, nil, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.DCTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte("2")},
		},
	})
	require.Equal(t, ErrNilUserAccount, err)
}

func TestDctRoles_ProcessBuiltinFunction_GetRolesFailShouldErr(t *testing.T) {
	t.Parallel()

	dctRolesF, _ := NewDCTRolesFunc(&mock.MarshalizerMock{Fail: true}, false)

	_, err := dctRolesF.ProcessBuiltinFunction(nil, &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					return nil, 0, nil
				},
			}
		},
	}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.DCTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte("2")},
		},
	})
	require.Error(t, err)
}

func TestDctRoles_ProcessBuiltinFunction_GetRolesFailShouldWorkEvenIfAccntTrieIsNil(t *testing.T) {
	t.Parallel()

	saveKeyWasCalled := false
	dctRolesF, _ := NewDCTRolesFunc(&mock.MarshalizerMock{}, false)

	_, err := dctRolesF.ProcessBuiltinFunction(nil, &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					return nil, 0, nil
				},
				SaveKeyValueCalled: func(_ []byte, _ []byte) error {
					saveKeyWasCalled = true
					return nil
				},
			}
		},
	}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.DCTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte("2")},
		},
	})
	require.NoError(t, err)
	require.True(t, saveKeyWasCalled)
}

func TestDctRoles_ProcessBuiltinFunction_SetRolesShouldWork(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	dctRolesF, _ := NewDCTRolesFunc(marshaller, true)

	acc := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					roles := &dct.DCTRoles{}
					serializedRoles, err := marshaller.Marshal(roles)
					return serializedRoles, 0, err
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					roles := &dct.DCTRoles{}
					_ = marshaller.Unmarshal(roles, value)
					require.Equal(t, roles.Roles, [][]byte{[]byte(core.DCTRoleLocalMint)})
					return nil
				},
			}
		},
	}
	_, err := dctRolesF.ProcessBuiltinFunction(nil, acc, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.DCTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte(core.DCTRoleLocalMint)},
		},
	})
	require.Nil(t, err)
}

func TestDctRoles_ProcessBuiltinFunction_SetRolesMultiNFT(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	dctRolesF, _ := NewDCTRolesFunc(marshaller, true)

	tokenID := []byte("tokenID")
	roleKey := append(roleKeyPrefix, tokenID...)

	saveNonceCalled := false
	acc := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					roles := &dct.DCTRoles{}
					serializedRoles, err := marshaller.Marshal(roles)
					return serializedRoles, 0, err
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					if bytes.Equal(key, roleKey) {
						roles := &dct.DCTRoles{}
						_ = marshaller.Unmarshal(roles, value)
						require.Equal(t, roles.Roles, [][]byte{[]byte(core.DCTRoleNFTCreate), []byte(core.DCTRoleNFTCreateMultiShard)})
						return nil
					}

					if bytes.Equal(key, getNonceKey(tokenID)) {
						saveNonceCalled = true
						require.Equal(t, uint64(math.MaxUint64/256), big.NewInt(0).SetBytes(value).Uint64())
					}

					return nil
				},
			}
		},
	}
	dstAddr := bytes.Repeat([]byte{1}, 32)
	_, err := dctRolesF.ProcessBuiltinFunction(nil, acc, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.DCTSCAddress,
			Arguments:  [][]byte{tokenID, []byte(core.DCTRoleNFTCreate), []byte(core.DCTRoleNFTCreateMultiShard)},
		},
		RecipientAddr: dstAddr,
	})

	require.Nil(t, err)
	require.True(t, saveNonceCalled)
}

func TestDctRoles_ProcessBuiltinFunction_SaveFailedShouldErr(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	dctRolesF, _ := NewDCTRolesFunc(marshaller, true)

	localErr := errors.New("local err")
	acc := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					roles := &dct.DCTRoles{}
					serializedRoles, err := marshaller.Marshal(roles)
					return serializedRoles, 0, err
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					return localErr
				},
			}
		},
	}
	_, err := dctRolesF.ProcessBuiltinFunction(nil, acc, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.DCTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte(core.DCTRoleLocalMint)},
		},
	})
	require.Equal(t, localErr, err)
}

func TestDctRoles_ProcessBuiltinFunction_UnsetRolesDoesNotExistsShouldWork(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	dctRolesF, _ := NewDCTRolesFunc(marshaller, false)

	acc := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					roles := &dct.DCTRoles{}
					serializedRoles, err := marshaller.Marshal(roles)
					return serializedRoles, 0, err
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					roles := &dct.DCTRoles{}
					_ = marshaller.Unmarshal(roles, value)
					require.Len(t, roles.Roles, 0)
					return nil
				},
			}
		},
	}
	_, err := dctRolesF.ProcessBuiltinFunction(nil, acc, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.DCTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte(core.DCTRoleLocalMint)},
		},
	})
	require.Nil(t, err)
}

func TestDctRoles_ProcessBuiltinFunction_UnsetRolesShouldWork(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	dctRolesF, _ := NewDCTRolesFunc(marshaller, false)

	acc := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					roles := &dct.DCTRoles{
						Roles: [][]byte{[]byte(core.DCTRoleLocalMint)},
					}
					serializedRoles, err := marshaller.Marshal(roles)
					return serializedRoles, 0, err
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					roles := &dct.DCTRoles{}
					_ = marshaller.Unmarshal(roles, value)
					require.Len(t, roles.Roles, 0)
					return nil
				},
			}
		},
	}
	_, err := dctRolesF.ProcessBuiltinFunction(nil, acc, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: core.DCTSCAddress,
			Arguments:  [][]byte{[]byte("1"), []byte(core.DCTRoleLocalMint)},
		},
	})
	require.Nil(t, err)
}

func TestDctRoles_CheckAllowedToExecuteNilAccountShouldErr(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	dctRolesF, _ := NewDCTRolesFunc(marshaller, false)

	err := dctRolesF.CheckAllowedToExecute(nil, []byte("ID"), []byte(core.DCTRoleLocalBurn))
	require.Equal(t, ErrNilUserAccount, err)
}

func TestDctRoles_CheckAllowedToExecuteCannotGetDCTRole(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{Fail: true}
	dctRolesF, _ := NewDCTRolesFunc(marshaller, false)

	err := dctRolesF.CheckAllowedToExecute(&mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					return nil, 0, nil
				},
			}
		},
	}, []byte("ID"), []byte(core.DCTRoleLocalBurn))
	require.Error(t, err)
}

func TestDctRoles_CheckAllowedToExecuteIsNewNotAllowed(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	dctRolesF, _ := NewDCTRolesFunc(marshaller, false)

	err := dctRolesF.CheckAllowedToExecute(&mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					return nil, 0, nil
				},
			}
		},
	}, []byte("ID"), []byte(core.DCTRoleLocalBurn))
	require.Equal(t, ErrActionNotAllowed, err)
}

func TestDctRoles_CheckAllowed_ShouldWork(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	dctRolesF, _ := NewDCTRolesFunc(marshaller, false)

	err := dctRolesF.CheckAllowedToExecute(&mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					roles := &dct.DCTRoles{
						Roles: [][]byte{[]byte(core.DCTRoleLocalMint)},
					}
					serializedRoles, err := marshaller.Marshal(roles)
					return serializedRoles, 0, err
				},
			}
		},
	}, []byte("ID"), []byte(core.DCTRoleLocalMint))
	require.Nil(t, err)
}

func TestDctRoles_CheckAllowedToExecuteRoleNotFind(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	dctRolesF, _ := NewDCTRolesFunc(marshaller, false)

	err := dctRolesF.CheckAllowedToExecute(&mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					roles := &dct.DCTRoles{
						Roles: [][]byte{[]byte(core.DCTRoleLocalBurn)},
					}
					serializedRoles, err := marshaller.Marshal(roles)
					return serializedRoles, 0, err
				},
			}
		},
	}, []byte("ID"), []byte(core.DCTRoleLocalMint))
	require.Equal(t, ErrActionNotAllowed, err)
}
