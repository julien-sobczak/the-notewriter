package markdown

import (
	"fmt"
	"strings"

	"github.com/gosimple/slug"
)

// Slug returns a slug from a list of raw Markdown input values that will be processed.
func Slug(values ...any) string {
	var parts []string
	for _, value := range values {
		switch v := value.(type) {
		case string:
			parts = append(parts, v)
		case Document:
			part := v.MustTransform(StripEmphasis())
			parts = append(parts, string(part))
		default:
			part := fmt.Sprintf("%s", v)
			parts = append(parts, part)
		}
	}
	return slug.Make(strings.Join(parts, " "))
}
