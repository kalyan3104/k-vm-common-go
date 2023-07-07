package builtInFunctions

import (
	"math/big"
	"testing"

	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/check"
	"github.com/kalyan3104/k-core/data/dct"
	"github.com/kalyan3104/k-vm-common-go/mock"
	vmcommon "github.com/kalyan3104/k-vm-common-go"
	"github.com/stretchr/testify/assert"
)

func TestNewDCTBurnFunc(t *testing.T) {
	t.Parallel()

	t.Run("nil marshaller should error", func(t *testing.T) {
		t.Parallel()

		burnFunc, err := NewDCTBurnFunc(10, nil, &mock.GlobalSettingsHandlerStub{}, &mock.EnableEpochsHandlerStub{
			IsGlobalMintBurnFlagEnabledField: true,
		})
		assert.Equal(t, ErrNilMarshalizer, err)
		assert.True(t, check.IfNil(burnFunc))
	})
	t.Run("nil enable epochs handler should error", func(t *testing.T) {
		t.Parallel()

		burnFunc, err := NewDCTBurnFunc(10, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, nil)
		assert.Equal(t, ErrNilEnableEpochsHandler, err)
		assert.True(t, check.IfNil(burnFunc))
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		burnFunc, err := NewDCTBurnFunc(10, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, &mock.EnableEpochsHandlerStub{
			IsGlobalMintBurnFlagEnabledField: true,
		})
		assert.Nil(t, err)
		assert.False(t, check.IfNil(burnFunc))
	})
}

func TestDCTBurn_ProcessBuiltInFunctionErrors(t *testing.T) {
	t.Parallel()

	globalSettingsHandler := &mock.GlobalSettingsHandlerStub{}
	burnFunc, _ := NewDCTBurnFunc(10, &mock.MarshalizerMock{}, globalSettingsHandler, &mock.EnableEpochsHandlerStub{
		IsGlobalMintBurnFlagEnabledField: true,
	})
	_, err := burnFunc.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	_, err = burnFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
	}
	key := []byte("key")
	value := []byte("value")
	input.Arguments = [][]byte{key, value}
	_, err = burnFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrAddressIsNotDCTSystemSC)

	input.RecipientAddr = core.DCTSCAddress
	input.GasProvided = burnFunc.funcGasCost - 1
	accSnd := mock.NewUserAccount([]byte("dst"))
	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrNotEnoughGas)

	_, err = burnFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrNilUserAccount)

	globalSettingsHandler.IsPausedCalled = func(token []byte) bool {
		return true
	}
	input.GasProvided = burnFunc.funcGasCost
	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrDCTTokenIsPaused)
}

func TestDCTBurn_ProcessBuiltInFunctionSenderBurns(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	globalSettingsHandler := &mock.GlobalSettingsHandlerStub{}
	burnFunc, _ := NewDCTBurnFunc(10, marshaller, globalSettingsHandler, &mock.EnableEpochsHandlerStub{
		IsGlobalMintBurnFlagEnabledField: true,
	})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
		RecipientAddr: core.DCTSCAddress,
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accSnd := mock.NewUserAccount([]byte("snd"))

	dctFrozen := DCTUserMetadata{Frozen: true}
	dctNotFrozen := DCTUserMetadata{Frozen: false}

	dctKey := append(burnFunc.keyPrefix, key...)
	dctToken := &dct.DCToken{Value: big.NewInt(100), Properties: dctFrozen.ToBytes()}
	marshaledData, _ := marshaller.Marshal(dctToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(dctKey, marshaledData)

	_, err := burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrDCTIsFrozenForAccount)

	globalSettingsHandler.IsPausedCalled = func(token []byte) bool {
		return true
	}
	dctToken = &dct.DCToken{Value: big.NewInt(100), Properties: dctNotFrozen.ToBytes()}
	marshaledData, _ = marshaller.Marshal(dctToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(dctKey, marshaledData)

	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrDCTTokenIsPaused)

	globalSettingsHandler.IsPausedCalled = func(token []byte) bool {
		return false
	}
	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Nil(t, err)

	marshaledData, _, _ = accSnd.AccountDataHandler().RetrieveValue(dctKey)
	_ = marshaller.Unmarshal(dctToken, marshaledData)
	assert.True(t, dctToken.Value.Cmp(big.NewInt(90)) == 0)

	value = big.NewInt(100).Bytes()
	input.Arguments = [][]byte{key, value}
	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrInsufficientFunds)

	value = big.NewInt(90).Bytes()
	input.Arguments = [][]byte{key, value}
	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Nil(t, err)

	marshaledData, _, _ = accSnd.AccountDataHandler().RetrieveValue(dctKey)
	assert.Equal(t, len(marshaledData), 0)
}
