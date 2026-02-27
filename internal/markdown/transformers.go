package markdown

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/julien-sobczak/the-notewriter/pkg/text"
)

// DocumentTransformer applies changes on a Markdown document
type DocumentTransformer func(document Document) (Document, error)

// Transform applies transformers successively to create a new Markdown document
func (m Document) Transform(transformers ...DocumentTransformer) (Document, error) {
	result := m
	for _, transformer := range transformers {
		resultTransformed, err := transformer(result)
		if err != nil {
			return m, err
		}
		result = resultTransformed
	}
	return result, nil
}

// MustTransform is similar to Transform but does not expect an error
func (m Document) MustTransform(transformers ...DocumentTransformer) Document {
	result, err := m.Transform(transformers...)
	if err != nil {
		panic(err)
	}
	return result
}

/*
 * Transformers
 */

// See https://docs.asciidoctor.org/asciidoc/latest/subs/replacements/
var AsciidocCharacterSubstitutions = map[string]string{
	"(C)":  "©",
	"(R)":  "®",
	"(TM)": "™",
	"--":   "—",
	"...":  "…",
	"->":   "→",
	"=>":   "⇒",
	"<-":   "←",
	"<=":   "⇐",
}

// ReplaceCharacters is a Markdown transformer to replace character sequences inside a document.
func ReplaceCharacters(characterReplacements map[string]string) DocumentTransformer {
	return func(document Document) (Document, error) {
		// Implementation: We must not replace characters inside code blocks (otherwise, `i--` => `i—`)

		var newLines []string

		lines := document.Lines()
		for _, line := range lines {
			if line.InsideCodeBlock {
				newLines = append(newLines, line.Text)
				continue
			}
			if line.IsHorizontalRule() {
				newLines = append(newLines, line.Text)
				continue
			}

			// Do not substitute inside `code` block
			parts := strings.Split(line.Text, "`")
			var newParts []string
			for i, part := range parts {
				if i%2 == 0 {
					for character, replacement := range characterReplacements {
						part = strings.ReplaceAll(part, character, replacement)
					}
				}
				newParts = append(newParts, part)
			}
			newLines = append(newLines, strings.Join(newParts, "`"))
		}

		newDoc := Document(strings.Join(newLines, "\n"))
		return newDoc, nil
	}
}

// StripHTMLComments transforms a Markdown document to remove HTML comments
func StripHTMLComments() DocumentTransformer {
	return func(document Document) (Document, error) {
		md := string(document)
		r := regexp.MustCompile(`(?s)<!--.+?-->`)
		md = r.ReplaceAllString(md, "")
		return Document(md).TrimSpace(), nil
	}
}

// StripMarkdownUnofficialComments transforms a Markdown document to remove HTML-like, mostly-official Markdown comments
func StripMarkdownUnofficialComments() DocumentTransformer {
	return func(document Document) (Document, error) {
		md := string(document)
		r := regexp.MustCompile(`(?s)<!---.+?--->`)
		md = r.ReplaceAllString(md, "")
		return Document(md).TrimSpace(), nil
	}
}

// AlignHeadings unindents headings.
//
// Ex: (### Note: Blabla)
//
//	blabla
//	#### Blablabla
//	blablabla
//	##### Blablablabla
//	blablablabla
//
// Becomes:
//
//	blabla
//	## Blablabla
//	blablabla
//	### Blablablabla
//	blablablabla
func AlignHeadings() DocumentTransformer {
	return func(document Document) (Document, error) {
		// Search for the highest heading level in the document
		minHeadingLevel := -1
		it := document.Iterator()
		for it.HasNext() {
			line := it.Next()
			ok, _, level := line.IsHeading()
			if ok {
				if minHeadingLevel == -1 || level < minHeadingLevel {
					minHeadingLevel = level
				}
			}
		}

		if minHeadingLevel == -1 { // no heading found = nothing to do
			return document, nil
		}

		// Up level to simulate a standalone Markdown document
		var res bytes.Buffer
		levelHeading := map[int]string{
			1: "#",
			2: "##",
			3: "###",
			4: "####",
			5: "#####",
			6: "######",
		}
		it = document.Iterator()
		for it.HasNext() {
			line := it.Next()
			ok, headingTitle, level := line.IsHeading()
			if ok {
				newLevel := level - minHeadingLevel + 2 // The top sub-heading should be ## because # is reserved for the document title
				res.WriteString(levelHeading[newLevel])
				res.WriteString(" ")
				res.WriteString(headingTitle)
				it.SkipHeading() // skip next line if alternate heading
			} else {
				res.WriteString(line.Text)
			}
			res.WriteString("\n")
		}
		return Document(res.String()), nil
	}
}

// ShiftHeadings shifts all heading levels in a document by the given amount.
// A positive shift increases the heading level (e.g., # becomes ## when shift=1).
// Headings that would exceed level 6 are capped at level 6.
func ShiftHeadings(shift int) DocumentTransformer {
	return func(document Document) (Document, error) {
		if shift == 0 {
			return document, nil
		}
		var res bytes.Buffer
		it := document.Iterator()
		for it.HasNext() {
			line := it.Next()
			ok, headingTitle, level := line.IsHeading()
			if ok {
				newLevel := level + shift
				if newLevel < 1 {
					newLevel = 1
				}
				if newLevel > 6 {
					newLevel = 6
				}
				res.WriteString(strings.Repeat("#", newLevel))
				res.WriteString(" ")
				res.WriteString(headingTitle)
				it.SkipHeading()
			} else {
				res.WriteString(line.Text)
			}
			res.WriteString("\n")
		}
		return Document(res.String()), nil
	}
}

func StripCodeBlocks() DocumentTransformer {
	return func(document Document) (Document, error) {
		var newLines []string

		iterator := document.Iterator()
		for iterator.HasNext() {
			line := iterator.Next()
			if line.InsideCodeBlock {
				// Preserve line count to not break line numbers
				newLines = append(newLines, "")
			} else {
				newLines = append(newLines, line.Text)
			}
		}

		return Document(strings.Join(newLines, "\n")), nil
	}
}

// StripTopHeading remove the header
func StripTopHeading() DocumentTransformer {
	return func(document Document) (Document, error) {
		iterator := document.Iterator()
		iterator.SkipBlankLines()
		for iterator.HasNext() {
			line := iterator.Next()
			if strings.HasPrefix(line.Text, "#") {
				// Found the top heading => return what follows
				iterator.SkipBlankLines()
				if !iterator.HasNext() {
					return EmptyDocument, nil
				}
				line := iterator.Next()
				return document.ExtractLines(line.Number, -1), nil
			} else {
				// Found no top heading => return from here
				return document.ExtractLines(line.Number, -1), nil
			}
		}
		return EmptyDocument, nil
	}
}

// SquashBlankLines removes blank lines when multiple successive blank lines are present
func SquashBlankLines() DocumentTransformer {
	return func(document Document) (Document, error) {
		return Document(text.SquashBlankLines(string(document))), nil
	}
}

// StripEmphasis remove Markdown emphasis characters.
func StripEmphasis() DocumentTransformer {
	return func(document Document) (Document, error) {
		text := string(document)

		reBoldAsterisks := regexp.MustCompile(`\*\*(.*?)\*\*`)
		reBoldUnderscores := regexp.MustCompile(`__(.*?)__`)
		reItalicAsterisks := regexp.MustCompile(`\*(.*?)\*`)
		reItalicUnderscores := regexp.MustCompile(`_(.*?)_`)
		reCode := regexp.MustCompile("`([^`].*?)`") // Important: do not match ```

		text = reBoldAsterisks.ReplaceAllString(text, "$1")
		text = reBoldUnderscores.ReplaceAllString(text, "$1")
		text = reItalicAsterisks.ReplaceAllString(text, "$1")
		text = reItalicUnderscores.ReplaceAllString(text, "$1")
		text = reCode.ReplaceAllString(text, "$1")

		return Document(text), nil
	}
}
