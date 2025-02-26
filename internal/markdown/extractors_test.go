package markdown_test

import (
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractLines(t *testing.T) {
	doc := markdown.Document(`
# Python

A basic Python script:

    $ print("Hello)

`)

	// Extract all lines
	assert.Equal(t, doc, doc.ExtractLines(0, -1))

	// Extract the content
	content := markdown.Document(`
A basic Python script:

    $ print("Hello)

`)
	assert.Equal(t, content, doc.ExtractLines(3, -1))

	// Extract the title
	title := markdown.Document(`# Python`)
	assert.Equal(t, title, doc.ExtractLines(2, 2))
}

func TestSplitByHorizontalRules(t *testing.T) {
	tests := []struct {
		name     string
		body     markdown.Document   // input
		expected []markdown.Document // output
	}{
		{
			name: "No section",
			body: `
This is a first section

This is a second section`,
			expected: []markdown.Document{
				"This is a first section\n\nThis is a second section",
			},
		},
		{
			name: "---",
			body: `
This is a first section

---

This is a second section`,
			expected: []markdown.Document{
				"This is a first section",
				"This is a second section",
			},
		},
		{
			name: "***",
			body: `
This is a first section

***

This is a second section`,
			expected: []markdown.Document{
				"This is a first section",
				"This is a second section",
			},
		},
		{
			name: "___",
			body: `
This is a first section

___

This is a second section`,
			expected: []markdown.Document{
				"This is a first section",
				"This is a second section",
			},
		},
		{
			name: "----",
			body: `
This is a first section

----

This is a second section`,
			expected: []markdown.Document{
				"This is a first section",
				"This is a second section",
			},
		},
		{
			name: "Missing blank lines",
			body: `
This is a first section
----
This is a second section`,
			expected: []markdown.Document{
				"This is a first section\n----\nThis is a second section",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.body.SplitByHorizontalRules()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestExtractCodeBlocks(t *testing.T) {
	tests := []struct {
		name     string
		body     markdown.Document     // input
		expected []*markdown.CodeBlock // output
	}{
		{
			name: "No code block",
			body: `
# Title

A basic note without code blocks.
`,
			expected: nil,
		},

		{
			name: "Single code block",
			body: "" +
				"# Title\n" +
				"\n" +
				"```python\n" +
				"print('Hey')\n" +
				"```\n",
			expected: []*markdown.CodeBlock{
				{
					Line:     3,
					Language: "python",
					Source:   "print('Hey')\n",
				},
			},
		},

		{
			name: "Multiple code blocks",
			body: "" +
				"# Title\n" +
				"\n" +
				"A first script in Python:\n" +
				"```python\n" +
				"print('Hey')\n" +
				"```\n" +
				"\n" +
				"A second script in Go:\n" +
				"```go hightlight\n" +
				"func main() {\n" +
				"    fmt.Println(`Hey`)\n" +
				"}\n" +
				"```\n",
			expected: []*markdown.CodeBlock{
				{
					Line:     4,
					Language: "python",
					Source:   "print('Hey')\n",
				},
				{
					Line:     9,
					Language: "go",
					Source:   "func main() {\n    fmt.Println(`Hey`)\n}\n",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.body.ExtractCodeBlocks()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestExtractComment(t *testing.T) {
	tests := []struct {
		name    string
		body    string // input
		content string // output
		comment string // output
	}{
		{
			name: "A simple quote without comment",
			body: `
> This is just a quote
`,
			content: "> This is just a quote",
			comment: "",
		},
		{
			name: "A simple quote with a comment",
			body: `
> This is just a quote

> A personal comment about this quote
`,
			content: "> This is just a quote",
			comment: "A personal comment about this quote",
		},
		{
			name: "A note with a comment",
			body: `
This is just a note

> A personal comment about this note
`,
			content: "This is just a note",
			comment: "A personal comment about this note",
		},
		{
			name: "A multiline comment",
			body: `
This is just a note

> A personal comment
> about this note
`,
			content: "This is just a note",
			comment: "A personal comment\nabout this note",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := markdown.Document(tt.body)
			content, comment := md.ExtractComment()
			assert.Equal(t, tt.content, string(content))
			assert.Equal(t, tt.comment, string(comment))
		})
	}
}

func TestExtractQuote(t *testing.T) {
	tests := []struct {
		name        string
		body        markdown.Document // input
		quote       markdown.Document // output
		attribution markdown.Document // output
	}{
		{
			name:        "No attribution",
			body:        "> A basic quote",
			quote:       "A basic quote\n",
			attribution: "",
		},
		{
			name:        "Multiline quote",
			body:        "> A basic quote\n> on two lines.",
			quote:       "A basic quote\non two lines.\n",
			attribution: "",
		},
		{
			name:        "With attribution using en-dash",
			body:        "> A basic quote\n> — Me",
			quote:       "A basic quote\n",
			attribution: "Me",
		},
		{
			name:        "With attribution using --",
			body:        "> A basic quote\n> -- Me",
			quote:       "A basic quote\n",
			attribution: "Me",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quote := tt.body.ExtractQuote()
			assert.Equal(t, tt.quote, quote.Text)
			assert.Equal(t, tt.attribution, quote.Attribution)
		})
	}
}

func TestExtractImages(t *testing.T) {
	tests := []struct {
		name     string
		body     markdown.Document // input
		expected markdown.Images   // output
	}{
		{
			name: "No images",
			body: `
# Title

A basic note without images.
`,
			expected: nil,
		},
		{
			name: "Single image",
			body: `
# Title

![Alt text](https://example.com/image.jpg "Title")
`,
			expected: []markdown.Image{
				{
					Text:  "Alt text",
					URL:   "https://example.com/image.jpg",
					Title: "Title",
					Line:  4,
				},
			},
		},
		{
			name: "Multiple images",
			body: `
# Title

![Alt text 1](https://example.com/image1.jpg "Title 1")

Some text in between.

![Alt text 2](https://example.com/image2.jpg "Title 2")
`,
			expected: []markdown.Image{
				{
					Text:  "Alt text 1",
					URL:   "https://example.com/image1.jpg",
					Title: "Title 1",
					Line:  4,
				},
				{
					Text:  "Alt text 2",
					URL:   "https://example.com/image2.jpg",
					Title: "Title 2",
					Line:  8,
				},
			},
		},
		{
			name: "Image without title",
			body: `
# Title

![Alt text](https://example.com/image.jpg)
`,
			expected: []markdown.Image{
				{
					Text:  "Alt text",
					URL:   "https://example.com/image.jpg",
					Title: "",
					Line:  4,
				},
			},
		},
		{
			name: "Image inside code block",
			body: markdown.UnescapeTestDocument(`
# Title

‛‛‛
![Alt text](https://example.com/image.jpg "Title")
‛‛‛
`),
			expected: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.body.ExtractImages()
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestImage(t *testing.T) {

	t.Run("Internal/External", func(t *testing.T) {
		tests := []struct {
			name     string
			image    markdown.Image // input
			internal bool           // output
		}{
			{
				name:     "External image",
				image:    markdown.Image{URL: "https://example.com/image.jpg"},
				internal: false,
			},
			{
				name:     "Internal image",
				image:    markdown.Image{URL: "/images/image.jpg"},
				internal: true,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.internal, tt.image.Internal())
				assert.Equal(t, !tt.internal, tt.image.External())
			})
		}
	})

}

func TestImageTransformers(t *testing.T) {

	t.Run("ResolveAbsoluteURL", func(t *testing.T) {
		tests := []struct {
			name             string
			image            markdown.Image // input
			absoluteFilePath string         // input
			expected         markdown.Image // output
		}{
			{
				name: "External links",
				image: markdown.Image{
					URL: "https://example.com/image.jpg",
				},
				absoluteFilePath: "/some/path/to/markdown.md",
				expected: markdown.Image{
					URL: "https://example.com/image.jpg",
				},
			},
			{
				name: "Absolute path",
				image: markdown.Image{
					URL: "/images/image.jpg",
				},
				absoluteFilePath: "/some/path/to/markdown.md",
				expected: markdown.Image{
					URL: "/images/image.jpg",
				},
			},
			{
				name: "Relative path",
				image: markdown.Image{
					URL: "../images/image.jpg",
				},
				absoluteFilePath: "/some/path/to/markdown.md",
				expected: markdown.Image{
					URL: "/some/path/images/image.jpg",
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				actual, err := tt.image.Transform(markdown.ResolveAbsoluteURL(tt.absoluteFilePath))
				require.NoError(t, err)
				assert.Equal(t, tt.expected, actual)
			})
		}
	})

	t.Run("ResolveRelativeURL", func(t *testing.T) {
		tests := []struct {
			name     string
			image    markdown.Image // input
			rootPath string         // input
			expected markdown.Image // output
		}{
			{
				name: "External links",
				image: markdown.Image{
					URL: "https://example.com/image.jpg",
				},
				rootPath: "/some/path/",
				expected: markdown.Image{
					URL: "https://example.com/image.jpg",
				},
			},
			{
				name: "Absolute path",
				image: markdown.Image{
					URL: "/some/path/images/image.jpg",
				},
				rootPath: "/some/path/",
				expected: markdown.Image{
					URL: "images/image.jpg",
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				actual, err := tt.image.Transform(markdown.ResolveRelativeURL(tt.rootPath))
				require.NoError(t, err)
				assert.Equal(t, tt.expected, actual)
			})
		}
	})

}

func TestImages(t *testing.T) {

	t.Run("URLs", func(t *testing.T) {
		tests := []struct {
			name     string
			images   markdown.Images // input
			expected []string        // output
		}{
			{
				name:     "No images",
				images:   markdown.Images{},
				expected: nil,
			},
			{
				name: "Single image",
				images: markdown.Images{
					{URL: "https://example.com/image.jpg"},
				},
				expected: []string{"https://example.com/image.jpg"},
			},
			{
				name: "Multiple images",
				images: markdown.Images{
					{URL: "https://example.com/image1.jpg"},
					{URL: "https://example.com/image2.jpg"},
				},
				expected: []string{
					"https://example.com/image1.jpg",
					"https://example.com/image2.jpg",
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				actual := tt.images.URLs()
				assert.Equal(t, tt.expected, actual)
			})
		}
	})
}
