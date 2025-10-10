package markdown

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/julien-sobczak/the-notewriter/pkg/aes"
	"gopkg.in/yaml.v3"
)

const (
	EncryptedVersion   = "1.0"
	EncryptedAlgorithm = "AES256"
)

// VaultFrontMatter represents the frontmatter of an encrypted file.
type VaultFrontMatter struct {
	Encrypted bool   `yaml:"encrypted"`
	Version   string `yaml:"version"`
	Algorithm string `yaml:"algorithm"`
	HMAC      string `yaml:"hmac"`
}

/* File methods */

func (m *File) Encrypted() bool {
	return m.FrontMatter.MatchesRegex(`(?m)^\s*encrypted:\s*true\s*$`)
}

func (f *File) EncryptRaw() ([]byte, error) {
	if f.Encrypted() {
		// Already encrypted
		return nil, fmt.Errorf("file %s is already encrypted", f.AbsolutePath)
	}

	// Encrypt
	key, err := aes.LoadKey()
	if err != nil {
		return nil, fmt.Errorf("error loading key: %v", err)
	}

	// Encrypt the content
	encrypted, err := aes.Encrypt(f.Content, key)
	if err != nil {
		return nil, err
	}

	// Encode as base64
	encodedContent := base64.StdEncoding.EncodeToString(encrypted)

	// Compute HMAC
	hmacValue := aes.ComputeHMAC(encrypted, key)

	// Construct the encrypted file format
	frontMatter := VaultFrontMatter{
		Encrypted: true,
		Version:   EncryptedVersion,
		Algorithm: EncryptedAlgorithm, // Only AES256 supported for now
		HMAC:      hmacValue,
	}
	frontMatterBytes, err := yaml.Marshal(&frontMatter)
	if err != nil {
		return nil, err
	}
	var result strings.Builder
	result.WriteString("---\n")
	result.WriteString(string(frontMatterBytes))
	result.WriteString("---\n\n")
	result.WriteString("```\n")
	result.WriteString(encodedContent)
	result.WriteString("\n```\n")

	return []byte(result.String()), nil
}

func (f *File) Encrypt() error {
	ff, err := os.Create(f.AbsolutePath)
	if err != nil {
		return err
	}
	if err := f.EncryptTo(ff); err != nil {
		return err
	}
	return nil
}

func (f *File) EncryptTo(w io.Writer) error {
	if f.Encrypted() {
		// Already encrypted
		return fmt.Errorf("file %s is already encrypted", f.AbsolutePath)
	}

	// Encrypt
	encrypted, err := f.EncryptRaw()
	if err != nil {
		return err
	}

	// Save
	if _, err := w.Write(encrypted); err != nil {
		return fmt.Errorf("error writing encrypted content: %v", err)
	}

	return nil
}

func (f *File) Decrypt() error {
	if !f.Encrypted() {
		return fmt.Errorf("file %s is not encrypted", f.AbsolutePath)
	}

	ff, err := os.Create(f.AbsolutePath)
	if err != nil {
		return err
	}
	if err := f.DecryptTo(ff); err != nil {
		return err
	}
	return nil
}

func (f *File) DecryptTo(w io.Writer) error {
	if !f.Encrypted() {
		return fmt.Errorf("file %s is not encrypted", f.AbsolutePath)
	}

	// Decrypt
	decrypted, err := f.DecryptRaw()
	if err != nil {
		return err
	}

	// Save
	if _, err := w.Write(decrypted); err != nil {
		return fmt.Errorf("error writing decrypted content: %v", err)
	}

	return nil
}

func (f *File) DecryptToMarkdown() (*File, error) {
	if !f.Encrypted() {
		return nil, fmt.Errorf("file %s is not encrypted", f.AbsolutePath)
	}

	// Decrypt
	decrypted, err := f.DecryptRaw()
	if err != nil {
		return nil, err
	}

	// Parse decrypted content as Markdown
	decryptedFile, err := ParseRaw(strings.NewReader(string(decrypted)), FileInfo{
		Path:  f.AbsolutePath,
		MTime: f.MTime,
		Size:  f.Size,
	})
	if err != nil {
		return nil, err
	}

	return decryptedFile, nil
}

func (f *File) DecryptRaw() ([]byte, error) {
	if !f.Encrypted() {
		// Not encrypted
		return nil, fmt.Errorf("file %s is not encrypted", f.AbsolutePath)
	}

	// Decrypt
	key, err := aes.LoadKey()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading key: %v\n", err)
		os.Exit(1)
	}

	var frontMatter VaultFrontMatter
	if err := f.FrontMatter.Unmarshal(&frontMatter); err != nil {
		return nil, fmt.Errorf("cannot parse front matter: %w", err)
	}

	// Verify version and algorithm
	if frontMatter.Version != EncryptedVersion {
		return nil, fmt.Errorf("unsupported version: %s", frontMatter.Version)
	}
	if frontMatter.Algorithm != EncryptedAlgorithm {
		return nil, fmt.Errorf("unsupported algorithm: %s", frontMatter.Algorithm)
	}

	// Extract encrypted content (between ``` markers)
	codeBlocks := f.Body.ExtractCodeBlocks()
	if len(codeBlocks) == 0 {
		return nil, errors.New("cannot find encrypted content block")
	}
	encodedContent := strings.TrimSpace(codeBlocks[0].Source)

	// Decode base64
	encrypted, err := base64.StdEncoding.DecodeString(encodedContent)
	if err != nil {
		return nil, fmt.Errorf("cannot decode base64: %w", err)
	}

	// Verify HMAC
	computedHMAC := aes.ComputeHMAC(encrypted, key)
	if computedHMAC != frontMatter.HMAC {
		return nil, errors.New("HMAC verification failed - file may have been tampered with")
	}

	// Decrypt
	plaintext, err := aes.Decrypt(encrypted, key)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}
