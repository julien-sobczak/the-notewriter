package markdown

import (
	"bytes"
	"path/filepath"
	"regexp"
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
func (m *Document) ExtractComment() (Document, Document) {
	body := m.TrimSpace()

	lines := body.Lines()
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
		content := body.ExtractLines(1, i+1)
		comment := Document(text.TrimLinePrefix(body.ExtractLines(i+2, -1).String(), "> "))
		return content.TrimSpace(), comment.TrimSpace()
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

// Image represents an image inside a Markdown document
type Image struct {
	Text  Document
	URL   string
	Title string
	Line  int
}

func (i Image) Internal() bool {
	return !i.External()
}

func (i Image) External() bool {
	schemaRegex := regexp.MustCompile(`^\w+://`)
	return schemaRegex.MatchString(i.URL)
}

// ImageTransformer applies changes on a Markdown image
type ImageTransformer func(image Image) (Image, error)

// Transform applies transformers successively to create a new Markdown document
func (i Image) Transform(transformers ...ImageTransformer) (Image, error) {
	result := i
	for _, transformer := range transformers {
		resultTransformed, err := transformer(result)
		if err != nil {
			return result, err
		}
		result = resultTransformed
	}
	return result, nil
}

// MustTransform is similar to Transform but does not expect an error
func (i Image) MustTransform(transformers ...ImageTransformer) Image {
	result, err := i.Transform(transformers...)
	if err != nil {
		panic(err)
	}
	return result
}

// ResolveAbsoluteURL resolves the raw URL if relative to absolute URLs
func ResolveAbsoluteURL(mdAbsolutePath string) ImageTransformer {
	return func(image Image) (Image, error) {
		if image.External() {
			// External links
			return image, nil

		}
		if strings.HasPrefix(image.URL, "/") {
			// Already absolute path
			return image, nil
		}

		// Ex: /some/path/to/markdown.md + ../index.md => /some/path/to/../markdown.md
		absolutePath, err := filepath.Abs(filepath.Join(filepath.Dir(mdAbsolutePath), image.URL))
		if err != nil {
			return image, err
		}

		image.URL = absolutePath
		return image, nil
	}
}

// ResolveRelativeURL resolves the raw URL if absolute to relative URLs
func ResolveRelativeURL(rootPath string) ImageTransformer {
	return func(image Image) (Image, error) {
		if image.External() {
			// External links
			return image, nil
		}

		relativePath, err := filepath.Rel(rootPath, image.URL)
		if err != nil {
			return image, err
		}

		image.URL = relativePath
		return image, nil
	}
}

type Images []Image

// Transform applies transformers successively to create new Markdown images
func (i Images) Transform(transformers ...ImageTransformer) (Images, error) {
	var results Images

	for _, image := range i {
		for _, transformer := range transformers {
			transformedImage, err := transformer(image)
			if err != nil {
				return nil, err
			}
			image = transformedImage
		}
		results = append(results, image)
	}

	return results, nil
}

// URLs returns the URLs of all images
func (i Images) URLs() []string {
	var urls []string
	for _, img := range i {
		urls = append(urls, img.URL)
	}
	return urls
}

// ExtractImages extracts all images present in a Markdown document
func (m Document) ExtractImages() Images {
	var results []Image

	// Ignore images inside code blocks (ex: a sample Markdown code block)
	doc := m.MustTransform(StripCodeBlocks())

	r := regexp.MustCompile(`!\[(.*?)\]\((\S*?)(?:\s+"(.*?)")?\)`)
	matches := r.FindAllStringSubmatch(string(doc), -1)
	for _, match := range matches {

		m := match[0]
		txt := match[1]
		url := match[2]
		title := match[3]

		line := text.LineNumber(string(doc), m)

		results = append(results, Image{
			Text:  Document(txt),
			URL:   url,
			Title: title,
			Line:  line,
		})
	}

	return results
}

// ExtractInternalImages extracts all intenral images present in a Markdown document
func (m Document) ExtractInternalImages() Images {
	var results []Image
	for _, img := range m.ExtractImages() {
		if img.Internal() {
			results = append(results, img)
		}
	}
	return results
}
