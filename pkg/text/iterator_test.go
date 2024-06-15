package text_test

import (
	"testing"

	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"github.com/stretchr/testify/assert"
)

func TestLineIterator(t *testing.T) {

	t.Run("Basic", func(t *testing.T) {
		iterator := text.NewLineIteratorFromText("line 1\nline 2\n\nline 3\n")

		var line1, line2, line3, line4, line5 text.Line

		for iterator.HasNext() {
			line := iterator.Next()
			if line.Number == 1 {
				line1 = line
				assert.Equal(t, "line 1", line.Text)
				// Missing lines are considered like blank lines
				assert.Equal(t, text.MissingLine, line.Prev())
				assert.True(t, line.Prev().IsBlank())
				assert.False(t, line.IsBlank())
				assert.True(t, line.IsFirst())
				assert.False(t, line.IsLast())
			}
			if line.Number == 2 {
				line2 = line
				assert.Equal(t, "line 2", line.Text)
				assert.Equal(t, "line 2", line.Text)
				assert.False(t, line.IsBlank())
				assert.False(t, line.IsFirst())
				assert.False(t, line.IsLast())
			}
			if line.Number == 3 {
				line3 = line
				assert.Equal(t, "", line.Text)
				assert.True(t, line.IsBlank())
				assert.False(t, line.IsFirst())
				assert.False(t, line.IsLast())
			}
			if line.Number == 4 {
				line4 = line
				assert.Equal(t, "line 3", line.Text)
				assert.False(t, line.IsBlank())
				assert.False(t, line.IsFirst())
				assert.False(t, line.IsLast())
			}
			if line.Number == 5 {
				line5 = line
				assert.Equal(t, "", line.Text)
				assert.True(t, line.IsBlank())
				// Missing lines are considered like blank lines
				assert.Equal(t, text.MissingLine, line.Next())
				assert.True(t, line.Next().IsBlank())
				assert.False(t, line.IsFirst())
				assert.True(t, line.IsLast())
			}
		}

		// Check links between lines
		assert.Equal(t, line2, line1.Next())
		assert.Equal(t, line1, line2.Prev())
		assert.Equal(t, line3, line2.Next())
		assert.Equal(t, line2, line3.Prev())
		assert.Equal(t, line4, line3.Next())
		assert.Equal(t, line3, line4.Prev())
		assert.Equal(t, line5, line4.Next())
		assert.Equal(t, line4, line5.Prev())
	})

	t.Run("SkipBlankLinkes", func(t *testing.T) {
		md := "" +
			/* 1 */ "\n" +
			/* 2 */ "\n" +
			/* 3 */ "# Title\n" +
			/* 4 */ "\n" +
			/* 5 */ "Text\n"
		iterator := text.NewLineIteratorFromText(md)

		// Jump to next non-blank line
		iterator.SkipBlankLines()
		assert.True(t, iterator.HasNext())
		titleLine := iterator.Next()
		assert.False(t, titleLine.IsBlank())
		assert.Equal(t, "# Title", titleLine.Text)
		assert.Equal(t, 3, titleLine.Number)

		// Jump to next non-blank line
		iterator.SkipBlankLines()
		assert.True(t, iterator.HasNext())
		textLine := iterator.Next()
		assert.False(t, textLine.IsBlank())
		assert.Equal(t, "Text", textLine.Text)
		assert.Equal(t, 5, textLine.Number)

		// Jump to next non-blank line
		iterator.SkipBlankLines()
		assert.False(t, iterator.HasNext()) // end of doc
	})
}
