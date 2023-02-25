package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdd(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		root := SetUpCollectionFromGoldenDirNamed(t, "TestFileSave")

		err := CurrentCollection().Add("go.md")
		require.NoError(t, err)

		data, err := os.ReadFile(filepath.Join(root, ".nt/index"))
		require.NoError(t, err)
		t.Log(string(data)) // BUT index.objects must only be modified after a commit (otherwise, we cannot found back the parent commit OID to restore)
		t.FailNow()
	})

}
