package testutil

import (
	"testing"

	"github.com/julien-sobczak/the-notewriter/pkg/filesystem"
)

// FreezeFileInfoReader replaces the default FileInfoReader with a clock-based one for testing purposes.
func FreezeFileInfoReader(t *testing.T) {
	filesystem.OverrideFileInfoReader(filesystem.NewClockBasedFileInfoReader())
	t.Cleanup(filesystem.RestoreFileInfoReader)
}
