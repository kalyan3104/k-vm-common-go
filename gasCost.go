package vmcommon

// BaseOperationCost defines cost for base operation cost
type BaseOperationCost struct {
	StorePerByte      uint64
	ReleasePerByte    uint64
	DataCopyPerByte   uint64
	PersistPerByte    uint64
	CompilePerByte    uint64
	AoTPreparePerByte uint64
}

// BuiltInCost defines cost for built-in methods
type BuiltInCost struct {
	ChangeOwnerAddress      uint64
	ClaimDeveloperRewards   uint64
	SaveUserName            uint64
	SaveKeyValue            uint64
	DCTTransfer             uint64
	DCTBurn                 uint64
	DCTLocalMint            uint64
	DCTLocalBurn            uint64
	DCTNFTCreate            uint64
	DCTNFTAddQuantity       uint64
	DCTNFTBurn              uint64
	DCTNFTTransfer          uint64
	DCTNFTChangeCreateOwner uint64
	DCTNFTMultiTransfer     uint64
	DCTNFTAddURI            uint64
	DCTNFTUpdateAttributes  uint64
}

// GasCost holds all the needed gas costs for system smart contracts
type GasCost struct {
	BaseOperationCost BaseOperationCost
	BuiltInCost       BuiltInCost
}

// SafeSubUint64 performs subtraction on uint64 and returns an error if it overflows
func SafeSubUint64(a, b uint64) (uint64, error) {
	if a < b {
		return 0, ErrSubtractionOverflow
	}
	return a - b, nil
}
