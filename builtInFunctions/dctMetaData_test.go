package builtInFunctions

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDCTGlobalMetaData_ToBytesWhenPaused(t *testing.T) {
	t.Parallel()

	dctMetaData := &DCTGlobalMetadata{
		Paused: true,
	}

	expected := make([]byte, lengthOfDCTMetadata)
	expected[0] = 1
	actual := dctMetaData.ToBytes()
	require.Equal(t, expected, actual)
}

func TestDCTGlobalMetaData_ToBytesWhenTransfer(t *testing.T) {
	t.Parallel()

	dctMetaData := &DCTGlobalMetadata{
		LimitedTransfer: true,
	}

	expected := make([]byte, lengthOfDCTMetadata)
	expected[0] = 2
	actual := dctMetaData.ToBytes()
	require.Equal(t, expected, actual)
}

func TestDCTGlobalMetaData_ToBytesWhenTransferAndPause(t *testing.T) {
	t.Parallel()

	dctMetaData := &DCTGlobalMetadata{
		Paused:          true,
		LimitedTransfer: true,
	}

	expected := make([]byte, lengthOfDCTMetadata)
	expected[0] = 3
	actual := dctMetaData.ToBytes()
	require.Equal(t, expected, actual)
}

func TestDCTGlobalMetaData_ToBytesWhenNotPaused(t *testing.T) {
	t.Parallel()

	dctMetaData := &DCTGlobalMetadata{
		Paused: false,
	}

	expected := make([]byte, lengthOfDCTMetadata)
	expected[0] = 0
	actual := dctMetaData.ToBytes()
	require.Equal(t, expected, actual)
}

func TestDCTGlobalMetadataFromBytes_InvalidLength(t *testing.T) {
	t.Parallel()

	emptyDctGlobalMetaData := DCTGlobalMetadata{}

	invalidLengthByteSlice := make([]byte, lengthOfDCTMetadata+1)

	result := DCTGlobalMetadataFromBytes(invalidLengthByteSlice)
	require.Equal(t, emptyDctGlobalMetaData, result)
}

func TestDCTGlobalMetadataFromBytes_ShouldSetPausedToTrue(t *testing.T) {
	t.Parallel()

	input := make([]byte, lengthOfDCTMetadata)
	input[0] = 1

	result := DCTGlobalMetadataFromBytes(input)
	require.True(t, result.Paused)
}

func TestDCTGlobalMetadataFromBytes_ShouldSetPausedToFalse(t *testing.T) {
	t.Parallel()

	input := make([]byte, lengthOfDCTMetadata)
	input[0] = 0

	result := DCTGlobalMetadataFromBytes(input)
	require.False(t, result.Paused)
}

func TestDCTUserMetaData_ToBytesWhenFrozen(t *testing.T) {
	t.Parallel()

	dctMetaData := &DCTUserMetadata{
		Frozen: true,
	}

	expected := make([]byte, lengthOfDCTMetadata)
	expected[0] = 1
	actual := dctMetaData.ToBytes()
	require.Equal(t, expected, actual)
}

func TestDCTUserMetaData_ToBytesWhenNotFrozen(t *testing.T) {
	t.Parallel()

	dctMetaData := &DCTUserMetadata{
		Frozen: false,
	}

	expected := make([]byte, lengthOfDCTMetadata)
	expected[0] = 0
	actual := dctMetaData.ToBytes()
	require.Equal(t, expected, actual)
}

func TestDCTUserMetadataFromBytes_InvalidLength(t *testing.T) {
	t.Parallel()

	emptyDctUserMetaData := DCTUserMetadata{}

	invalidLengthByteSlice := make([]byte, lengthOfDCTMetadata+1)

	result := DCTUserMetadataFromBytes(invalidLengthByteSlice)
	require.Equal(t, emptyDctUserMetaData, result)
}

func TestDCTUserMetadataFromBytes_ShouldSetFrozenToTrue(t *testing.T) {
	t.Parallel()

	input := make([]byte, lengthOfDCTMetadata)
	input[0] = 1

	result := DCTUserMetadataFromBytes(input)
	require.True(t, result.Frozen)
}

func TestDCTUserMetadataFromBytes_ShouldSetFrozenToFalse(t *testing.T) {
	t.Parallel()

	input := make([]byte, lengthOfDCTMetadata)
	input[0] = 0

	result := DCTUserMetadataFromBytes(input)
	require.False(t, result.Frozen)
}

func TestDCTGlobalMetadata_FromBytes(t *testing.T) {
	require.True(t, DCTGlobalMetadataFromBytes([]byte{1, 0}).Paused)
	require.False(t, DCTGlobalMetadataFromBytes([]byte{1, 0}).LimitedTransfer)
	require.True(t, DCTGlobalMetadataFromBytes([]byte{2, 0}).LimitedTransfer)
	require.False(t, DCTGlobalMetadataFromBytes([]byte{2, 0}).Paused)
	require.False(t, DCTGlobalMetadataFromBytes([]byte{0, 0}).LimitedTransfer)
	require.False(t, DCTGlobalMetadataFromBytes([]byte{0, 0}).Paused)
	require.True(t, DCTGlobalMetadataFromBytes([]byte{3, 0}).Paused)
	require.True(t, DCTGlobalMetadataFromBytes([]byte{3, 0}).LimitedTransfer)
}
