package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoldenFile(t *testing.T) {
	content := GoldenFile(t)
	assert.Equal(t, "# TestGoldenFile\n\nHi!\n", string(content))
}

func TestGoldenFileNamed(t *testing.T) {
	content := GoldenFileNamed(t, "TestGoldenFileNamedWithAnotherName.md")
	assert.Equal(t, "# TestGoldenFileNamedWithAnotherName\n\nHello!\n", string(content))
}
