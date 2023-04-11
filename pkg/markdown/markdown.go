package markdown

import (
	"bytes"
	"strings"

	"github.com/julien-sobczak/the-notetaker/pkg/text"
)

func ToMarkdown(markdownText string) string {
	markdownText = AlignHeadings(markdownText)
	markdownText = text.SquashBlankLines(markdownText)

	return strings.TrimSpace(markdownText)
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
func AlignHeadings(text string) string {
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
		return text
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
			newLevel := level - minHeadingLevel + 2 // The top sub-heading should be ##
			res.WriteString(levelHeading[newLevel])
			res.WriteString(" ")
			res.WriteString(headingTitle)
		} else {
			res.WriteString(line)
		}
		res.WriteString("\n")
	}
	return res.String()
}

// IsHeading returns if a givne line is a Markdown heading and its level.
func IsHeading(line string) (bool, string, int) {
	if !strings.HasPrefix(line, "#") {
		return false, "", 0
	}
	if strings.HasPrefix(line, "###### ") {
		return true, strings.TrimPrefix(line, "###### "), 6
	} else if strings.HasPrefix(line, "##### ") {
		return true, strings.TrimPrefix(line, "##### "), 5
	} else if strings.HasPrefix(line, "#### ") {
		return true, strings.TrimPrefix(line, "#### "), 4
	} else if strings.HasPrefix(line, "### ") {
		return true, strings.TrimPrefix(line, "### "), 3
	} else if strings.HasPrefix(line, "## ") {
		return true, strings.TrimPrefix(line, "## "), 2
	} else if strings.HasPrefix(line, "# ") {
		return true, strings.TrimPrefix(line, "# "), 1
	}

	return false, "", 0
}

// ReplaceAsciidocCharacterSubstitutions replaces Asciidoc special character sequences by their equivalent in Unicode.
func ReplaceAsciidocCharacterSubstitutions(md string) string {
	// See https://docs.asciidoctor.org/asciidoc/latest/subs/replacements/

	// Implementation: We must not replace characters inside code blocks (otherwise, `i--` => `i—`)

	var newLines []string

	lines := strings.Split(md, "\n")
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
				part = strings.ReplaceAll(part, "(C)", "©")
				part = strings.ReplaceAll(part, "(R)", "®")
				part = strings.ReplaceAll(part, "(TM)", "™")
				part = strings.ReplaceAll(part, "--", "—")
				part = strings.ReplaceAll(part, "...", "…")
				part = strings.ReplaceAll(part, "->", "→")
				part = strings.ReplaceAll(part, "=>", "⇒")
				part = strings.ReplaceAll(part, "<-", "←")
				part = strings.ReplaceAll(part, "<=", "⇐")
			}
			newParts = append(newParts, part)
		}
		newLines = append(newLines, strings.Join(newParts, "`"))
	}

	return strings.Join(newLines, "\n")
}

// StripTopHeading remove the header
func StripTopHeading(md string) string {
	lines := strings.Split(md, "\n")
	i := 0

	// Skip leading blank lines
	for i < len(lines) && text.IsBlank(lines[i]) {
		i++
	}
	if i == len(lines) {
		// EOF
		return ""
	}

	// Check first non-empty line
	if strings.HasPrefix(lines[i], "#") {
		i++
		// Advance to next non-blank line
		for i < len(lines) && text.IsBlank(lines[i]) {
			i++
		}
		if i == len(lines) {
			// EOF
			return ""
		}
	}

	return strings.Join(lines[i:], "\n")
}

// StripComment extracts the optional user comment from a note body.
func StripComment(body string) (string, string) {
	body = strings.TrimSpace(body)

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

// ExtractQuote extracts a quote from a note content (support basic and sugar syntax)
func ExtractQuote(md string) (string, string) {
	var quote bytes.Buffer
	var attribution string
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
	return quote.String(), attribution
}
