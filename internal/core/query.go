package core

import (
	"errors"
	"strings"
	"text/scanner"
	"unicode"
)

type Query struct {
	Slug       string
	Types      []string
	Tags       []string
	Attributes map[string]interface{}
	Path       string
	Terms      []string
}

// NewQuery instantiates a new query.
func NewQuery() *Query {
	return &Query{
		Attributes: make(map[string]interface{}),
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
		switch s.TokenText() {

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
			result.Types = append(result.Types, s.TokenText())

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
				result.Path = strings.TrimRight(strings.TrimLeft(pathValue, `"`), `"`)
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
				result.Path = pathBuilder.String()
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
			result.Tags = append(result.Tags, tag)

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
			result.Attributes[attributeName] = strings.TrimRight(strings.TrimLeft(s.TokenText(), `"`), `"`)

		default:
			// Term
			term := strings.TrimRight(strings.TrimLeft(s.TokenText(), `"`), `"`)
			result.Terms = append(result.Terms, term)
		}
	}
}
