package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/data/dct"
	"github.com/kalyan3104/k-core/data/smartContractResult"
	vmcommon "github.com/kalyan3104/k-vm-common-go"
	"github.com/kalyan3104/k-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
)

func createNewDCTDataStorageHandler() *dctDataStorage {
	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts := &mock.AccountsStub{LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
		return acnt, nil
	}}
	args := ArgsNewDCTDataStorage{
		Accounts:              accounts,
		GlobalSettingsHandler: &mock.GlobalSettingsHandlerStub{},
		Marshalizer:           &mock.MarshalizerMock{},
		EnableEpochsHandler: &mock.EnableEpochsHandlerStub{
			IsSaveToSystemAccountFlagEnabledField: true,
			IsSendAlwaysFlagEnabledField:          true,
		},
		ShardCoordinator: &mock.ShardCoordinatorStub{},
	}
	dataStore, _ := NewDCTDataStorage(args)
	return dataStore
}

func createMockArgsForNewDCTDataStorage() ArgsNewDCTDataStorage {
	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	accounts := &mock.AccountsStub{LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
		return acnt, nil
	}}
	args := ArgsNewDCTDataStorage{
		Accounts:              accounts,
		GlobalSettingsHandler: &mock.GlobalSettingsHandlerStub{},
		Marshalizer:           &mock.MarshalizerMock{},
		EnableEpochsHandler: &mock.EnableEpochsHandlerStub{
			IsSaveToSystemAccountFlagEnabledField: true,
			IsSendAlwaysFlagEnabledField:          true,
		},
		ShardCoordinator: &mock.ShardCoordinatorStub{},
	}
	return args
}

func createNewDCTDataStorageHandlerWithArgs(
	globalSettingsHandler vmcommon.DCTGlobalSettingsHandler,
	accounts vmcommon.AccountsAdapter,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) *dctDataStorage {
	args := ArgsNewDCTDataStorage{
		Accounts:              accounts,
		GlobalSettingsHandler: globalSettingsHandler,
		Marshalizer:           &mock.MarshalizerMock{},
		EnableEpochsHandler:   enableEpochsHandler,
		ShardCoordinator:      &mock.ShardCoordinatorStub{},
	}
	dataStore, _ := NewDCTDataStorage(args)
	return dataStore
}

func TestNewDCTDataStorage(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewDCTDataStorage()
	args.Marshalizer = nil
	e, err := NewDCTDataStorage(args)
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilMarshalizer)

	args = createMockArgsForNewDCTDataStorage()
	args.Accounts = nil
	e, err = NewDCTDataStorage(args)
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilAccountsAdapter)

	args = createMockArgsForNewDCTDataStorage()
	args.ShardCoordinator = nil
	e, err = NewDCTDataStorage(args)
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilShardCoordinator)

	args = createMockArgsForNewDCTDataStorage()
	args.GlobalSettingsHandler = nil
	e, err = NewDCTDataStorage(args)
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilGlobalSettingsHandler)

	args = createMockArgsForNewDCTDataStorage()
	args.EnableEpochsHandler = nil
	e, err = NewDCTDataStorage(args)
	assert.Nil(t, e)
	assert.Equal(t, err, ErrNilEnableEpochsHandler)

	args = createMockArgsForNewDCTDataStorage()
	e, err = NewDCTDataStorage(args)
	assert.Nil(t, err)
	assert.False(t, e.IsInterfaceNil())
}

func TestDctDataStorage_GetDCTNFTTokenOnDestinationNoDataInSystemAcc(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewDCTDataStorage()
	e, _ := NewDCTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	dctData := &dct.DCToken{
		TokenMetaData: &dct.MetaData{
			Name: []byte("test"),
		},
		Value: big.NewInt(10),
	}

	tokenIdentifier := "testTkn"
	key := baseDCTKeyPrefix + tokenIdentifier
	nonce := uint64(10)
	dctDataBytes, _ := args.Marshalizer.Marshal(dctData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, dctDataBytes)

	dctDataGet, _, err := e.GetDCTNFTTokenOnDestination(userAcc, []byte(key), nonce)
	assert.Nil(t, err)
	assert.Equal(t, dctData, dctDataGet)
}

func TestDctDataStorage_GetDCTNFTTokenOnDestinationGetDataFromSystemAcc(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewDCTDataStorage()
	e, _ := NewDCTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	dctData := &dct.DCToken{
		Value: big.NewInt(10),
	}

	tokenIdentifier := "testTkn"
	key := baseDCTKeyPrefix + tokenIdentifier
	nonce := uint64(10)
	dctDataBytes, _ := args.Marshalizer.Marshal(dctData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, dctDataBytes)

	systemAcc, _ := e.getSystemAccount(defaultQueryOptions())
	metaData := &dct.MetaData{
		Name: []byte("test"),
	}
	dctDataOnSystemAcc := &dct.DCToken{TokenMetaData: metaData}
	dctMetaDataBytes, _ := args.Marshalizer.Marshal(dctDataOnSystemAcc)
	_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, dctMetaDataBytes)

	dctDataGet, _, err := e.GetDCTNFTTokenOnDestination(userAcc, []byte(key), nonce)
	assert.Nil(t, err)
	dctData.TokenMetaData = metaData
	assert.Equal(t, dctData, dctDataGet)
}

func TestDctDataStorage_GetDCTNFTTokenOnDestinationWithCustomSystemAccount(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewDCTDataStorage()
	e, _ := NewDCTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	dctData := &dct.DCToken{
		Value: big.NewInt(10),
	}

	tokenIdentifier := "testTkn"
	key := baseDCTKeyPrefix + tokenIdentifier
	nonce := uint64(10)
	dctDataBytes, _ := args.Marshalizer.Marshal(dctData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, dctDataBytes)

	systemAcc, _ := e.getSystemAccount(defaultQueryOptions())
	metaData := &dct.MetaData{
		Name: []byte("test"),
	}
	dctDataOnSystemAcc := &dct.DCToken{TokenMetaData: metaData}
	dctMetaDataBytes, _ := args.Marshalizer.Marshal(dctDataOnSystemAcc)
	_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, dctMetaDataBytes)

	retrieveValueFromCustomAccountCalled := false
	customSystemAccount := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(key []byte) ([]byte, uint32, error) {
					retrieveValueFromCustomAccountCalled = true
					return dctMetaDataBytes, 0, nil
				},
			}
		},
	}
	dctDataGet, _, err := e.GetDCTNFTTokenOnDestinationWithCustomSystemAccount(userAcc, []byte(key), nonce, customSystemAccount)
	assert.Nil(t, err)
	dctData.TokenMetaData = metaData
	assert.Equal(t, dctData, dctDataGet)
	assert.True(t, retrieveValueFromCustomAccountCalled)
}

func TestDctDataStorage_GetDCTNFTTokenOnDestinationMarshalERR(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewDCTDataStorage()
	e, _ := NewDCTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	dctData := &dct.DCToken{
		Value: big.NewInt(10),
		TokenMetaData: &dct.MetaData{
			Name: []byte("test"),
		},
	}

	tokenIdentifier := "testTkn"
	key := baseDCTKeyPrefix + tokenIdentifier
	nonce := uint64(10)
	dctDataBytes, _ := args.Marshalizer.Marshal(dctData)
	dctDataBytes = append(dctDataBytes, dctDataBytes...)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, dctDataBytes)

	_, _, err := e.GetDCTNFTTokenOnDestination(userAcc, []byte(key), nonce)
	assert.NotNil(t, err)

	_, err = e.GetDCTNFTTokenOnSender(userAcc, []byte(key), nonce)
	assert.NotNil(t, err)
}

func TestDctDataStorage_MarshalErrorOnSystemACC(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewDCTDataStorage()
	e, _ := NewDCTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	dctData := &dct.DCToken{
		Value: big.NewInt(10),
	}

	tokenIdentifier := "testTkn"
	key := baseDCTKeyPrefix + tokenIdentifier
	nonce := uint64(10)
	dctDataBytes, _ := args.Marshalizer.Marshal(dctData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, dctDataBytes)

	systemAcc, _ := e.getSystemAccount(defaultQueryOptions())
	metaData := &dct.MetaData{
		Name: []byte("test"),
	}
	dctDataOnSystemAcc := &dct.DCToken{TokenMetaData: metaData}
	dctMetaDataBytes, _ := args.Marshalizer.Marshal(dctDataOnSystemAcc)
	dctMetaDataBytes = append(dctMetaDataBytes, dctMetaDataBytes...)
	_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, dctMetaDataBytes)

	_, _, err := e.GetDCTNFTTokenOnDestination(userAcc, []byte(key), nonce)
	assert.NotNil(t, err)
}

func TestDCTDataStorage_saveDataToSystemAccNotNFTOrMetaData(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewDCTDataStorage()
	e, _ := NewDCTDataStorage(args)

	err := e.saveDCTMetaDataToSystemAccount(nil, 0, []byte("TCK"), 0, nil, true)
	assert.Nil(t, err)

	err = e.saveDCTMetaDataToSystemAccount(nil, 0, []byte("TCK"), 1, &dct.DCToken{}, true)
	assert.Nil(t, err)
}

func TestDctDataStorage_SaveDCTNFTTokenNoChangeInSystemAcc(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewDCTDataStorage()
	e, _ := NewDCTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	dctData := &dct.DCToken{
		Value: big.NewInt(10),
	}

	tokenIdentifier := "testTkn"
	key := baseDCTKeyPrefix + tokenIdentifier
	nonce := uint64(10)
	dctDataBytes, _ := args.Marshalizer.Marshal(dctData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, dctDataBytes)

	systemAcc, _ := e.getSystemAccount(defaultQueryOptions())
	metaData := &dct.MetaData{
		Name: []byte("test"),
	}
	dctDataOnSystemAcc := &dct.DCToken{TokenMetaData: metaData}
	dctMetaDataBytes, _ := args.Marshalizer.Marshal(dctDataOnSystemAcc)
	_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, dctMetaDataBytes)

	newMetaData := &dct.MetaData{Name: []byte("newName")}
	transferDCTData := &dct.DCToken{Value: big.NewInt(100), TokenMetaData: newMetaData}
	_, err := e.SaveDCTNFTToken([]byte("address"), userAcc, []byte(key), nonce, transferDCTData, false, false)
	assert.Nil(t, err)

	dctDataGet, _, err := e.GetDCTNFTTokenOnDestination(userAcc, []byte(key), nonce)
	assert.Nil(t, err)
	dctData.TokenMetaData = metaData
	dctData.Value = big.NewInt(100)
	assert.Equal(t, dctData, dctDataGet)
}

func TestDctDataStorage_SaveDCTNFTTokenAlwaysSaveTokenMetaDataEnabled(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewDCTDataStorage()
	args.EnableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsSaveToSystemAccountFlagEnabledField: true,
		IsSendAlwaysFlagEnabledField:          true,
		IsAlwaysSaveTokenMetaDataEnabledField: true,
	}
	dataStorage, _ := NewDCTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	nonce := uint64(10)

	t.Run("new token should not rewrite metadata", func(t *testing.T) {
		newToken := &dct.DCToken{
			Value: big.NewInt(10),
		}
		tokenIdentifier := "newTkn"
		key := baseDCTKeyPrefix + tokenIdentifier
		tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)

		_ = saveDCTData(userAcc, newToken, tokenKey, args.Marshalizer)

		systemAcc, _ := dataStorage.getSystemAccount(defaultQueryOptions())
		metaData := &dct.MetaData{
			Name: []byte("test"),
		}
		dctDataOnSystemAcc := &dct.DCToken{
			TokenMetaData: metaData,
			Reserved:      []byte{1},
		}
		dctMetaDataBytes, _ := args.Marshalizer.Marshal(dctDataOnSystemAcc)
		_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, dctMetaDataBytes)

		newMetaData := &dct.MetaData{Name: []byte("newName")}
		transferDCTData := &dct.DCToken{Value: big.NewInt(100), TokenMetaData: newMetaData}
		_, err := dataStorage.SaveDCTNFTToken([]byte("address"), userAcc, []byte(key), nonce, transferDCTData, false, false)
		assert.Nil(t, err)

		dctDataGet, _, err := dataStorage.GetDCTNFTTokenOnDestination(userAcc, []byte(key), nonce)
		assert.Nil(t, err)

		expectedDCTData := &dct.DCToken{
			Value:         big.NewInt(100),
			TokenMetaData: metaData,
		}
		assert.Equal(t, expectedDCTData, dctDataGet)
	})
	t.Run("old token should rewrite metadata", func(t *testing.T) {
		newToken := &dct.DCToken{
			Value: big.NewInt(10),
		}
		tokenIdentifier := "newTkn"
		key := baseDCTKeyPrefix + tokenIdentifier
		tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)

		_ = saveDCTData(userAcc, newToken, tokenKey, args.Marshalizer)

		systemAcc, _ := dataStorage.getSystemAccount(defaultQueryOptions())
		metaData := &dct.MetaData{
			Name: []byte("test"),
		}
		dctDataOnSystemAcc := &dct.DCToken{
			TokenMetaData: metaData,
		}
		dctMetaDataBytes, _ := args.Marshalizer.Marshal(dctDataOnSystemAcc)
		_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, dctMetaDataBytes)

		newMetaData := &dct.MetaData{Name: []byte("newName")}
		transferDCTData := &dct.DCToken{Value: big.NewInt(100), TokenMetaData: newMetaData}
		dctDataGet := setAndGetStoredToken(t, dataStorage, userAcc, []byte(key), nonce, transferDCTData)

		expectedDCTData := &dct.DCToken{
			Value:         big.NewInt(100),
			TokenMetaData: newMetaData,
		}
		assert.Equal(t, expectedDCTData, dctDataGet)
	})
	t.Run("old token should not rewrite metadata if the flags are not set", func(t *testing.T) {
		localArgs := createMockArgsForNewDCTDataStorage()
		localEpochsHandler := &mock.EnableEpochsHandlerStub{
			IsSaveToSystemAccountFlagEnabledField: true,
			IsSendAlwaysFlagEnabledField:          true,
			IsAlwaysSaveTokenMetaDataEnabledField: true,
		}
		localArgs.EnableEpochsHandler = localEpochsHandler
		localDataStorage, _ := NewDCTDataStorage(localArgs)

		newToken := &dct.DCToken{
			Value: big.NewInt(10),
		}
		tokenIdentifier := "newTkn"
		key := baseDCTKeyPrefix + tokenIdentifier
		tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)

		_ = saveDCTData(userAcc, newToken, tokenKey, localArgs.Marshalizer)

		systemAcc, _ := localDataStorage.getSystemAccount(defaultQueryOptions())
		metaData := &dct.MetaData{
			Name: []byte("test"),
		}
		dctDataOnSystemAcc := &dct.DCToken{
			TokenMetaData: metaData,
		}
		dctMetaDataBytes, _ := localArgs.Marshalizer.Marshal(dctDataOnSystemAcc)
		_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, dctMetaDataBytes)

		newMetaData := &dct.MetaData{Name: []byte("newName")}
		transferDCTData := &dct.DCToken{Value: big.NewInt(100), TokenMetaData: newMetaData}
		expectedDCTData := &dct.DCToken{
			Value:         big.NewInt(100),
			TokenMetaData: metaData,
		}

		localEpochsHandler.IsAlwaysSaveTokenMetaDataEnabledField = false
		localEpochsHandler.IsSendAlwaysFlagEnabledField = true

		dctDataGet := setAndGetStoredToken(t, localDataStorage, userAcc, []byte(key), nonce, transferDCTData)
		assert.Equal(t, expectedDCTData, dctDataGet)

		localEpochsHandler.IsAlwaysSaveTokenMetaDataEnabledField = true
		localEpochsHandler.IsSendAlwaysFlagEnabledField = false

		dctDataGet = setAndGetStoredToken(t, localDataStorage, userAcc, []byte(key), nonce, transferDCTData)
		assert.Equal(t, expectedDCTData, dctDataGet)
	})
}

func setAndGetStoredToken(
	tb testing.TB,
	dctDataStorage *dctDataStorage,
	userAcc vmcommon.UserAccountHandler,
	key []byte,
	nonce uint64,
	transferDCTData *dct.DCToken,
) *dct.DCToken {
	_, err := dctDataStorage.SaveDCTNFTToken([]byte("address"), userAcc, key, nonce, transferDCTData, false, false)
	assert.Nil(tb, err)

	dctDataGet, _, err := dctDataStorage.GetDCTNFTTokenOnDestination(userAcc, key, nonce)
	assert.Nil(tb, err)

	return dctDataGet
}

func TestDctDataStorage_SaveDCTNFTTokenWhenQuantityZero(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewDCTDataStorage()
	e, _ := NewDCTDataStorage(args)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	nonce := uint64(10)
	dctData := &dct.DCToken{
		Value: big.NewInt(10),
		TokenMetaData: &dct.MetaData{
			Name:  []byte("test"),
			Nonce: nonce,
		},
	}

	tokenIdentifier := "testTkn"
	key := baseDCTKeyPrefix + tokenIdentifier
	dctDataBytes, _ := args.Marshalizer.Marshal(dctData)
	tokenKey := append([]byte(key), big.NewInt(int64(nonce)).Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, dctDataBytes)

	dctData.Value = big.NewInt(0)
	_, err := e.SaveDCTNFTToken([]byte("address"), userAcc, []byte(key), nonce, dctData, false, false)
	assert.Nil(t, err)

	val, _, err := userAcc.AccountDataHandler().RetrieveValue(tokenKey)
	assert.Nil(t, val)
	assert.Nil(t, err)

	dctMetaData, err := e.getDCTMetaDataFromSystemAccount(tokenKey, defaultQueryOptions())
	assert.Nil(t, err)
	assert.Equal(t, dctData.TokenMetaData, dctMetaData)
}

func TestDctDataStorage_WasAlreadySentToDestinationShard(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewDCTDataStorage()
	shardCoordinator := &mock.ShardCoordinatorStub{}
	args.ShardCoordinator = shardCoordinator
	e, _ := NewDCTDataStorage(args)

	tickerID := []byte("ticker")
	dstAddress := []byte("dstAddress")
	val, err := e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 0, dstAddress)
	assert.True(t, val)
	assert.Nil(t, err)

	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.True(t, val)
	assert.Nil(t, err)

	enableEpochsHandler, _ := args.EnableEpochsHandler.(*mock.EnableEpochsHandlerStub)
	enableEpochsHandler.IsSendAlwaysFlagEnabledField = false
	shardCoordinator.ComputeIdCalled = func(_ []byte) uint32 {
		return core.MetachainShardId
	}
	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.True(t, val)
	assert.Nil(t, err)

	enableEpochsHandler.IsSendAlwaysFlagEnabledField = true

	shardCoordinator.ComputeIdCalled = func(_ []byte) uint32 {
		return 1
	}
	shardCoordinator.NumberOfShardsCalled = func() uint32 {
		return 5
	}
	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.False(t, val)
	assert.Nil(t, err)

	systemAcc, _ := e.getSystemAccount(defaultQueryOptions())
	metaData := &dct.MetaData{
		Name: []byte("test"),
	}
	dctDataOnSystemAcc := &dct.DCToken{TokenMetaData: metaData}
	dctMetaDataBytes, _ := args.Marshalizer.Marshal(dctDataOnSystemAcc)
	key := baseDCTKeyPrefix + string(tickerID)
	tokenKey := append([]byte(key), big.NewInt(1).Bytes()...)
	_ = systemAcc.AccountDataHandler().SaveKeyValue(tokenKey, dctMetaDataBytes)

	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.False(t, val)
	assert.Nil(t, err)

	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.False(t, val)
	assert.Nil(t, err)

	enableEpochsHandler.IsSendAlwaysFlagEnabledField = false
	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.False(t, val)
	assert.Nil(t, err)

	shardCoordinator.NumberOfShardsCalled = func() uint32 {
		return 10
	}
	val, err = e.WasAlreadySentToDestinationShardAndUpdateState(tickerID, 1, dstAddress)
	assert.True(t, val)
	assert.Nil(t, err)
}

func TestDctDataStorage_SaveNFTMetaDataToSystemAccount(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewDCTDataStorage()
	shardCoordinator := &mock.ShardCoordinatorStub{}
	args.ShardCoordinator = shardCoordinator
	e, _ := NewDCTDataStorage(args)

	enableEpochsHandler, _ := args.EnableEpochsHandler.(*mock.EnableEpochsHandlerStub)
	enableEpochsHandler.IsSaveToSystemAccountFlagEnabledField = false
	err := e.SaveNFTMetaDataToSystemAccount(nil)
	assert.Nil(t, err)

	enableEpochsHandler.IsSaveToSystemAccountFlagEnabledField = true
	err = e.SaveNFTMetaDataToSystemAccount(nil)
	assert.Nil(t, err)

	enableEpochsHandler.IsSendAlwaysFlagEnabledField = false
	err = e.SaveNFTMetaDataToSystemAccount(nil)
	assert.Equal(t, err, ErrNilTransactionHandler)

	scr := &smartContractResult.SmartContractResult{
		SndAddr: []byte("address1"),
		RcvAddr: []byte("address2"),
	}

	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.Nil(t, err)

	shardCoordinator.ComputeIdCalled = func(address []byte) uint32 {
		if bytes.Equal(address, scr.SndAddr) {
			return 0
		}
		if bytes.Equal(address, scr.RcvAddr) {
			return 1
		}
		return 2
	}
	shardCoordinator.NumberOfShardsCalled = func() uint32 {
		return 3
	}
	shardCoordinator.SelfIdCalled = func() uint32 {
		return 1
	}

	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.Nil(t, err)

	scr.Data = []byte("function")
	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.Nil(t, err)

	scr.Data = []byte("function@01@02@03@04")
	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.Nil(t, err)

	scr.Data = []byte(core.BuiltInFunctionDCTNFTTransfer + "@01@02@03@04")
	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.NotNil(t, err)

	scr.Data = []byte(core.BuiltInFunctionDCTNFTTransfer + "@01@02@03@00")
	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.Nil(t, err)

	tickerID := []byte("TCK")
	dctData := &dct.DCToken{
		Value: big.NewInt(10),
		TokenMetaData: &dct.MetaData{
			Name: []byte("test"),
		},
	}
	dctMarshalled, _ := args.Marshalizer.Marshal(dctData)
	scr.Data = []byte(core.BuiltInFunctionDCTNFTTransfer + "@" + hex.EncodeToString(tickerID) + "@01@01@" + hex.EncodeToString(dctMarshalled))
	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.Nil(t, err)

	key := baseDCTKeyPrefix + string(tickerID)
	tokenKey := append([]byte(key), big.NewInt(1).Bytes()...)
	dctGetData, _, _ := e.getDCTDigitalTokenDataFromSystemAccount(tokenKey, defaultQueryOptions())

	assert.Equal(t, dctData.TokenMetaData, dctGetData.TokenMetaData)
}

func TestDctDataStorage_SaveNFTMetaDataToSystemAccountWithMultiTransfer(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewDCTDataStorage()
	shardCoordinator := &mock.ShardCoordinatorStub{}
	args.ShardCoordinator = shardCoordinator
	args.EnableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsSaveToSystemAccountFlagEnabledField: true,
	}
	e, _ := NewDCTDataStorage(args)

	scr := &smartContractResult.SmartContractResult{
		SndAddr: []byte("address1"),
		RcvAddr: []byte("address2"),
	}

	shardCoordinator.ComputeIdCalled = func(address []byte) uint32 {
		if bytes.Equal(address, scr.SndAddr) {
			return 0
		}
		if bytes.Equal(address, scr.RcvAddr) {
			return 1
		}
		return 2
	}
	shardCoordinator.NumberOfShardsCalled = func() uint32 {
		return 3
	}
	shardCoordinator.SelfIdCalled = func() uint32 {
		return 1
	}

	tickerID := []byte("TCK")
	dctData := &dct.DCToken{
		Value: big.NewInt(10),
		TokenMetaData: &dct.MetaData{
			Name: []byte("test"),
		},
	}
	dctMarshalled, _ := args.Marshalizer.Marshal(dctData)
	scr.Data = []byte(core.BuiltInFunctionMultiDCTNFTTransfer + "@00@" + hex.EncodeToString(tickerID) + "@01@01@" + hex.EncodeToString(dctMarshalled))
	err := e.SaveNFTMetaDataToSystemAccount(scr)
	assert.True(t, errors.Is(err, ErrInvalidArguments))

	scr.Data = []byte(core.BuiltInFunctionMultiDCTNFTTransfer + "@02@" + hex.EncodeToString(tickerID) + "@01@01@" + hex.EncodeToString(dctMarshalled))
	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.True(t, errors.Is(err, ErrInvalidArguments))

	scr.Data = []byte(core.BuiltInFunctionMultiDCTNFTTransfer + "@02@" + hex.EncodeToString(tickerID) + "@02@10@" +
		hex.EncodeToString(tickerID) + "@01@" + hex.EncodeToString(dctMarshalled))
	err = e.SaveNFTMetaDataToSystemAccount(scr)
	assert.Nil(t, err)

	key := baseDCTKeyPrefix + string(tickerID)
	tokenKey := append([]byte(key), big.NewInt(1).Bytes()...)
	dctGetData, _, _ := e.getDCTDigitalTokenDataFromSystemAccount(tokenKey, defaultQueryOptions())

	assert.Equal(t, dctData.TokenMetaData, dctGetData.TokenMetaData)

	otherTokenKey := append([]byte(key), big.NewInt(2).Bytes()...)
	dctGetData, _, err = e.getDCTDigitalTokenDataFromSystemAccount(otherTokenKey, defaultQueryOptions())
	assert.Nil(t, dctGetData)
	assert.Nil(t, err)
}

func TestDctDataStorage_checkCollectionFrozen(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewDCTDataStorage()
	shardCoordinator := &mock.ShardCoordinatorStub{}
	args.ShardCoordinator = shardCoordinator
	e, _ := NewDCTDataStorage(args)

	enableEpochsHandler, _ := args.EnableEpochsHandler.(*mock.EnableEpochsHandlerStub)
	enableEpochsHandler.IsCheckFrozenCollectionFlagEnabledField = false

	acnt, _ := e.accounts.LoadAccount([]byte("address1"))
	userAcc := acnt.(vmcommon.UserAccountHandler)

	tickerID := []byte("TOKEN-ABCDEF")
	dctTokenKey := append(e.keyPrefix, tickerID...)
	err := e.checkCollectionIsFrozenForAccount(userAcc, dctTokenKey, 1, false)
	assert.Nil(t, err)

	enableEpochsHandler.IsCheckFrozenCollectionFlagEnabledField = true
	err = e.checkCollectionIsFrozenForAccount(userAcc, dctTokenKey, 0, false)
	assert.Nil(t, err)

	err = e.checkCollectionIsFrozenForAccount(userAcc, dctTokenKey, 1, true)
	assert.Nil(t, err)

	err = e.checkCollectionIsFrozenForAccount(userAcc, dctTokenKey, 1, false)
	assert.Nil(t, err)

	tokenData, _ := getDCTDataFromKey(userAcc, dctTokenKey, e.marshaller)

	dctUserMetadata := DCTUserMetadataFromBytes(tokenData.Properties)
	dctUserMetadata.Frozen = false
	tokenData.Properties = dctUserMetadata.ToBytes()
	_ = saveDCTData(userAcc, tokenData, dctTokenKey, e.marshaller)

	err = e.checkCollectionIsFrozenForAccount(userAcc, dctTokenKey, 1, false)
	assert.Nil(t, err)

	dctUserMetadata.Frozen = true
	tokenData.Properties = dctUserMetadata.ToBytes()
	_ = saveDCTData(userAcc, tokenData, dctTokenKey, e.marshaller)

	err = e.checkCollectionIsFrozenForAccount(userAcc, dctTokenKey, 1, false)
	assert.Equal(t, err, ErrDCTIsFrozenForAccount)
}

func TestDctDataStorage_AddToLiquiditySystemAcc(t *testing.T) {
	t.Parallel()

	args := createMockArgsForNewDCTDataStorage()
	e, _ := NewDCTDataStorage(args)

	tokenKey := append(e.keyPrefix, []byte("TOKEN-ababab")...)
	nonce := uint64(10)
	err := e.AddToLiquiditySystemAcc(tokenKey, nonce, big.NewInt(10))
	assert.Equal(t, err, ErrNilDCTData)

	systemAcc, _ := e.getSystemAccount(defaultQueryOptions())
	dctData := &dct.DCToken{Value: big.NewInt(0)}
	marshalledData, _ := e.marshaller.Marshal(dctData)

	dctNFTTokenKey := computeDCTNFTTokenKey(tokenKey, nonce)
	_ = systemAcc.AccountDataHandler().SaveKeyValue(dctNFTTokenKey, marshalledData)

	err = e.AddToLiquiditySystemAcc(tokenKey, nonce, big.NewInt(10))
	assert.Nil(t, err)

	dctData = &dct.DCToken{Value: big.NewInt(10), Reserved: []byte{1}}
	marshalledData, _ = e.marshaller.Marshal(dctData)

	_ = systemAcc.AccountDataHandler().SaveKeyValue(dctNFTTokenKey, marshalledData)
	err = e.AddToLiquiditySystemAcc(tokenKey, nonce, big.NewInt(10))
	assert.Nil(t, err)

	dctData, _, _ = e.getDCTDigitalTokenDataFromSystemAccount(dctNFTTokenKey, defaultQueryOptions())
	assert.Equal(t, dctData.Value, big.NewInt(20))

	err = e.AddToLiquiditySystemAcc(tokenKey, nonce, big.NewInt(-20))
	assert.Nil(t, err)

	dctData, _, _ = e.getDCTDigitalTokenDataFromSystemAccount(dctNFTTokenKey, defaultQueryOptions())
	assert.Nil(t, dctData)
}
