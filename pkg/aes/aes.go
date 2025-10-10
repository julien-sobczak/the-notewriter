package aes

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
)

// LoadKey loads the encryption key from the file specified in NT_VAULT_AES_KEY_FILE
func LoadKey() ([]byte, error) {
	keyFilePath := os.Getenv("NT_VAULT_AES_KEY_FILE")
	if keyFilePath == "" {
		return nil, errors.New("NT_VAULT_AES_KEY_FILE environment variable is not set")
	}

	// Check file permissions
	fileInfo, err := os.Stat(keyFilePath)
	if err != nil {
		return nil, fmt.Errorf("cannot stat key file: %w", err)
	}

	mode := fileInfo.Mode().Perm()
	// Must be 0600 or 0400
	if mode != 0600 && mode != 0400 {
		return nil, fmt.Errorf("key file has too open permissions: %o (must be 0600 or 0400)", mode)
	}

	// Read key file
	key, err := os.ReadFile(keyFilePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read key file: %w", err)
	}

	// Key must be 32 bytes for AES-256
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be exactly 32 bytes for AES-256, got %d bytes", len(key))
	}

	return key, nil
}

// Encrypt encrypts the given data using AES-256-GCM
func Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt and authenticate
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts the given data using AES-256-GCM
func Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// ComputeHMAC computes HMAC-SHA256 for the given data
func ComputeHMAC(data []byte, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
