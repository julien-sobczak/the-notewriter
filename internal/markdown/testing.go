package markdown

import "github.com/julien-sobczak/the-notewriter/pkg/text"

// UnescapeTestDocument wraps text.UnescapeTestContent.
func UnescapeTestDocument(md string) Document {
	return Document(text.UnescapeTestContent(md))
}
