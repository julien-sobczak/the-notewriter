package markdown_test

import (
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/stretchr/testify/assert"
)

func TestIsHeading(t *testing.T) {
	ok, _, _ := markdown.IsHeading("Some text")
	assert.False(t, ok)

	ok, _, _ = markdown.IsHeading("")
	assert.False(t, ok)

	ok, title, level := markdown.IsHeading("# Heading 1")
	assert.True(t, ok)
	assert.Equal(t, "Heading 1", title)
	assert.Equal(t, 1, level)

	ok, title, level = markdown.IsHeading("## Heading 2")
	assert.True(t, ok)
	assert.Equal(t, "Heading 2", title)
	assert.Equal(t, 2, level)

	ok, title, level = markdown.IsHeading("### Heading 3")
	assert.True(t, ok)
	assert.Equal(t, "Heading 3", title)
	assert.Equal(t, 3, level)

	ok, title, level = markdown.IsHeading("#### Heading 4")
	assert.True(t, ok)
	assert.Equal(t, "Heading 4", title)
	assert.Equal(t, 4, level)

	ok, title, level = markdown.IsHeading("##### Heading 5")
	assert.True(t, ok)
	assert.Equal(t, "Heading 5", title)
	assert.Equal(t, 5, level)

	ok, title, level = markdown.IsHeading("###### Heading 6")
	assert.True(t, ok)
	assert.Equal(t, "Heading 6", title)
	assert.Equal(t, 6, level)

	// Sub levels are not currently supported
	ok, _, _ = markdown.IsHeading("####### Heading 7")
	assert.False(t, ok)
}
