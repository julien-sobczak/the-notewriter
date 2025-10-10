package e2e_test

import (
	"os"
	"testing"

	. "github.com/julien-sobczak/the-notewriter/internal/core" // Required to import testing utilities
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVault(t *testing.T) {
	tr := NewTestRepository(t,
		WithFreezeNow(),
		WithFile("security.md", `
---
tags: secure
---

# Security

## Note: AES

The Advanced Encryption Standard (AES) is a symmetric block cipher chosen by the U.S. government to protect classified information
`))

	// File with secure tag must be encrypted first
	_, err := CurrentRepository().Lint(AnyPath, nil)
	require.ErrorContains(t, err, "secure but unencrypted")

	os.Create(tr.AbsolutePath("security.md."))
	mdFile := tr.ParseMarkdown("security.md")

	// Retry after encrypting
	err = mdFile.Encrypt()
	require.NoError(t, err)
	_, err = CurrentRepository().Add(AnyPath)
	require.NoError(t, err)

	// Encrypted file must be detected as such
	mdFile, err = markdown.ParseFileRaw(tr.AbsolutePath("security.md"))
	require.NoError(t, err)
	assert.True(t, mdFile.Encrypted())
	// But can be decrypted
	err = mdFile.Decrypt()
	require.NoError(t, err)
	content := tr.ReadFile("security.md")
	assert.Contains(t, content, "The Advanced Encryption Standard (AES) is a symmetric block cipher")

	// Commit the changes
	err = CurrentRepository().Commit()
	require.NoError(t, err)
}
