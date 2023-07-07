package builtInFunctions

const lengthOfDCTMetadata = 2

const (
	// MetadataPaused is the location of paused flag in the dct global meta data
	MetadataPaused = 1
	// MetadataLimitedTransfer is the location of limited transfer flag in the dct global meta data
	MetadataLimitedTransfer = 2
	// BurnRoleForAll is the location of burn role for all flag in the dct global meta data
	BurnRoleForAll = 4
)

const (
	// MetadataFrozen is the location of frozen flag in the dct user meta data
	MetadataFrozen = 1
)

// DCTGlobalMetadata represents dct global metadata saved on system account
type DCTGlobalMetadata struct {
	Paused          bool
	LimitedTransfer bool
	BurnRoleForAll  bool
}

// DCTGlobalMetadataFromBytes creates a metadata object from bytes
func DCTGlobalMetadataFromBytes(bytes []byte) DCTGlobalMetadata {
	if len(bytes) != lengthOfDCTMetadata {
		return DCTGlobalMetadata{}
	}

	return DCTGlobalMetadata{
		Paused:          (bytes[0] & MetadataPaused) != 0,
		LimitedTransfer: (bytes[0] & MetadataLimitedTransfer) != 0,
		BurnRoleForAll:  (bytes[0] & BurnRoleForAll) != 0,
	}
}

// ToBytes converts the metadata to bytes
func (metadata *DCTGlobalMetadata) ToBytes() []byte {
	bytes := make([]byte, lengthOfDCTMetadata)

	if metadata.Paused {
		bytes[0] |= MetadataPaused
	}
	if metadata.LimitedTransfer {
		bytes[0] |= MetadataLimitedTransfer
	}
	if metadata.BurnRoleForAll {
		bytes[0] |= BurnRoleForAll
	}

	return bytes
}

// DCTUserMetadata represents dct user metadata saved on every account
type DCTUserMetadata struct {
	Frozen bool
}

// DCTUserMetadataFromBytes creates a metadata object from bytes
func DCTUserMetadataFromBytes(bytes []byte) DCTUserMetadata {
	if len(bytes) != lengthOfDCTMetadata {
		return DCTUserMetadata{}
	}

	return DCTUserMetadata{
		Frozen: (bytes[0] & MetadataFrozen) != 0,
	}
}

// ToBytes converts the metadata to bytes
func (metadata *DCTUserMetadata) ToBytes() []byte {
	bytes := make([]byte, lengthOfDCTMetadata)

	if metadata.Frozen {
		bytes[0] |= MetadataFrozen
	}

	return bytes
}
