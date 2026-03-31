package anki

import (
	"testing"

	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"github.com/stretchr/testify/assert"
)

func TestHtmlToMarkdown(t *testing.T) {

	t.Run("Basic HTML", func(t *testing.T) {
		tests := []struct {
			name          string
			html          string
			expectedText  string
			expectedMedia []string
		}{
			{
				name:          "empty string",
				html:          "",
				expectedText:  "",
				expectedMedia: []string{},
			},
			{
				name:          "plain text",
				html:          "plain text",
				expectedText:  "plain text",
				expectedMedia: []string{},
			},
			{
				name:          "bold with b tag",
				html:          "<b>bold text</b>",
				expectedText:  "**bold text**",
				expectedMedia: []string{},
			},
			{
				name:          "bold with strong tag",
				html:          "<strong>bold text</strong>",
				expectedText:  "**bold text**",
				expectedMedia: []string{},
			},
			{
				name:          "italic with i tag",
				html:          "<i>italic text</i>",
				expectedText:  "_italic text_",
				expectedMedia: []string{},
			},
			{
				name:          "italic with em tag",
				html:          "<em>italic text</em>",
				expectedText:  "_italic text_",
				expectedMedia: []string{},
			},
			{
				name:          "mixed formatting",
				html:          "<b>bold</b> and <i>italic</i>",
				expectedText:  "**bold** and _italic_",
				expectedMedia: []string{},
			},
			{
				name:          "img tag with src",
				html:          `<img src="image.jpg">`,
				expectedText:  `![image.jpg](./image.jpg)`,
				expectedMedia: []string{"image.jpg"},
			},
			{
				name:          "img tag with quotes",
				html:          `<img src='image.png'>`,
				expectedText:  `![image.png](./image.png)`,
				expectedMedia: []string{"image.png"},
			},
			{
				name:          "sound tag",
				html:          `[sound:audio.mp3]`,
				expectedText:  `![audio.mp3](./audio.mp3)`,
				expectedMedia: []string{"audio.mp3"},
			},
			{
				name:          "multiple sounds",
				html:          `[sound:file1.mp3] and [sound:file2.mp3]`,
				expectedText:  `![file1.mp3](./file1.mp3) and ![file2.mp3](./file2.mp3)`,
				expectedMedia: []string{"file1.mp3", "file2.mp3"},
			},
			{
				name:          "combined formatting and media",
				html:          `<b>Bold</b> with <img src="pic.jpg"> and [sound:note.mp3]`,
				expectedText:  `**Bold** with ![pic.jpg](./pic.jpg) and ![note.mp3](./note.mp3)`,
				expectedMedia: []string{"pic.jpg", "note.mp3"},
			},
			{
				name:          "nested tags",
				html:          `<b><i>bold italic</i></b>`,
				expectedText:  `**_bold italic_**`,
				expectedMedia: []string{},
			},
			{
				name:          "img with extra attributes",
				html:          `<img class="image" src="test.gif" alt="test">`,
				expectedText:  `![test.gif](./test.gif)`,
				expectedMedia: []string{"test.gif"},
			},
		}

		importer := &Importer{MediaDir: ""}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				text, media := importer.htmlToMarkdown(tt.html)
				assert.Equal(t, tt.expectedText, text)
				assert.Equal(t, tt.expectedMedia, media)
			})
		}
	})

	t.Run("With Media Dir", func(t *testing.T) {
		importer := &Importer{MediaDir: "assets"}
		text, media := importer.htmlToMarkdown(`<img src="photo.jpg">`)
		expectedText := `![photo.jpg](./assets/photo.jpg)`
		expectedMedia := []string{"photo.jpg"}

		assert.Equal(t, expectedText, text)
		assert.Equal(t, expectedMedia, media)
	})

	t.Run("Real-World HTML", func(t *testing.T) {
		tests := []struct {
			name     string
			html     string
			markdown string
		}{
			{
				name: "Markdown Emphasis",
				html: "(System Design) What is the <b>latency</b>? What is the <b>throughput</b>?<div><br></div><div><i>An assembly line is manufacturing cars. It takes eight hours to manufacture a car and that the factory produces one hundred and twenty cars per day.</i></div>",
				markdown: `(System Design) What is the **latency**? What is the **throughput**?

_An assembly line is manufacturing cars. It takes eight hours to manufacture a car and that the factory produces one hundred and twenty cars per day._`,
			},

			{
				name: "Markdown Codeblocks",
				html: `(Go)&nbsp;<b>Compile</b>?<pre><code>func (p Point) Distance(q Point) float64 { ... }

p := Point{1, 2}
q := Point{4, 6}
fmt.Println(Distance(p, q))
fmt.Println(p.Distance(q))</code></pre>`,
				markdown: text.UnescapeTestContent(`(Go) **Compile**?
‛‛‛
func (p Point) Distance(q Point) float64 { ... }

p := Point{1, 2}
q := Point{4, 6}
fmt.Println(Distance(p, q))
fmt.Println(p.Distance(q))
‛‛‛`),
			},
		}

		importer := &Importer{MediaDir: ""}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				text, _ := importer.htmlToMarkdown(tt.html)
				assert.Equal(t, tt.markdown, text)
			})
		}
	})
}
