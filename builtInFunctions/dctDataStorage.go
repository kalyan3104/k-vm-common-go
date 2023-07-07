package builtInFunctions

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/check"
	"github.com/kalyan3104/k-core/data"
	"github.com/kalyan3104/k-core/data/dct"
	vmcommon "github.com/kalyan3104/k-vm-common-go"
	"github.com/kalyan3104/k-vm-common-go/parsers"
)

const existsOnShard = byte(1)

type queryOptions struct {
	isCustomSystemAccountSet bool
	customSystemAccount      vmcommon.UserAccountHandler
}

func defaultQueryOptions() queryOptions {
	return queryOptions{}
}

type dctDataStorage struct {
	accounts              vmcommon.AccountsAdapter
	globalSettingsHandler vmcommon.DCTGlobalSettingsHandler
	marshaller            vmcommon.Marshalizer
	keyPrefix             []byte
	shardCoordinator      vmcommon.Coordinator
	txDataParser          vmcommon.CallArgsParser
	enableEpochsHandler   vmcommon.EnableEpochsHandler
}

// ArgsNewDCTDataStorage defines the argument list for new dct data storage handler
type ArgsNewDCTDataStorage struct {
	Accounts              vmcommon.AccountsAdapter
	GlobalSettingsHandler vmcommon.DCTGlobalSettingsHandler
	Marshalizer           vmcommon.Marshalizer
	EnableEpochsHandler   vmcommon.EnableEpochsHandler
	ShardCoordinator      vmcommon.Coordinator
}

// NewDCTDataStorage creates a new dct data storage handler
func NewDCTDataStorage(args ArgsNewDCTDataStorage) (*dctDataStorage, error) {
	if check.IfNil(args.Accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(args.GlobalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(args.Marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(args.EnableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}
	if check.IfNil(args.ShardCoordinator) {
		return nil, ErrNilShardCoordinator
	}

	e := &dctDataStorage{
		accounts:              args.Accounts,
		globalSettingsHandler: args.GlobalSettingsHandler,
		marshaller:            args.Marshalizer,
		keyPrefix:             []byte(baseDCTKeyPrefix),
		shardCoordinator:      args.ShardCoordinator,
		txDataParser:          parsers.NewCallArgsParser(),
		enableEpochsHandler:   args.EnableEpochsHandler,
	}

	return e, nil
}

// GetDCTNFTTokenOnSender gets the nft token on sender account
func (e *dctDataStorage) GetDCTNFTTokenOnSender(
	accnt vmcommon.UserAccountHandler,
	dctTokenKey []byte,
	nonce uint64,
) (*dct.DCToken, error) {
	dctData, isNew, err := e.GetDCTNFTTokenOnDestination(accnt, dctTokenKey, nonce)
	if err != nil {
		return nil, err
	}
	if isNew {
		return nil, ErrNewNFTDataOnSenderAddress
	}

	return dctData, nil
}

// GetDCTNFTTokenOnDestination gets the nft token on destination account
func (e *dctDataStorage) GetDCTNFTTokenOnDestination(
	accnt vmcommon.UserAccountHandler,
	dctTokenKey []byte,
	nonce uint64,
) (*dct.DCToken, bool, error) {
	return e.getDCTNFTTokenOnDestinationWithAccountsAdapterOptions(accnt, dctTokenKey, nonce, defaultQueryOptions())
}

// GetDCTNFTTokenOnDestinationWithCustomSystemAccount gets the nft token on destination account by using a custom system account
func (e *dctDataStorage) GetDCTNFTTokenOnDestinationWithCustomSystemAccount(
	accnt vmcommon.UserAccountHandler,
	dctTokenKey []byte,
	nonce uint64,
	customSystemAccount vmcommon.UserAccountHandler,
) (*dct.DCToken, bool, error) {
	if check.IfNil(customSystemAccount) {
		return nil, false, ErrNilUserAccount
	}

	queryOpts := queryOptions{
		isCustomSystemAccountSet: true,
		customSystemAccount:      customSystemAccount,
	}

	return e.getDCTNFTTokenOnDestinationWithAccountsAdapterOptions(accnt, dctTokenKey, nonce, queryOpts)
}

func (e *dctDataStorage) getDCTNFTTokenOnDestinationWithAccountsAdapterOptions(
	accnt vmcommon.UserAccountHandler,
	dctTokenKey []byte,
	nonce uint64,
	options queryOptions,
) (*dct.DCToken, bool, error) {
	dctNFTTokenKey := computeDCTNFTTokenKey(dctTokenKey, nonce)
	dctData := &dct.DCToken{
		Value: big.NewInt(0),
		Type:  uint32(core.Fungible),
	}
	marshaledData, _, err := accnt.AccountDataHandler().RetrieveValue(dctNFTTokenKey)
	if err != nil || len(marshaledData) == 0 {
		return dctData, true, nil
	}

	err = e.marshaller.Unmarshal(dctData, marshaledData)
	if err != nil {
		return nil, false, err
	}

	if !e.enableEpochsHandler.IsSaveToSystemAccountFlagEnabled() || nonce == 0 {
		return dctData, false, nil
	}

	dctMetaData, err := e.getDCTMetaDataFromSystemAccount(dctNFTTokenKey, options)
	if err != nil {
		return nil, false, err
	}
	if dctMetaData != nil {
		dctData.TokenMetaData = dctMetaData
	}

	return dctData, false, nil
}

func (e *dctDataStorage) getDCTDigitalTokenDataFromSystemAccount(
	tokenKey []byte,
	options queryOptions,
) (*dct.DCToken, vmcommon.UserAccountHandler, error) {
	systemAcc, err := e.getSystemAccount(options)
	if err != nil {
		return nil, nil, err
	}

	marshaledData, _, err := systemAcc.AccountDataHandler().RetrieveValue(tokenKey)
	if err != nil || len(marshaledData) == 0 {
		return nil, systemAcc, nil
	}

	dctData := &dct.DCToken{}
	err = e.marshaller.Unmarshal(dctData, marshaledData)
	if err != nil {
		return nil, nil, err
	}

	return dctData, systemAcc, nil
}

func (e *dctDataStorage) getDCTMetaDataFromSystemAccount(
	tokenKey []byte,
	options queryOptions,
) (*dct.MetaData, error) {
	dctData, _, err := e.getDCTDigitalTokenDataFromSystemAccount(tokenKey, options)
	if err != nil {
		return nil, err
	}
	if dctData == nil {
		return nil, nil
	}

	return dctData.TokenMetaData, nil
}

// CheckCollectionIsFrozenForAccount returns
func (e *dctDataStorage) checkCollectionIsFrozenForAccount(
	accnt vmcommon.UserAccountHandler,
	dctTokenKey []byte,
	nonce uint64,
	isReturnWithError bool,
) error {
	if !e.enableEpochsHandler.IsCheckFrozenCollectionFlagEnabled() {
		return nil
	}
	if nonce == 0 || isReturnWithError {
		return nil
	}

	dctData := &dct.DCToken{
		Value: big.NewInt(0),
		Type:  uint32(core.Fungible),
	}
	marshaledData, _, err := accnt.AccountDataHandler().RetrieveValue(dctTokenKey)
	if err != nil || len(marshaledData) == 0 {
		return nil
	}

	err = e.marshaller.Unmarshal(dctData, marshaledData)
	if err != nil {
		return err
	}

	dctUserMetaData := DCTUserMetadataFromBytes(dctData.Properties)
	if dctUserMetaData.Frozen {
		return ErrDCTIsFrozenForAccount
	}

	return nil
}

func (e *dctDataStorage) checkFrozenPauseProperties(
	acnt vmcommon.UserAccountHandler,
	dctTokenKey []byte,
	nonce uint64,
	dctData *dct.DCToken,
	isReturnWithError bool,
) error {
	err := checkFrozeAndPause(acnt.AddressBytes(), dctTokenKey, dctData, e.globalSettingsHandler, isReturnWithError)
	if err != nil {
		return err
	}

	dctNFTTokenKey := computeDCTNFTTokenKey(dctTokenKey, nonce)
	err = checkFrozeAndPause(acnt.AddressBytes(), dctNFTTokenKey, dctData, e.globalSettingsHandler, isReturnWithError)
	if err != nil {
		return err
	}

	err = e.checkCollectionIsFrozenForAccount(acnt, dctTokenKey, nonce, isReturnWithError)
	if err != nil {
		return err
	}

	return nil
}

// AddToLiquiditySystemAcc will increase/decrease the liquidity for DCT Tokens on the metadata
func (e *dctDataStorage) AddToLiquiditySystemAcc(
	dctTokenKey []byte,
	nonce uint64,
	transferValue *big.Int,
) error {
	isSaveToSystemAccountFlagEnabled := e.enableEpochsHandler.IsSaveToSystemAccountFlagEnabled()
	isSendAlwaysFlagEnabled := e.enableEpochsHandler.IsSendAlwaysFlagEnabled()
	if !isSaveToSystemAccountFlagEnabled || !isSendAlwaysFlagEnabled || nonce == 0 {
		return nil
	}

	dctNFTTokenKey := computeDCTNFTTokenKey(dctTokenKey, nonce)
	dctData, systemAcc, err := e.getDCTDigitalTokenDataFromSystemAccount(dctNFTTokenKey, defaultQueryOptions())
	if err != nil {
		return err
	}

	if dctData == nil {
		return ErrNilDCTData
	}

	// old style metaData - nothing to do
	if len(dctData.Reserved) == 0 {
		return nil
	}

	if e.enableEpochsHandler.IsFixOldTokenLiquidityEnabled() {
		// old tokens which were transferred intra shard before the activation of this flag
		if dctData.Value.Cmp(zero) == 0 && transferValue.Cmp(zero) < 0 {
			dctData.Reserved = nil
			return e.marshalAndSaveData(systemAcc, dctData, dctNFTTokenKey)
		}
	}

	dctData.Value.Add(dctData.Value, transferValue)
	if dctData.Value.Cmp(zero) < 0 {
		return ErrInvalidLiquidityForDCT
	}

	if dctData.Value.Cmp(zero) == 0 {
		err = systemAcc.AccountDataHandler().SaveKeyValue(dctNFTTokenKey, nil)
		if err != nil {
			return err
		}

		return e.accounts.SaveAccount(systemAcc)
	}

	err = e.marshalAndSaveData(systemAcc, dctData, dctNFTTokenKey)
	if err != nil {
		return err
	}

	return nil
}

// SaveDCTNFTToken saves the nft token to the account and system account
func (e *dctDataStorage) SaveDCTNFTToken(
	senderAddress []byte,
	acnt vmcommon.UserAccountHandler,
	dctTokenKey []byte,
	nonce uint64,
	dctData *dct.DCToken,
	mustUpdateAllFields bool,
	isReturnWithError bool,
) ([]byte, error) {
	err := e.checkFrozenPauseProperties(acnt, dctTokenKey, nonce, dctData, isReturnWithError)
	if err != nil {
		return nil, err
	}

	dctNFTTokenKey := computeDCTNFTTokenKey(dctTokenKey, nonce)
	senderShardID := e.shardCoordinator.ComputeId(senderAddress)
	if e.enableEpochsHandler.IsSaveToSystemAccountFlagEnabled() {
		err = e.saveDCTMetaDataToSystemAccount(acnt, senderShardID, dctNFTTokenKey, nonce, dctData, mustUpdateAllFields)
		if err != nil {
			return nil, err
		}
	}

	if dctData.Value.Cmp(zero) <= 0 {
		return nil, acnt.AccountDataHandler().SaveKeyValue(dctNFTTokenKey, nil)
	}

	if !e.enableEpochsHandler.IsSaveToSystemAccountFlagEnabled() {
		marshaledData, errMarshal := e.marshaller.Marshal(dctData)
		if errMarshal != nil {
			return nil, errMarshal
		}

		return marshaledData, acnt.AccountDataHandler().SaveKeyValue(dctNFTTokenKey, marshaledData)
	}

	dctDataOnAccount := &dct.DCToken{
		Type:       dctData.Type,
		Value:      dctData.Value,
		Properties: dctData.Properties,
	}
	marshaledData, err := e.marshaller.Marshal(dctDataOnAccount)
	if err != nil {
		return nil, err
	}

	return marshaledData, acnt.AccountDataHandler().SaveKeyValue(dctNFTTokenKey, marshaledData)
}

func (e *dctDataStorage) saveDCTMetaDataToSystemAccount(
	userAcc vmcommon.UserAccountHandler,
	senderShardID uint32,
	dctNFTTokenKey []byte,
	nonce uint64,
	dctData *dct.DCToken,
	mustUpdateAllFields bool,
) error {
	if nonce == 0 {
		return nil
	}
	if dctData.TokenMetaData == nil {
		return nil
	}

	systemAcc, err := e.getSystemAccount(defaultQueryOptions())
	if err != nil {
		return err
	}

	currentSaveData, _, _ := systemAcc.AccountDataHandler().RetrieveValue(dctNFTTokenKey)
	err = e.saveMetadataIfRequired(dctNFTTokenKey, systemAcc, currentSaveData, dctData)
	if err != nil {
		return err
	}

	if !mustUpdateAllFields && len(currentSaveData) > 0 {
		return nil
	}

	dctDataOnSystemAcc := &dct.DCToken{
		Type:          dctData.Type,
		Value:         big.NewInt(0),
		TokenMetaData: dctData.TokenMetaData,
		Properties:    make([]byte, e.shardCoordinator.NumberOfShards()),
	}
	isSendAlwaysFlagEnabled := e.enableEpochsHandler.IsSendAlwaysFlagEnabled()
	if len(currentSaveData) == 0 && isSendAlwaysFlagEnabled {
		dctDataOnSystemAcc.Properties = nil
		dctDataOnSystemAcc.Reserved = []byte{1}

		err = e.setReservedToNilForOldToken(dctDataOnSystemAcc, userAcc, dctNFTTokenKey)
		if err != nil {
			return err
		}
	}

	if !isSendAlwaysFlagEnabled {
		selfID := e.shardCoordinator.SelfId()
		if selfID != core.MetachainShardId {
			dctDataOnSystemAcc.Properties[selfID] = existsOnShard
		}
		if senderShardID != core.MetachainShardId {
			dctDataOnSystemAcc.Properties[senderShardID] = existsOnShard
		}
	}

	return e.marshalAndSaveData(systemAcc, dctDataOnSystemAcc, dctNFTTokenKey)
}

func (e *dctDataStorage) saveMetadataIfRequired(
	dctNFTTokenKey []byte,
	systemAcc vmcommon.UserAccountHandler,
	currentSaveData []byte,
	dctData *dct.DCToken,
) error {
	if !e.enableEpochsHandler.IsAlwaysSaveTokenMetaDataEnabled() {
		return nil
	}
	if !e.enableEpochsHandler.IsSendAlwaysFlagEnabled() {
		// do not re-write the metadata if it is not sent, as it will cause data loss
		return nil
	}
	if len(currentSaveData) == 0 {
		// optimization: do not try to write here the token metadata, it will be written automatically by the next step
		return nil
	}

	dctDataOnSystemAcc := &dct.DCToken{}
	err := e.marshaller.Unmarshal(dctDataOnSystemAcc, currentSaveData)
	if err != nil {
		return err
	}
	if len(dctDataOnSystemAcc.Reserved) > 0 {
		return nil
	}

	dctDataOnSystemAcc.TokenMetaData = dctData.TokenMetaData
	return e.marshalAndSaveData(systemAcc, dctDataOnSystemAcc, dctNFTTokenKey)
}

func (e *dctDataStorage) setReservedToNilForOldToken(
	dctDataOnSystemAcc *dct.DCToken,
	userAcc vmcommon.UserAccountHandler,
	dctNFTTokenKey []byte,
) error {
	if !e.enableEpochsHandler.IsFixOldTokenLiquidityEnabled() {
		return nil
	}

	if check.IfNil(userAcc) {
		return ErrNilUserAccount
	}
	dataOnUserAcc, _, errNotCritical := userAcc.AccountDataHandler().RetrieveValue(dctNFTTokenKey)
	shouldIgnoreToken := errNotCritical != nil || len(dataOnUserAcc) == 0
	if shouldIgnoreToken {
		return nil
	}

	dctDataOnUserAcc := &dct.DCToken{}
	err := e.marshaller.Unmarshal(dctDataOnUserAcc, dataOnUserAcc)
	if err != nil {
		return err
	}

	// tokens which were last moved before flagOptimizeNFTStore keep the dct metaData on the user account
	// these are not compatible with the new liquidity model,so we set the reserved field to nil
	if dctDataOnUserAcc.TokenMetaData != nil {
		dctDataOnSystemAcc.Reserved = nil
	}

	return nil
}

func (e *dctDataStorage) marshalAndSaveData(
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

	return e.accounts.SaveAccount(systemAcc)
}

func (e *dctDataStorage) getSystemAccount(options queryOptions) (vmcommon.UserAccountHandler, error) {
	if options.isCustomSystemAccountSet && !check.IfNil(options.customSystemAccount) {
		return options.customSystemAccount, nil
	}

	return e.loadSystemAccount()
}

func (e *dctDataStorage) loadSystemAccount() (vmcommon.UserAccountHandler, error) {
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

//TODO: merge properties in case of shard merge

// WasAlreadySentToDestinationShardAndUpdateState checks whether NFT metadata was sent to destination shard or not
// and saves the destination shard as sent
func (e *dctDataStorage) WasAlreadySentToDestinationShardAndUpdateState(
	tickerID []byte,
	nonce uint64,
	dstAddress []byte,
) (bool, error) {
	if !e.enableEpochsHandler.IsSaveToSystemAccountFlagEnabled() {
		return false, nil
	}

	if nonce == 0 {
		return true, nil
	}
	dstShardID := e.shardCoordinator.ComputeId(dstAddress)
	if dstShardID == e.shardCoordinator.SelfId() {
		return true, nil
	}

	if e.enableEpochsHandler.IsSendAlwaysFlagEnabled() {
		return false, nil
	}

	if dstShardID == core.MetachainShardId {
		return true, nil
	}
	dctTokenKey := append(e.keyPrefix, tickerID...)
	dctNFTTokenKey := computeDCTNFTTokenKey(dctTokenKey, nonce)

	dctData, systemAcc, err := e.getDCTDigitalTokenDataFromSystemAccount(dctNFTTokenKey, defaultQueryOptions())
	if err != nil {
		return false, err
	}
	if dctData == nil {
		return false, nil
	}

	if uint32(len(dctData.Properties)) < e.shardCoordinator.NumberOfShards() {
		newSlice := make([]byte, e.shardCoordinator.NumberOfShards())
		for i, val := range dctData.Properties {
			newSlice[i] = val
		}
		dctData.Properties = newSlice
	}

	if dctData.Properties[dstShardID] > 0 {
		return true, nil
	}

	dctData.Properties[dstShardID] = existsOnShard
	return false, e.marshalAndSaveData(systemAcc, dctData, dctNFTTokenKey)
}

// SaveNFTMetaDataToSystemAccount this saves the NFT metadata to the system account even if there was an error in processing
func (e *dctDataStorage) SaveNFTMetaDataToSystemAccount(
	tx data.TransactionHandler,
) error {
	if !e.enableEpochsHandler.IsSaveToSystemAccountFlagEnabled() {
		return nil
	}
	if e.enableEpochsHandler.IsSendAlwaysFlagEnabled() {
		return nil
	}
	if check.IfNil(tx) {
		return ErrNilTransactionHandler
	}

	sndShardID := e.shardCoordinator.ComputeId(tx.GetSndAddr())
	dstShardID := e.shardCoordinator.ComputeId(tx.GetRcvAddr())
	isCrossShardTxAtDest := sndShardID != dstShardID && e.shardCoordinator.SelfId() == dstShardID
	if !isCrossShardTxAtDest {
		return nil
	}

	function, arguments, err := e.txDataParser.ParseData(string(tx.GetData()))
	if err != nil {
		return nil
	}
	if len(arguments) < 4 {
		return nil
	}

	switch function {
	case core.BuiltInFunctionDCTNFTTransfer:
		return e.addMetaDataToSystemAccountFromNFTTransfer(sndShardID, arguments)
	case core.BuiltInFunctionMultiDCTNFTTransfer:
		return e.addMetaDataToSystemAccountFromMultiTransfer(sndShardID, arguments)
	default:
		return nil
	}
}

func (e *dctDataStorage) addMetaDataToSystemAccountFromNFTTransfer(
	sndShardID uint32,
	arguments [][]byte,
) error {
	if !bytes.Equal(arguments[3], zeroByteArray) {
		dctTransferData := &dct.DCToken{}
		err := e.marshaller.Unmarshal(dctTransferData, arguments[3])
		if err != nil {
			return err
		}
		dctTokenKey := append(e.keyPrefix, arguments[0]...)
		nonce := big.NewInt(0).SetBytes(arguments[1]).Uint64()
		dctNFTTokenKey := computeDCTNFTTokenKey(dctTokenKey, nonce)

		return e.saveDCTMetaDataToSystemAccount(nil, sndShardID, dctNFTTokenKey, nonce, dctTransferData, true)
	}
	return nil
}

func (e *dctDataStorage) addMetaDataToSystemAccountFromMultiTransfer(
	sndShardID uint32,
	arguments [][]byte,
) error {
	numOfTransfers := big.NewInt(0).SetBytes(arguments[0]).Uint64()
	if numOfTransfers == 0 {
		return fmt.Errorf("%w, 0 tokens to transfer", ErrInvalidArguments)
	}
	minNumOfArguments := numOfTransfers*argumentsPerTransfer + 1
	if uint64(len(arguments)) < minNumOfArguments {
		return fmt.Errorf("%w, invalid number of arguments", ErrInvalidArguments)
	}

	startIndex := uint64(1)
	for i := uint64(0); i < numOfTransfers; i++ {
		tokenStartIndex := startIndex + i*argumentsPerTransfer
		tokenID := arguments[tokenStartIndex]
		nonce := big.NewInt(0).SetBytes(arguments[tokenStartIndex+1]).Uint64()

		if nonce > 0 && len(arguments[tokenStartIndex+2]) > vmcommon.MaxLengthForValueToOptTransfer {
			dctTransferData := &dct.DCToken{}
			marshaledNFTTransfer := arguments[tokenStartIndex+2]
			err := e.marshaller.Unmarshal(dctTransferData, marshaledNFTTransfer)
			if err != nil {
				return fmt.Errorf("%w for token %s", err, string(tokenID))
			}

			dctTokenKey := append(e.keyPrefix, tokenID...)
			dctNFTTokenKey := computeDCTNFTTokenKey(dctTokenKey, nonce)
			err = e.saveDCTMetaDataToSystemAccount(nil, sndShardID, dctNFTTokenKey, nonce, dctTransferData, true)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *dctDataStorage) IsInterfaceNil() bool {
	return e == nil
}
