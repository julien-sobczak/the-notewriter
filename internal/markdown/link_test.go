package markdown_test

import (
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLink(t *testing.T) {

	t.Run("Internal", func(t *testing.T) {
		link := markdown.Link{
			Text: "External URL",
			URL:  "https://www.google.com",
		}
		assert.False(t, link.Internal())

		link = markdown.Link{
			Text: "Local file",
			URL:  "file.md",
		}
		assert.True(t, link.Internal())

		link = markdown.Link{
			Text: "Relative path",
			URL:  "./dir/file.md",
		}
		assert.True(t, link.Internal())

		link = markdown.Link{
			Text: "Absolute path",
			URL:  "/home/me/file.md",
		}
		assert.True(t, link.Internal())

		link = markdown.Link{
			Text: "File protocol",
			URL:  "file:///home/me/file.md",
		}
		assert.True(t, link.Internal())

		link = markdown.Link{
			Text: "S3 protocol",
			URL:  "s3://bucket/dir/file.md",
		}
		assert.False(t, link.Internal())
	})

}

func TestLinks(t *testing.T) {

	t.Run("Syntaxes", func(t *testing.T) {
		text := markdown.Document(`
[text](https://github.com)
[some text](./some-file.txt)
[text](file.md "Title")
[text](https://github.com#anchor "A long title")
			`)

		actual := text.Links()
		require.Len(t, actual, 4)

		expected := []markdown.Link{
			{
				Text:  "text",
				URL:   "https://github.com",
				Title: "",
				Line:  2,
			},
			{
				Text:  "some text",
				URL:   "./some-file.txt",
				Title: "",
				Line:  3,
			},
			{
				Text:  "text",
				URL:   "file.md",
				Title: "Title",
				Line:  4,
			},
			{
				Text:  "text",
				URL:   "https://github.com#anchor",
				Title: "A long title",
				Line:  5,
			},
		}
		assert.Equal(t, expected, actual)
	})

	t.Run("Example", func(t *testing.T) {
		filename := testutil.SetUpFromGoldenFileNamed(t, "TestMarkdown/links.md")
		md, err := markdown.ParseFile(filename)
		require.NoError(t, err)

		actual := md.Body.Links()
		require.Len(t, actual, 3)

		expected := []markdown.Link{
			{
				Text:  "Links",
				URL:   "./links.md#links",
				Title: "",
				Line:  5,
			},
			{
				Text:  "an optional title",
				URL:   "https://www.markdownguide.org/basic-syntax/#adding-titles",
				Title: "Adding titles",
				Line:  7,
			},
			{
				Text:  "documentation",
				URL:   "https://www.markdownguide.org/basic-syntax/#links",
				Title: "",
				Line:  9,
			},
		}
		assert.Equal(t, expected, actual)
	})

}
