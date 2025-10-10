package markdown_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/stretchr/testify/require"
)

func TestMarkdownEncryption(t *testing.T) {
	tr := core.NewTestRepository(t)
	tr.WriteFile("path/to/insecure.md", `
# nt-vault

## Note: How to

‛‛‛shell
$ nt-vault create path/to/secure.md
$ nt-vault edit path/to/secure.md
$ nt-vault view path/to/secure.md
‛‛‛
`)
	mdInsecure, err := markdown.ParseFileRaw(filepath.Join(tr.Root, "path/to/insecure.md"))
	require.NoError(t, err)
	require.False(t, mdInsecure.Encrypted())

	f, err := os.Create(filepath.Join(tr.Root, "path/to/secure.md"))
	require.NoError(t, err)
	err = mdInsecure.EncryptTo(f)
	require.NoError(t, err)

	mdSecure, err := markdown.ParseFileRaw(filepath.Join(tr.Root, "path/to/secure.md"))
	require.NoError(t, err)
	require.True(t, mdSecure.Encrypted())

	content, err := mdSecure.DecryptRaw()
	require.NoError(t, err)
	require.Equal(t, mdInsecure.Content, content)
}
