package wzeromark

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHKDFKeyGen(t *testing.T) {
	orgMasterKey := []byte("this_is_a_test_organization_master_key")
	systemSalt := []byte("system_wide_salt_value")
	keyGen := newHmacKeygen(orgMasterKey, systemSalt)

	timestamp := time.Date(2025, 11, 12, 9, 0, 0, 0, time.UTC)
	key, err := keyGen.Generate(timestamp)
	require.NoError(t, err)
	require.Len(t, key, keyLen)

	// Generate the key again with the same parameters to ensure consistency
	key2, err := keyGen.Generate(timestamp)
	require.NoError(t, err)
	require.Equal(t, key, key2)

	// Generate the key with a different timestamp to ensure it changes
	timestamp2 := time.Date(2025, 11, 12, 10, 0, 0, 0, time.UTC)
	key3, err := keyGen.Generate(timestamp2)
	require.NoError(t, err)
	require.NotEqual(t, key, key3)

	// Generate the key with a different timestamp in the same hour to ensure it remains the same
	timestamp3 := time.Date(2025, 11, 12, 9, 30, 0, 0, time.UTC)
	key4, err := keyGen.Generate(timestamp3)
	require.NoError(t, err)
	require.Equal(t, key, key4)

	// Generate the key with a different organization master key to ensure it changes
	orgMasterKey2 := []byte("this_is_a_different_organization_master_key")
	keyGen2 := newHmacKeygen(orgMasterKey2, systemSalt)
	key5, err := keyGen2.Generate(timestamp)
	require.NoError(t, err)
	require.NotEqual(t, key, key5)

	// Generate the key with a different system salt to ensure it changes
	systemSalt2 := []byte("different_system_salt_value")
	keyGen3 := newHmacKeygen(orgMasterKey, systemSalt2)
	key6, err := keyGen3.Generate(timestamp)
	require.NoError(t, err)
	require.NotEqual(t, key, key6)

	// Generate the key using ed25519 keygen to ensure different output
	ed25519KeyGen := newEd25519Keygen(orgMasterKey, systemSalt)
	key7, err := ed25519KeyGen.Generate(timestamp)
	require.NoError(t, err)
	require.NotEqual(t, key, key7)
}
