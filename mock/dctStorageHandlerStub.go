package mock

import (
	"math/big"

	"github.com/kalyan3104/k-core/data"
	"github.com/kalyan3104/k-core/data/dct"
	vmcommon "github.com/kalyan3104/k-vm-common-go"
)

// DCTNFTStorageHandlerStub -
type DCTNFTStorageHandlerStub struct {
	SaveDCTNFTTokenCalled                                    func(senderAddress []byte, acnt vmcommon.UserAccountHandler, dctTokenKey []byte, nonce uint64, dctData *dct.DCToken, mustUpdateAllFields bool, isReturnWithError bool) ([]byte, error)
	GetDCTNFTTokenOnSenderCalled                             func(acnt vmcommon.UserAccountHandler, dctTokenKey []byte, nonce uint64) (*dct.DCToken, error)
	GetDCTNFTTokenOnDestinationCalled                        func(acnt vmcommon.UserAccountHandler, dctTokenKey []byte, nonce uint64) (*dct.DCToken, bool, error)
	GetDCTNFTTokenOnDestinationWithCustomSystemAccountCalled func(accnt vmcommon.UserAccountHandler, dctTokenKey []byte, nonce uint64, systemAccount vmcommon.UserAccountHandler) (*dct.DCToken, bool, error)
	WasAlreadySentToDestinationShardAndUpdateStateCalled     func(tickerID []byte, nonce uint64, dstAddress []byte) (bool, error)
	SaveNFTMetaDataToSystemAccountCalled                     func(tx data.TransactionHandler) error
	AddToLiquiditySystemAccCalled                            func(dctTokenKey []byte, nonce uint64, transferValue *big.Int) error
}

// SaveDCTNFTToken -
func (stub *DCTNFTStorageHandlerStub) SaveDCTNFTToken(senderAddress []byte, acnt vmcommon.UserAccountHandler, dctTokenKey []byte, nonce uint64, dctData *dct.DCToken, mustUpdateAllFields bool, isReturnWithError bool) ([]byte, error) {
	if stub.SaveDCTNFTTokenCalled != nil {
		return stub.SaveDCTNFTTokenCalled(senderAddress, acnt, dctTokenKey, nonce, dctData, mustUpdateAllFields, isReturnWithError)
	}
	return nil, nil
}

// GetDCTNFTTokenOnSender -
func (stub *DCTNFTStorageHandlerStub) GetDCTNFTTokenOnSender(acnt vmcommon.UserAccountHandler, dctTokenKey []byte, nonce uint64) (*dct.DCToken, error) {
	if stub.GetDCTNFTTokenOnSenderCalled != nil {
		return stub.GetDCTNFTTokenOnSenderCalled(acnt, dctTokenKey, nonce)
	}
	return nil, nil
}

// GetDCTNFTTokenOnDestination -
func (stub *DCTNFTStorageHandlerStub) GetDCTNFTTokenOnDestination(acnt vmcommon.UserAccountHandler, dctTokenKey []byte, nonce uint64) (*dct.DCToken, bool, error) {
	if stub.GetDCTNFTTokenOnDestinationCalled != nil {
		return stub.GetDCTNFTTokenOnDestinationCalled(acnt, dctTokenKey, nonce)
	}
	return nil, false, nil
}

// GetDCTNFTTokenOnDestinationWithCustomSystemAccount -
func (stub *DCTNFTStorageHandlerStub) GetDCTNFTTokenOnDestinationWithCustomSystemAccount(accnt vmcommon.UserAccountHandler, dctTokenKey []byte, nonce uint64, systemAccount vmcommon.UserAccountHandler) (*dct.DCToken, bool, error) {
	if stub.GetDCTNFTTokenOnDestinationWithCustomSystemAccountCalled != nil {
		return stub.GetDCTNFTTokenOnDestinationWithCustomSystemAccountCalled(accnt, dctTokenKey, nonce, systemAccount)
	}
	return nil, false, nil
}

// WasAlreadySentToDestinationShardAndUpdateState -
func (stub *DCTNFTStorageHandlerStub) WasAlreadySentToDestinationShardAndUpdateState(tickerID []byte, nonce uint64, dstAddress []byte) (bool, error) {
	if stub.WasAlreadySentToDestinationShardAndUpdateStateCalled != nil {
		return stub.WasAlreadySentToDestinationShardAndUpdateStateCalled(tickerID, nonce, dstAddress)
	}
	return false, nil
}

// SaveNFTMetaDataToSystemAccount -
func (stub *DCTNFTStorageHandlerStub) SaveNFTMetaDataToSystemAccount(tx data.TransactionHandler) error {
	if stub.SaveNFTMetaDataToSystemAccountCalled != nil {
		return stub.SaveNFTMetaDataToSystemAccountCalled(tx)
	}
	return nil
}

// AddToLiquiditySystemAcc -
func (stub *DCTNFTStorageHandlerStub) AddToLiquiditySystemAcc(dctTokenKey []byte, nonce uint64, transferValue *big.Int) error {
	if stub.AddToLiquiditySystemAccCalled != nil {
		return stub.AddToLiquiditySystemAccCalled(dctTokenKey, nonce, transferValue)
	}
	return nil
}

// IsInterfaceNil -
func (stub *DCTNFTStorageHandlerStub) IsInterfaceNil() bool {
	return stub == nil
}
