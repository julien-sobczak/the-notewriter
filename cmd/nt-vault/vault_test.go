package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	// Create a test key (32 bytes for AES-256)
	key := []byte("12345678901234567890123456789012")

	// Test data
	plaintext := []byte("This is a secret message")

	// Encrypt
	encrypted, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Verify encrypted data is different
	if string(encrypted) == string(plaintext) {
		t.Error("Encrypted data should be different from plaintext")
	}

	// Decrypt
	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	// Verify decrypted data matches original
	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted data doesn't match original. Got %q, want %q", string(decrypted), string(plaintext))
	}
}

func TestEncryptDecryptWithWrongKey(t *testing.T) {
	// Create test keys
	key1 := []byte("12345678901234567890123456789012")
	key2 := []byte("abcdefghijklmnopqrstuvwxyz123456")

	plaintext := []byte("Secret message")

	// Encrypt with key1
	encrypted, err := Encrypt(plaintext, key1)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Try to decrypt with key2 (should fail)
	_, err = Decrypt(encrypted, key2)
	if err == nil {
		t.Error("Decrypt should fail with wrong key")
	}
}

func TestComputeHMAC(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	data := []byte("test data")

	hmac1 := ComputeHMAC(data, key)
	hmac2 := ComputeHMAC(data, key)

	// HMAC should be consistent
	if hmac1 != hmac2 {
		t.Error("HMAC should be consistent for same data and key")
	}

	// Different data should produce different HMAC
	data2 := []byte("different data")
	hmac3 := ComputeHMAC(data2, key)
	if hmac1 == hmac3 {
		t.Error("Different data should produce different HMAC")
	}
}

func TestEncryptFile(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	content := []byte(`---
title: "Test Note"
---

# Test Note

Some content here.`)

	encrypted, err := EncryptFile(content, key)
	if err != nil {
		t.Fatalf("EncryptFile failed: %v", err)
	}

	// Verify the encrypted file has the correct structure
	encryptedStr := string(encrypted)
	if !IsEncrypted(encrypted) {
		t.Error("Encrypted file should be marked as encrypted")
	}

	// Check for required components
	if !contains(encryptedStr, "encrypted: true") {
		t.Error("Encrypted file missing 'encrypted: true'")
	}
	if !contains(encryptedStr, "version: \"1.0\"") {
		t.Error("Encrypted file missing version")
	}
	if !contains(encryptedStr, "algorithm: AES256") {
		t.Error("Encrypted file missing algorithm")
	}
	if !contains(encryptedStr, "hmac:") {
		t.Error("Encrypted file missing HMAC")
	}
	if !contains(encryptedStr, "```") {
		t.Error("Encrypted file missing code block markers")
	}
}

func TestDecryptFile(t *testing.T) {
	// Create a temporary key file
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "test.key")
	key := []byte("12345678901234567890123456789012")
	if err := os.WriteFile(keyFile, key, 0600); err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	os.Setenv("NT_VAULT_AES_KEY_FILE", keyFile)
	defer os.Unsetenv("NT_VAULT_AES_KEY_FILE")

	// Original content
	content := []byte(`---
title: "Secret"
---

# Secret Note

Password: secret123`)

	// Encrypt
	encrypted, err := EncryptFile(content, key)
	if err != nil {
		t.Fatalf("EncryptFile failed: %v", err)
	}

	// Decrypt
	decrypted, err := DecryptFile(encrypted)
	if err != nil {
		t.Fatalf("DecryptFile failed: %v", err)
	}

	// Verify content matches
	if string(decrypted) != string(content) {
		t.Errorf("Decrypted content doesn't match original.\nGot:\n%s\n\nWant:\n%s", string(decrypted), string(content))
	}
}

func TestIsEncrypted(t *testing.T) {
	key := []byte("12345678901234567890123456789012")

	// Plaintext file
	plaintext := []byte(`---
title: "Test"
---

# Test`)
	if IsEncrypted(plaintext) {
		t.Error("Plaintext should not be detected as encrypted")
	}

	// Encrypted file
	encrypted, err := EncryptFile(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptFile failed: %v", err)
	}
	if !IsEncrypted(encrypted) {
		t.Error("Encrypted file should be detected as encrypted")
	}
}

func TestLoadKeyWithWrongPermissions(t *testing.T) {
	// Create a temporary key file with wrong permissions
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "test.key")
	key := []byte("12345678901234567890123456789012")
	if err := os.WriteFile(keyFile, key, 0644); err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	os.Setenv("NT_VAULT_AES_KEY_FILE", keyFile)
	defer os.Unsetenv("NT_VAULT_AES_KEY_FILE")

	// Should fail due to wrong permissions
	_, err := LoadKey()
	if err == nil {
		t.Error("LoadKey should fail with wrong permissions")
	}
	if !contains(err.Error(), "too open permissions") {
		t.Errorf("Expected 'too open permissions' error, got: %v", err)
	}
}

func TestLoadKeyWithCorrectPermissions(t *testing.T) {
	// Test both 0600 and 0400
	testCases := []struct {
		name string
		perm os.FileMode
	}{
		{"0600", 0600},
		{"0400", 0400},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			perm := tc.perm
			tmpDir := t.TempDir()
			keyFile := filepath.Join(tmpDir, "test.key")
			key := []byte("12345678901234567890123456789012")
			if err := os.WriteFile(keyFile, key, perm); err != nil {
				t.Fatalf("Failed to create key file: %v", err)
			}
			os.Setenv("NT_VAULT_AES_KEY_FILE", keyFile)
			defer os.Unsetenv("NT_VAULT_AES_KEY_FILE")

			loadedKey, err := LoadKey()
			if err != nil {
				t.Errorf("LoadKey should succeed with %o permissions: %v", perm, err)
			}
			if string(loadedKey) != string(key) {
				t.Error("Loaded key doesn't match original")
			}
		})
	}
}

func TestLoadKeyWithWrongSize(t *testing.T) {
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "test.key")
	// Wrong size key (16 bytes instead of 32)
	key := []byte("1234567890123456")
	if err := os.WriteFile(keyFile, key, 0600); err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	os.Setenv("NT_VAULT_AES_KEY_FILE", keyFile)
	defer os.Unsetenv("NT_VAULT_AES_KEY_FILE")

	_, err := LoadKey()
	if err == nil {
		t.Error("LoadKey should fail with wrong key size")
	}
	if !contains(err.Error(), "32 bytes") {
		t.Errorf("Expected '32 bytes' error, got: %v", err)
	}
}

func TestGetEditor(t *testing.T) {
	// Save original EDITOR
	origEditor := os.Getenv("EDITOR")
	defer os.Setenv("EDITOR", origEditor)

	// Test default
	os.Unsetenv("EDITOR")
	if editor := GetEditor(); editor != "vi" {
		t.Errorf("Default editor should be 'vi', got %q", editor)
	}

	// Test custom
	os.Setenv("EDITOR", "nano")
	if editor := GetEditor(); editor != "nano" {
		t.Errorf("Editor should be 'nano', got %q", editor)
	}
}

func TestGetPager(t *testing.T) {
	// Save original PAGER
	origPager := os.Getenv("PAGER")
	defer os.Setenv("PAGER", origPager)

	// Test default
	os.Unsetenv("PAGER")
	if pager := GetPager(); pager != "less" {
		t.Errorf("Default pager should be 'less', got %q", pager)
	}

	// Test custom
	os.Setenv("PAGER", "more")
	if pager := GetPager(); pager != "more" {
		t.Errorf("Pager should be 'more', got %q", pager)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
