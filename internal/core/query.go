package core

import (
	"errors"
	"strings"
	"text/scanner"
	"unicode"
)

// QueryCondition represents a single condition in a query with an optional operator.
// The operator is "=" by default (or when the prefix "+" is present) and "<>" when the prefix "-" is present.
type QueryCondition struct {
	Operator string // "=" (positive/default) or "<>" (negation with "-" prefix)
	Operand  string // The value to match
}

// IsNegated returns true if this condition is a negation (NOT) condition.
func (c QueryCondition) IsNegated() bool {
	return c.Operator == "<>"
}

type Query struct {
	Slug       string
	Types      []QueryCondition
	Tags       []QueryCondition
	Attributes map[string]QueryCondition
	Path       []QueryCondition
	Terms      []QueryCondition
}

// NewQuery instantiates a new query.
func NewQuery() *Query {
	return &Query{
		Attributes: make(map[string]QueryCondition),
	}
}

// ParseQuery parses a user query.
func ParseQuery(q string) (*Query, error) {
	result := NewQuery()

	var s scanner.Scanner
	s.Init(strings.NewReader(q))
	s.Filename = ""

	for {
		token := s.Scan()
		if token == scanner.EOF {
			return result, nil
		}

		// Check for operator prefix: "-" means negation ("<>"), "+" means positive ("=")
		operator := "="
		tokenText := s.TokenText()
		if tokenText == "-" || tokenText == "+" {
			if tokenText == "-" {
				operator = "<>"
			}
			token = s.Scan()
			if token == scanner.EOF {
				return nil, errors.New("unexpected EOF after operator prefix")
			}
			tokenText = s.TokenText()
		}

		switch tokenText {

		case "slug":
			// Slug
			colonToken := s.Scan()
			if colonToken == scanner.EOF {
				return nil, errors.New("unexpected EOF when : was expected")
			}
			slugValueToken := s.Scan()
			if slugValueToken == scanner.EOF {
				return nil, errors.New("unexpected EOF when slug value was expected")
			}
			slugToken := s.TokenText()
			for {
				v := s.Peek()
				if v == scanner.EOF || v != '-' {
					break
				}
				s.Scan() // advance -
				slugValueToken = s.Scan()
				if slugValueToken == scanner.EOF {
					return nil, errors.New("unexpected EOF in the middle of a slug")
				}
				slugToken += "-" + s.TokenText()
			}
			result.Slug = slugToken

		case "type":
			// Type
			colonToken := s.Scan()
			if colonToken == scanner.EOF {
				return nil, errors.New("unexpected EOF when : was expected")
			}

			typeToken := s.Scan()
			if typeToken == scanner.EOF {
				return nil, errors.New("unexpected EOF when a type value was expected")
			}
			result.Types = append(result.Types, QueryCondition{Operator: operator, Operand: s.TokenText()})

		case "path":
			// Path
			colonToken := s.Scan()
			if colonToken == scanner.EOF {
				return nil, errors.New("unexpected EOF when : was expected")
			}

			pathToken := s.Scan()
			if pathToken == scanner.EOF {
				return nil, errors.New("unexpected EOF when a path was expected")
			}

			pathValue := s.TokenText()

			// If the path starts with quotes, it's a quoted path - just remove the quotes
			if strings.HasPrefix(pathValue, `"`) {
				result.Path = append(result.Path, QueryCondition{
					Operator: operator,
					Operand:  strings.TrimRight(strings.TrimLeft(pathValue, `"`), `"`),
				})
			} else {
				// For unquoted paths, continue reading tokens to build the full path
				// This handles cases like: path:thoughts/on-learning.md
				var pathBuilder strings.Builder
				pathBuilder.WriteString(pathValue)

				for {
					nextRune := s.Peek()
					if nextRune == scanner.EOF {
						break
					}
					// Continue if it's a path separator, dot, hyphen, or alphanumeric
					if nextRune == '/' || nextRune == '-' || nextRune == '.' ||
						unicode.IsLetter(rune(nextRune)) || unicode.IsDigit(rune(nextRune)) {
						s.Scan() // consume the token
						pathBuilder.WriteString(s.TokenText())
					} else {
						break
					}
				}
				result.Path = append(result.Path, QueryCondition{Operator: operator, Operand: pathBuilder.String()})
			}

		case "#":
			// Tag
			tagNameToken := s.Scan()
			if tagNameToken == scanner.EOF {
				return nil, errors.New("unexpected EOF when a tag name was expected")
			}
			tag := s.TokenText()

			// Continue reading to build the full tag including slashes and hyphens
			// This handles cases like: #todo/read, #project/personal/goals
			for {
				nextRune := s.Peek()
				if nextRune == scanner.EOF {
					break
				}
				// Continue if it's a separator we want to include in tags or alphanumeric
				if nextRune == '/' || nextRune == '-' ||
					unicode.IsLetter(rune(nextRune)) || unicode.IsDigit(rune(nextRune)) {
					s.Scan() // consume the token
					tag += s.TokenText()
				} else {
					break
				}
			}
			result.Tags = append(result.Tags, QueryCondition{Operator: operator, Operand: tag})

		case "@":
			// Attribute
			attributeNameToken := s.Scan()
			if attributeNameToken == scanner.EOF {
				return nil, errors.New("unexpected EOF when a tag name was expected")
			}
			attributeName := s.TokenText()

			colonToken := s.Scan()
			if colonToken == scanner.EOF {
				return nil, errors.New("unexpected EOF when : was expected")
			}

			attributeValueToken := s.Scan()
			if attributeValueToken == scanner.EOF {
				return nil, errors.New("unexpected EOF when an attribute value was expected")
			}
			result.Attributes[attributeName] = QueryCondition{
				Operator: operator,
				Operand:  strings.TrimRight(strings.TrimLeft(s.TokenText(), `"`), `"`),
			}

		default:
			// Term
			term := strings.TrimRight(strings.TrimLeft(tokenText, `"`), `"`)
			result.Terms = append(result.Terms, QueryCondition{Operator: operator, Operand: term})
		}
	}
}

// MatchesParsed checks if a note matches the given query
func (q *Query) MatchesParsed(note *ParsedNote) bool {
	// Check type filter
	if len(q.Types) > 0 {
		// Positive types: note must match at least one positive type condition
		// Negative types: note must not match any negative type condition
		var hasPositive bool
		positiveMatch := false
		for _, cond := range q.Types {
			if cond.IsNegated() {
				if note.Type == cond.Operand {
					return false // explicitly excluded
				}
			} else {
				hasPositive = true
				if note.Type == cond.Operand {
					positiveMatch = true
				}
			}
		}
		if hasPositive && !positiveMatch {
			return false
		}
	}

	// Check tag filter
	if len(q.Tags) > 0 {
		for _, cond := range q.Tags {
			if cond.IsNegated() {
				if note.NoteTags.Includes(cond.Operand) {
					return false // explicitly excluded
				}
			} else {
				if !note.NoteTags.Includes(cond.Operand) {
					return false
				}
			}
		}
	}

	// Check attribute filter
	for attrName, cond := range q.Attributes {
		noteAttrValue, exists := note.Attributes[attrName]
		if cond.IsNegated() {
			if exists && noteAttrValue == cond.Operand {
				return false // explicitly excluded
			}
		} else {
			if !exists || noteAttrValue != cond.Operand {
				return false
			}
		}
	}

	// Check slug filter
	if q.Slug != "" && note.Slug != q.Slug {
		return false
	}

	// Check path filter
	if len(q.Path) > 0 {
		var hasPositive bool
		positiveMatch := false
		for _, cond := range q.Path {
			if cond.IsNegated() {
				if strings.HasPrefix(note.RelativePath, cond.Operand) {
					return false // explicitly excluded
				}
			} else {
				hasPositive = true
				if strings.HasPrefix(note.RelativePath, cond.Operand) {
					positiveMatch = true
				}
			}
		}
		if hasPositive && !positiveMatch {
			return false
		}
	}

	// Check terms (search in title and body)
	if len(q.Terms) > 0 {
		searchText := strings.ToLower(note.Title.String() + " " + note.Body.String())
		for _, cond := range q.Terms {
			if cond.IsNegated() {
				if strings.Contains(searchText, strings.ToLower(cond.Operand)) {
					return false // explicitly excluded
				}
			} else {
				if !strings.Contains(searchText, strings.ToLower(cond.Operand)) {
					return false
				}
			}
		}
	}

	return true
}
