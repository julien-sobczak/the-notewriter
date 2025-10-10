package aes_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/julien-sobczak/the-notewriter/pkg/aes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadKey(t *testing.T) {

	t.Run("EnvNotSet", func(t *testing.T) {
		orig := os.Getenv("NT_VAULT_AES_KEY_FILE")
		os.Unsetenv("NT_VAULT_AES_KEY_FILE")
		defer os.Setenv("NT_VAULT_AES_KEY_FILE", orig)

		key, err := aes.LoadKey()
		assert.Nil(t, key)
		assert.ErrorContains(t, err, "NT_VAULT_AES_KEY_FILE environment variable is not set")
	})

	t.Run("FileDoesNotExist", func(t *testing.T) {
		tmp := filepath.Join(os.TempDir(), "nonexistent-key-file")
		os.Setenv("NT_VAULT_AES_KEY_FILE", tmp)
		key, err := aes.LoadKey()
		assert.Nil(t, key)
		assert.ErrorContains(t, err, "cannot stat key file")
	})

	t.Run("PermissionsTooOpen", func(t *testing.T) {
		tmp := filepath.Join(os.TempDir(), "testkey-perms")
		_ = os.WriteFile(tmp, make([]byte, 32), 0644)
		defer os.Remove(tmp)
		os.Setenv("NT_VAULT_AES_KEY_FILE", tmp)

		key, err := aes.LoadKey()
		assert.Nil(t, key)
		assert.ErrorContains(t, err, "key file has too open permissions")
	})

	t.Run("WrongKeyLength", func(t *testing.T) {
		tmp := filepath.Join(os.TempDir(), "testkey-length")
		_ = os.WriteFile(tmp, make([]byte, 16), 0600)
		defer os.Remove(tmp)
		os.Setenv("NT_VAULT_AES_KEY_FILE", tmp)

		key, err := aes.LoadKey()
		assert.Nil(t, key)
		assert.ErrorContains(t, err, "key must be exactly 32 bytes for AES-256")
	})

	t.Run("Success", func(t *testing.T) {
		tmp := filepath.Join(os.TempDir(), "testkey-success")
		keyData := make([]byte, 32)
		for i := range keyData {
			keyData[i] = byte(i)
		}
		require.NoError(t, os.WriteFile(tmp, keyData, 0600))
		defer os.Remove(tmp)
		os.Setenv("NT_VAULT_AES_KEY_FILE", tmp)

		key, err := aes.LoadKey()
		require.NoError(t, err)
		assert.Equal(t, keyData, key)
	})
}

func TestComputeHMAC(t *testing.T) {
	t.Run("ValidHMAC", func(t *testing.T) {
		data := []byte("test data")
		key := []byte("01234567890123456789012345678901") // 32 bytes key
		hmac1 := aes.ComputeHMAC(data, key)
		hmac2 := aes.ComputeHMAC(data, key)
		assert.Equal(t, hmac1, hmac2, "HMACs should match for the same data and key")
	})

	t.Run("DifferentKeys", func(t *testing.T) {
		data := []byte("test data")
		key1 := []byte("01234567890123456789012345678901") // 32 bytes key
		key2 := []byte("12345678901234567890123456789012") // Different 32 bytes key
		hmac1 := aes.ComputeHMAC(data, key1)
		hmac2 := aes.ComputeHMAC(data, key2)
		assert.NotEqual(t, hmac1, hmac2, "HMACs should differ for the same data with different keys")
	})

	t.Run("DifferentData", func(t *testing.T) {
		data1 := []byte("test data 1")
		data2 := []byte("test data 2")
		key := []byte("01234567890123456789012345678901") // 32 bytes key
		hmac1 := aes.ComputeHMAC(data1, key)
		hmac2 := aes.ComputeHMAC(data2, key)
		assert.NotEqual(t, hmac1, hmac2, "HMACs should differ for different data with the same key")
	})
}

func TestEncryption(t *testing.T) {
	key := []byte("01234567890123456789012345678901") // 32 bytes key

	t.Run("EncryptDecrypt", func(t *testing.T) {
		plaintext := []byte("This is a secret message.")
		ciphertext, err := aes.Encrypt(plaintext, key)
		require.NoError(t, err, "Encryption should not fail")

		decrypted, err := aes.Decrypt(ciphertext, key)
		require.NoError(t, err, "Decryption should not fail")
		assert.Equal(t, plaintext, decrypted, "Decrypted text should match the original plaintext")
	})

	t.Run("EncryptComputeHMAC", func(t *testing.T) {
		plaintext := []byte("This is a secret message.")
		ciphertext, err := aes.Encrypt(plaintext, key)
		require.NoError(t, err, "Encryption should not fail")

		hmac := aes.ComputeHMAC(ciphertext, key)
		assert.NotEmpty(t, hmac, "HMAC should not be empty")
	})
}
