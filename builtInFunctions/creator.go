package builtInFunctions

import (
	"github.com/kalyan3104/k-core/core"
	"github.com/kalyan3104/k-core/core/check"
	vmcommon "github.com/kalyan3104/k-vm-common-go"
	"github.com/mitchellh/mapstructure"
)

var _ vmcommon.BuiltInFunctionFactory = (*builtInFuncCreator)(nil)

var trueHandler = func() bool { return true }
var falseHandler = func() bool { return false }

// ArgsCreateBuiltInFunctionContainer defines the input arguments to create built in functions container
type ArgsCreateBuiltInFunctionContainer struct {
	GasMap                           map[string]map[string]uint64
	MapDNSAddresses                  map[string]struct{}
	EnableUserNameChange             bool
	Marshalizer                      vmcommon.Marshalizer
	Accounts                         vmcommon.AccountsAdapter
	ShardCoordinator                 vmcommon.Coordinator
	EnableEpochsHandler              vmcommon.EnableEpochsHandler
	MaxNumOfAddressesForTransferRole uint32
	ConfigAddress                    []byte
}

type builtInFuncCreator struct {
	mapDNSAddresses                  map[string]struct{}
	enableUserNameChange             bool
	marshaller                       vmcommon.Marshalizer
	accounts                         vmcommon.AccountsAdapter
	builtInFunctions                 vmcommon.BuiltInFunctionContainer
	gasConfig                        *vmcommon.GasCost
	shardCoordinator                 vmcommon.Coordinator
	dctStorageHandler                vmcommon.DCTNFTStorageHandler
	dctGlobalSettingsHandler         vmcommon.DCTGlobalSettingsHandler
	enableEpochsHandler              vmcommon.EnableEpochsHandler
	maxNumOfAddressesForTransferRole uint32
	configAddress                    []byte
}

// NewBuiltInFunctionsCreator creates a component which will instantiate the built in functions contracts
func NewBuiltInFunctionsCreator(args ArgsCreateBuiltInFunctionContainer) (*builtInFuncCreator, error) {
	if check.IfNil(args.Marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(args.Accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if args.MapDNSAddresses == nil {
		return nil, ErrNilDnsAddresses
	}
	if check.IfNil(args.ShardCoordinator) {
		return nil, ErrNilShardCoordinator
	}
	if check.IfNil(args.EnableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}

	b := &builtInFuncCreator{
		mapDNSAddresses:                  args.MapDNSAddresses,
		enableUserNameChange:             args.EnableUserNameChange,
		marshaller:                       args.Marshalizer,
		accounts:                         args.Accounts,
		shardCoordinator:                 args.ShardCoordinator,
		enableEpochsHandler:              args.EnableEpochsHandler,
		maxNumOfAddressesForTransferRole: args.MaxNumOfAddressesForTransferRole,
		configAddress:                    args.ConfigAddress,
	}

	var err error
	b.gasConfig, err = createGasConfig(args.GasMap)
	if err != nil {
		return nil, err
	}
	b.builtInFunctions = NewBuiltInFunctionContainer()

	return b, nil
}

// GasScheduleChange is called when gas schedule is changed, thus all contracts must be updated
func (b *builtInFuncCreator) GasScheduleChange(gasSchedule map[string]map[string]uint64) {
	newGasConfig, err := createGasConfig(gasSchedule)
	if err != nil {
		return
	}

	b.gasConfig = newGasConfig
	for key := range b.builtInFunctions.Keys() {
		builtInFunc, errGet := b.builtInFunctions.Get(key)
		if errGet != nil {
			return
		}

		builtInFunc.SetNewGasConfig(b.gasConfig)
	}
}

// NFTStorageHandler will return the dct storage handler from the built in functions factory
func (b *builtInFuncCreator) NFTStorageHandler() vmcommon.SimpleDCTNFTStorageHandler {
	return b.dctStorageHandler
}

// DCTGlobalSettingsHandler will return the dct global settings handler from the built in functions factory
func (b *builtInFuncCreator) DCTGlobalSettingsHandler() vmcommon.DCTGlobalSettingsHandler {
	return b.dctGlobalSettingsHandler
}

// BuiltInFunctionContainer will return the built in function container
func (b *builtInFuncCreator) BuiltInFunctionContainer() vmcommon.BuiltInFunctionContainer {
	return b.builtInFunctions
}

// CreateBuiltInFunctionContainer will create the list of built-in functions
func (b *builtInFuncCreator) CreateBuiltInFunctionContainer() error {

	b.builtInFunctions = NewBuiltInFunctionContainer()
	var newFunc vmcommon.BuiltinFunction
	newFunc = NewClaimDeveloperRewardsFunc(b.gasConfig.BuiltInCost.ClaimDeveloperRewards)
	err := b.builtInFunctions.Add(core.BuiltInFunctionClaimDeveloperRewards, newFunc)
	if err != nil {
		return err
	}

	newFunc = NewChangeOwnerAddressFunc(b.gasConfig.BuiltInCost.ChangeOwnerAddress)
	err = b.builtInFunctions.Add(core.BuiltInFunctionChangeOwnerAddress, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewSaveUserNameFunc(b.gasConfig.BuiltInCost.SaveUserName, b.mapDNSAddresses, b.enableUserNameChange)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionSetUserName, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewSaveKeyValueStorageFunc(b.gasConfig.BaseOperationCost, b.gasConfig.BuiltInCost.SaveKeyValue)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionSaveKeyValue, newFunc)
	if err != nil {
		return err
	}

	globalSettingsFunc, err := NewDCTGlobalSettingsFunc(b.accounts, b.marshaller, true, core.BuiltInFunctionDCTPause, trueHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCTPause, globalSettingsFunc)
	if err != nil {
		return err
	}
	b.dctGlobalSettingsHandler = globalSettingsFunc

	setRoleFunc, err := NewDCTRolesFunc(b.marshaller, true)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionSetDCTRole, setRoleFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTTransferFunc(
		b.gasConfig.BuiltInCost.DCTTransfer,
		b.marshaller,
		globalSettingsFunc,
		b.shardCoordinator,
		setRoleFunc,
		b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCTTransfer, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTBurnFunc(b.gasConfig.BuiltInCost.DCTBurn, b.marshaller, globalSettingsFunc, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCTBurn, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTGlobalSettingsFunc(b.accounts, b.marshaller, false, core.BuiltInFunctionDCTUnPause, trueHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCTUnPause, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTRolesFunc(b.marshaller, false)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionUnSetDCTRole, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTLocalBurnFunc(b.gasConfig.BuiltInCost.DCTLocalBurn, b.marshaller, globalSettingsFunc, setRoleFunc)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCTLocalBurn, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTLocalMintFunc(b.gasConfig.BuiltInCost.DCTLocalMint, b.marshaller, globalSettingsFunc, setRoleFunc)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCTLocalMint, newFunc)
	if err != nil {
		return err
	}

	args := ArgsNewDCTDataStorage{
		Accounts:              b.accounts,
		GlobalSettingsHandler: globalSettingsFunc,
		Marshalizer:           b.marshaller,
		EnableEpochsHandler:   b.enableEpochsHandler,
		ShardCoordinator:      b.shardCoordinator,
	}
	b.dctStorageHandler, err = NewDCTDataStorage(args)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTNFTAddQuantityFunc(b.gasConfig.BuiltInCost.DCTNFTAddQuantity, b.dctStorageHandler, globalSettingsFunc, setRoleFunc, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCTNFTAddQuantity, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTNFTBurnFunc(b.gasConfig.BuiltInCost.DCTNFTBurn, b.dctStorageHandler, globalSettingsFunc, setRoleFunc)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCTNFTBurn, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTNFTCreateFunc(b.gasConfig.BuiltInCost.DCTNFTCreate, b.gasConfig.BaseOperationCost, b.marshaller, globalSettingsFunc, setRoleFunc, b.dctStorageHandler, b.accounts, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCTNFTCreate, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTFreezeWipeFunc(b.dctStorageHandler, b.enableEpochsHandler, b.marshaller, true, false)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCTFreeze, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTFreezeWipeFunc(b.dctStorageHandler, b.enableEpochsHandler, b.marshaller, false, false)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCTUnFreeze, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTFreezeWipeFunc(b.dctStorageHandler, b.enableEpochsHandler, b.marshaller, false, true)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCTWipe, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTNFTTransferFunc(b.gasConfig.BuiltInCost.DCTNFTTransfer,
		b.marshaller,
		globalSettingsFunc,
		b.accounts,
		b.shardCoordinator,
		b.gasConfig.BaseOperationCost,
		setRoleFunc,
		b.dctStorageHandler,
		b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCTNFTTransfer, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTNFTCreateRoleTransfer(b.marshaller, b.accounts, b.shardCoordinator)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCTNFTCreateRoleTransfer, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTNFTUpdateAttributesFunc(b.gasConfig.BuiltInCost.DCTNFTUpdateAttributes, b.gasConfig.BaseOperationCost, b.dctStorageHandler, globalSettingsFunc, setRoleFunc, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCTNFTUpdateAttributes, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTNFTAddUriFunc(b.gasConfig.BuiltInCost.DCTNFTAddURI, b.gasConfig.BaseOperationCost, b.dctStorageHandler, globalSettingsFunc, setRoleFunc, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCTNFTAddURI, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTNFTMultiTransferFunc(b.gasConfig.BuiltInCost.DCTNFTMultiTransfer,
		b.marshaller,
		globalSettingsFunc,
		b.accounts,
		b.shardCoordinator,
		b.gasConfig.BaseOperationCost,
		b.enableEpochsHandler,
		setRoleFunc,
		b.dctStorageHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMultiDCTNFTTransfer, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTGlobalSettingsFunc(b.accounts, b.marshaller, true, core.BuiltInFunctionDCTSetLimitedTransfer, b.enableEpochsHandler.IsDCTTransferRoleFlagEnabled)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCTSetLimitedTransfer, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTGlobalSettingsFunc(b.accounts, b.marshaller, false, core.BuiltInFunctionDCTUnSetLimitedTransfer, b.enableEpochsHandler.IsDCTTransferRoleFlagEnabled)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCTUnSetLimitedTransfer, newFunc)
	if err != nil {
		return err
	}

	argsNewDeleteFunc := ArgsNewDCTDeleteMetadata{
		FuncGasCost:         b.gasConfig.BuiltInCost.DCTNFTBurn,
		Marshalizer:         b.marshaller,
		Accounts:            b.accounts,
		AllowedAddress:      b.configAddress,
		Delete:              true,
		EnableEpochsHandler: b.enableEpochsHandler,
	}
	newFunc, err = NewDCTDeleteMetadataFunc(argsNewDeleteFunc)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.DCTDeleteMetadata, newFunc)
	if err != nil {
		return err
	}

	argsNewDeleteFunc.Delete = false
	newFunc, err = NewDCTDeleteMetadataFunc(argsNewDeleteFunc)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.DCTAddMetadata, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTGlobalSettingsFunc(b.accounts, b.marshaller, true, vmcommon.BuiltInFunctionDCTSetBurnRoleForAll, b.enableEpochsHandler.IsSendAlwaysFlagEnabled)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionDCTSetBurnRoleForAll, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTGlobalSettingsFunc(b.accounts, b.marshaller, false, vmcommon.BuiltInFunctionDCTUnSetBurnRoleForAll, b.enableEpochsHandler.IsSendAlwaysFlagEnabled)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionDCTUnSetBurnRoleForAll, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTTransferRoleAddressFunc(b.accounts, b.marshaller, b.maxNumOfAddressesForTransferRole, false, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionDCTTransferRoleDeleteAddress, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCTTransferRoleAddressFunc(b.accounts, b.marshaller, b.maxNumOfAddressesForTransferRole, true, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionDCTTransferRoleAddAddress, newFunc)
	if err != nil {
		return err
	}

	return nil
}

func createGasConfig(gasMap map[string]map[string]uint64) (*vmcommon.GasCost, error) {
	baseOps := &vmcommon.BaseOperationCost{}
	err := mapstructure.Decode(gasMap[core.BaseOperationCostString], baseOps)
	if err != nil {
		return nil, err
	}

	err = check.ForZeroUintFields(*baseOps)
	if err != nil {
		return nil, err
	}

	builtInOps := &vmcommon.BuiltInCost{}
	err = mapstructure.Decode(gasMap[core.BuiltInCostString], builtInOps)
	if err != nil {
		return nil, err
	}

	err = check.ForZeroUintFields(*builtInOps)
	if err != nil {
		return nil, err
	}

	gasCost := vmcommon.GasCost{
		BaseOperationCost: *baseOps,
		BuiltInCost:       *builtInOps,
	}

	return &gasCost, nil
}

// SetPayableHandler sets the payableCheck interface to the needed functions
func (b *builtInFuncCreator) SetPayableHandler(payableHandler vmcommon.PayableHandler) error {
	payableChecker, err := NewPayableCheckFunc(
		payableHandler,
		b.enableEpochsHandler,
	)
	if err != nil {
		return err
	}

	listOfTransferFunc := []string{
		core.BuiltInFunctionMultiDCTNFTTransfer,
		core.BuiltInFunctionDCTNFTTransfer,
		core.BuiltInFunctionDCTTransfer}

	for _, transferFunc := range listOfTransferFunc {
		builtInFunc, err := b.builtInFunctions.Get(transferFunc)
		if err != nil {
			return err
		}

		dctTransferFunc, ok := builtInFunc.(vmcommon.AcceptPayableChecker)
		if !ok {
			return ErrWrongTypeAssertion
		}

		err = dctTransferFunc.SetPayableChecker(payableChecker)
		if err != nil {
			return err
		}
	}

	return nil
}

// IsInterfaceNil returns true if underlying object is nil
func (b *builtInFuncCreator) IsInterfaceNil() bool {
	return b == nil
}
