package markdown

import (
	"bytes"
	"strings"

	"github.com/julien-sobczak/the-notewriter/pkg/text"
)


// ExtractLines extracts a Markdown document
func (m Document) ExtractLines(start, end int) Document {
	return Document(text.ExtractLines(string(m), start, end))
}

// SplitByHorizontalRules splits a Markdown document into multiple documents
// using Markdown horizontal rule characters as separators.
func (m Document) SplitByHorizontalRules() []Document {
	// See https://www.markdownguide.org/basic-syntax/#horizontal-rules

	var results []Document

	var content bytes.Buffer

	m.Transform()

	iterator := m.Iterator()
	for iterator.HasNext() {
		line := iterator.Next()
		text := strings.TrimSpace(line.Text)

		// At least 3 identical characters (-, _, *)
		// Blank lines before and after the horizontal rule is strongly recommended for compatibility.
		if line.Prev().IsBlank() && line.Next().IsBlank() &&
			(strings.HasPrefix(text, "---") && strings.Trim(text, "-") == "") ||
			(strings.HasPrefix(text, "___") && strings.Trim(text, "_") == "") ||
			(strings.HasPrefix(text, "***") && strings.Trim(text, "*") == "") {

			results = append(results, Document(content.String()).TrimSpace())
			content.Reset()
			continue
		}

		content.WriteString(text)
		content.WriteString("\n")
	}

	// Check if current section is empty
	lastContent := content.String()
	if !text.IsBlank(lastContent) {
		results = append(results, Document(lastContent).TrimSpace())
	}

	return results
}

// CodeBlock represents a code block inside a Markdown document
type CodeBlock struct {
	Line     int
	Language string
	Source   string
}

// ExtractCodeBlocks extracts all code blocks present in a Markdown document
func (m Document) ExtractCodeBlocks() []*CodeBlock {
	var results []*CodeBlock

	insideCodeBlock := false
	var currentSource bytes.Buffer
	var currentLine int
	var currentLanguage string

	md := string(m)
	for i, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(line, "```") {
			if !insideCodeBlock {
				// start of code block
				currentLine = i + 1 // lines start at 1
				currentLanguage = strings.TrimPrefix(line, "```")
				index := strings.Index(currentLanguage, " ")
				if index > -1 {
					currentLanguage = currentLanguage[:index]
				}
				insideCodeBlock = true
			} else {
				// end of code block
				results = append(results, &CodeBlock{
					Line:     currentLine,
					Source:   currentSource.String(),
					Language: currentLanguage,
				})
				// Reset for next code block
				insideCodeBlock = false
				currentSource.Reset()
				currentLine = 0
				currentLanguage = ""
			}
		} else if insideCodeBlock {
			currentSource.WriteString(line)
			currentSource.WriteRune('\n')
		}
	}

	return results
}

// StripComment extracts the optional user comment from a note body.
func (m *Document) ExtractComment() (string, string) {
	body := string(m.TrimSpace())

	lines := strings.Split(body, "\n")
	if len(lines) == 0 {
		return "", ""
	}

	i := len(lines) - 1

	// No comment or simply end with a standard quote?
	if !strings.HasPrefix(lines[i], "> ") || strings.HasPrefix(lines[i], "> —") || strings.HasPrefix(lines[i], "> --") {
		return body, ""
	}

	// Rewind until start of comment
	for ; i > 0; i-- {
		if !strings.HasPrefix(lines[i], "> ") {
			break
		}
	}

	// A blank line must precede the comment and other non-blank lines must exists before
	if text.IsBlank(lines[i]) && i > 0 {
		content := text.ExtractLines(body, 1, i+1)
		comment := text.TrimLinePrefix(text.ExtractLines(body, i+2, -1), "> ")
		return strings.TrimSpace(content), strings.TrimSpace(comment)
	} else {
		return body, ""
	}
}

type Quote struct {
	Text        Document
	Attribution Document
}

// ExtractQuote extracts a quote from a note content (support basic and sugar syntax)
func (m Document) ExtractQuote() Quote {
	var quote bytes.Buffer
	var attribution string

	md := string(m)
	lines := strings.Split(strings.TrimSpace(md), "\n")
	for i, line := range lines {
		if text.IsBlank(line) {
			quote.WriteRune('\n')
		} else if strings.HasPrefix(line, "> ") {
			line = strings.TrimPrefix(line, "> ")
			hasAttributionPrefix := strings.HasPrefix(line, "—") || strings.HasPrefix(line, "--")
			isLastLine := i == len(lines)-1 || text.IsBlank(lines[i+1])
			if hasAttributionPrefix && isLastLine {
				attribution = strings.TrimPrefix(line, "—")
				attribution = strings.TrimPrefix(attribution, "--")
				attribution = strings.TrimSpace(attribution)
				break
			}
			quote.WriteString(line)
			quote.WriteRune('\n')
		} else {
			quote.WriteString(line)
			quote.WriteRune('\n')
		}
	}
	return Quote{
		Text:        Document(quote.String()),
		Attribution: Document(attribution),
	}
}
