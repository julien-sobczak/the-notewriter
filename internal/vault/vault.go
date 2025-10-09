package vault

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
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	EncryptedVersion   = "1.0"
	EncryptedAlgorithm = "AES256"
)

type VaultHeader struct {
	Encrypted bool   `yaml:"encrypted"`
	Version   string `yaml:"version"`
	Algorithm string `yaml:"algorithm"`
	HMAC      string `yaml:"hmac"`
}

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

// EncryptFile encrypts a markdown file and returns the encrypted content
func EncryptFile(content []byte, key []byte) ([]byte, error) {
	// Encrypt the content
	encrypted, err := Encrypt(content, key)
	if err != nil {
		return nil, err
	}

	// Encode as base64
	encodedContent := base64.StdEncoding.EncodeToString(encrypted)

	// Compute HMAC
	hmacValue := ComputeHMAC(encrypted, key)

	// Create header
	header := VaultHeader{
		Encrypted: true,
		Version:   EncryptedVersion,
		Algorithm: EncryptedAlgorithm,
		HMAC:      hmacValue,
	}

	// Marshal header to YAML
	headerBytes, err := yaml.Marshal(&header)
	if err != nil {
		return nil, err
	}

	// Construct the encrypted file format
	var result strings.Builder
	result.WriteString("---\n")
	result.WriteString(string(headerBytes))
	result.WriteString("---\n\n")
	result.WriteString("```\n")
	result.WriteString(encodedContent)
	result.WriteString("\n```\n")

	return []byte(result.String()), nil
}

// DecryptFile decrypts an encrypted markdown file and returns the original content
func DecryptFile(content []byte, key []byte) ([]byte, error) {
	// Parse the file
	contentStr := string(content)

	// Extract frontmatter
	if !strings.HasPrefix(contentStr, "---\n") {
		return nil, errors.New("file does not have valid frontmatter")
	}

	parts := strings.SplitN(contentStr, "---\n", 3)
	if len(parts) < 3 {
		return nil, errors.New("invalid encrypted file format")
	}

	// Parse header
	var header VaultHeader
	if err := yaml.Unmarshal([]byte(parts[1]), &header); err != nil {
		return nil, fmt.Errorf("cannot parse header: %w", err)
	}

	// Verify it's encrypted
	if !header.Encrypted {
		return nil, errors.New("file is not encrypted")
	}

	// Verify version and algorithm
	if header.Version != EncryptedVersion {
		return nil, fmt.Errorf("unsupported version: %s", header.Version)
	}
	if header.Algorithm != EncryptedAlgorithm {
		return nil, fmt.Errorf("unsupported algorithm: %s", header.Algorithm)
	}

	// Extract encrypted content (between ``` markers)
	body := parts[2]
	startIdx := strings.Index(body, "```\n")
	if startIdx == -1 {
		return nil, errors.New("cannot find encrypted content block")
	}
	body = body[startIdx+4:]

	endIdx := strings.Index(body, "\n```")
	if endIdx == -1 {
		return nil, errors.New("cannot find end of encrypted content block")
	}
	encodedContent := strings.TrimSpace(body[:endIdx])

	// Decode base64
	encrypted, err := base64.StdEncoding.DecodeString(encodedContent)
	if err != nil {
		return nil, fmt.Errorf("cannot decode base64: %w", err)
	}

	// Verify HMAC
	computedHMAC := ComputeHMAC(encrypted, key)
	if computedHMAC != header.HMAC {
		return nil, errors.New("HMAC verification failed - file may have been tampered with")
	}

	// Decrypt
	plaintext, err := Decrypt(encrypted, key)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// IsEncrypted checks if a file is encrypted
func IsEncrypted(content []byte) bool {
	contentStr := string(content)
	if !strings.HasPrefix(contentStr, "---\n") {
		return false
	}

	parts := strings.SplitN(contentStr, "---\n", 3)
	if len(parts) < 2 {
		return false
	}

	var header VaultHeader
	if err := yaml.Unmarshal([]byte(parts[1]), &header); err != nil {
		return false
	}

	return header.Encrypted
}
