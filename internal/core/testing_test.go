package core

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetUpCollectionFromGoldenDirNamed(t *testing.T) {
	dirname := SetUpCollectionFromGoldenDirNamed(t, "example")
	require.FileExists(t, filepath.Join(dirname, "thoughts/on-notetaking.md"))
}
