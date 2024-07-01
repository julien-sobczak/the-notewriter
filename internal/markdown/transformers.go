package markdown

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/julien-sobczak/the-notewriter/pkg/text"
)

// Transformer applies changes on a Markdown document
type Transformer func(document Document) (Document, error)

// Transform applies transformers successively to create a new Markdown document
func (m Document) Transform(transformers ...Transformer) (Document, error) {
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
func (m Document) MustTransform(transformers ...Transformer) Document {
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
func ReplaceCharacters(characterReplacements map[string]string) Transformer {
	return func(document Document) (Document, error) {
		// TODO Reuse current code but in a more-robust way
		// - search only in texts (no code-block, no inside links/images/etc., line separator)
		// Implementation: We must not replace characters inside code blocks (otherwise, `i--` => `i—`)

		doc := string(document)

		var newLines []string

		lines := strings.Split(doc, "\n")
		insideSourceBlocks := false
		for _, line := range lines {
			if strings.HasPrefix(line, "    ") {
				newLines = append(newLines, line)
				continue
			}
			if strings.HasPrefix(line, "```") {
				insideSourceBlocks = !insideSourceBlocks
				newLines = append(newLines, line)
				continue
			}
			if strings.HasPrefix(line, "---") { // line separator
				newLines = append(newLines, line)
				continue
			}
			if insideSourceBlocks {
				newLines = append(newLines, line)
				continue
			}

			// Do not substitute inside `code` block
			parts := strings.Split(line, "`")
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
func StripHTMLComments() Transformer {
	return func(document Document) (Document, error) {
		md := string(document)
		r := regexp.MustCompile(`(?s)<!--.+?-->`)
		md = r.ReplaceAllString(md, "")
		return Document(md).TrimSpace(), nil
	}
}

// StripMarkdownUnofficialComments transforms a Markdown document to remove HTML-like, mostly-official Markdown comments
func StripMarkdownUnofficialComments() Transformer {
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
func AlignHeadings() Transformer {
	return func(document Document) (Document, error) {
		text := string(document)
		// Search for top subheading level
		minHeadingLevel := -1
		for _, line := range strings.Split(text, "\n") {
			ok, _, level := IsHeading(line)
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
			7: "#######",
		}
		for _, line := range strings.Split(text, "\n") {
			ok, headingTitle, level := IsHeading(line)
			if ok {
				newLevel := level - minHeadingLevel + 2 // The top sub-heading should be ## because # is reserved for the document title
				res.WriteString(levelHeading[newLevel])
				res.WriteString(" ")
				res.WriteString(headingTitle)
			} else {
				res.WriteString(line)
			}
			res.WriteString("\n")
		}
		return Document(res.String()), nil
	}
}

// StripCodeBlocks removes code blocks from a Markdown document.
func StripCodeBlocks() Transformer {
	return func(document Document) (Document, error) {
		var newLines []string

		insideCodeBlock := false
		iterator := document.Iterator()
		for iterator.HasNext() {
			line := iterator.Next()
			if strings.HasPrefix(line.Text, "```") { // Syntax 1
				insideCodeBlock = !insideCodeBlock
				newLines = append(newLines, "")
				continue
			}
			if strings.HasPrefix(line.Text, "    ") || insideCodeBlock { // Syntax 2
				newLines = append(newLines, "")
				continue
			}

			newLines = append(newLines, line.Text)
		}

		return Document(strings.Join(newLines, "\n")), nil
	}
}

// StripTopHeading remove the header
func StripTopHeading() Transformer {
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
func SquashBlankLines() Transformer {
	return func(document Document) (Document, error) {
		return Document(text.SquashBlankLines(string(document))), nil
	}
}

// StripEmphasis remove Markdown emphasis characters.
func StripEmphasis() Transformer {
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
