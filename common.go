package vmcommon

import "math/big"

const tickerMinLength = 3
const tickerMaxLength = 10
const additionalRandomCharsLength = 6
const identifierMinLength = tickerMinLength + additionalRandomCharsLength + 1
const identifierMaxLength = tickerMaxLength + additionalRandomCharsLength + 1

// DCTDeleteMetadata represents the defined built in function name for dct delete metadata
const DCTDeleteMetadata = "DCTDeleteMetadata"

// DCTAddMetadata represents the defined built in function name for dct add metadata
const DCTAddMetadata = "DCTAddMetadata"

// BuiltInFunctionDCTSetBurnRoleForAll represents the defined built in function name for dct set burn role for all
const BuiltInFunctionDCTSetBurnRoleForAll = "DCTSetBurnRoleForAll"

// BuiltInFunctionDCTUnSetBurnRoleForAll represents the defined built in function name for dct unset burn role for all
const BuiltInFunctionDCTUnSetBurnRoleForAll = "DCTUnSetBurnRoleForAll"

// BuiltInFunctionDCTTransferRoleAddAddress represents the defined built in function name for dct transfer role add address
const BuiltInFunctionDCTTransferRoleAddAddress = "DCTTransferRoleAddAddress"

// BuiltInFunctionDCTTransferRoleDeleteAddress represents the defined built in function name for transfer role delete address
const BuiltInFunctionDCTTransferRoleDeleteAddress = "DCTTransferRoleDeleteAddress"

// DCTRoleBurnForAll represents the role for burn for all
const DCTRoleBurnForAll = "DCTRoleBurnForAll"

// ValidateToken - validates the token ID
func ValidateToken(tokenID []byte) bool {
	tokenIDLen := len(tokenID)
	if tokenIDLen < identifierMinLength || tokenIDLen > identifierMaxLength {
		return false
	}

	tickerLen := tokenIDLen - additionalRandomCharsLength

	if !isTickerValid(tokenID[0 : tickerLen-1]) {
		return false
	}

	// dash char between the random chars and the ticker
	if tokenID[tickerLen-1] != '-' {
		return false
	}

	if !randomCharsAreValid(tokenID[tickerLen:tokenIDLen]) {
		return false
	}

	return true
}

// ticker must be all uppercase alphanumeric
func isTickerValid(tickerName []byte) bool {
	if len(tickerName) < tickerMinLength || len(tickerName) > tickerMaxLength {
		return false
	}
	for _, ch := range tickerName {
		isBigCharacter := ch >= 'A' && ch <= 'Z'
		isNumber := ch >= '0' && ch <= '9'
		isReadable := isBigCharacter || isNumber
		if !isReadable {
			return false
		}
	}

	return true
}

// random chars are alphanumeric lowercase
func randomCharsAreValid(chars []byte) bool {
	if len(chars) != additionalRandomCharsLength {
		return false
	}
	for _, ch := range chars {
		isSmallCharacter := ch >= 'a' && ch <= 'f'
		isNumber := ch >= '0' && ch <= '9'
		isReadable := isSmallCharacter || isNumber
		if !isReadable {
			return false
		}
	}

	return true
}

// ZeroValueIfNil returns 0 if the input is nil, otherwise returns the input
func ZeroValueIfNil(value *big.Int) *big.Int {
	if value == nil {
		return big.NewInt(0)
	}

	return value
}
