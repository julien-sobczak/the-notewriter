package core

import (
	"errors"
	"strings"
	"text/scanner"
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
			result.Path = strings.TrimRight(strings.TrimLeft(s.TokenText(), `"`), `"`)

		case "#":
			// Tag
			tagNameToken := s.Scan()
			if tagNameToken == scanner.EOF {
				return nil, errors.New("unexpected EOF when a tag name was expected")
			}
			tag := s.TokenText()
			for {
				v := s.Peek()
				if v == scanner.EOF || v != '-' {
					break
				}
				s.Scan() // advance -
				tagNameToken := s.Scan()
				if tagNameToken == scanner.EOF {
					return nil, errors.New("unexpected EOF in the middle of a tag name")
				}
				tag += "-" + s.TokenText()
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

// MatchesParsed checks if a note matches the given query
func (q *Query) MatchesParsed(note *ParsedNote) bool {
	// Check type filter
	if len(q.Types) > 0 {
		found := false
		for _, t := range q.Types {
			if note.Type == t {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check tag filter
	if len(q.Tags) > 0 {
		for _, tag := range q.Tags {
			if !note.NoteTags.Includes(tag) {
				return false
			}
		}
	}

	// Check attribute filter
	for attrName, attrValue := range q.Attributes {
		noteAttrValue, exists := note.Attributes[attrName]
		if !exists || noteAttrValue != attrValue {
			return false
		}
	}

	// Check slug filter
	if q.Slug != "" && note.Slug != q.Slug {
		return false
	}

	// Check terms (search in title and body)
	if len(q.Terms) > 0 {
		searchText := strings.ToLower(note.Title.String() + " " + note.Body.String())
		for _, term := range q.Terms {
			if !strings.Contains(searchText, strings.ToLower(term)) {
				return false
			}
		}
	}

	return true
}
