package core

import (
	"errors"
	"strings"
	"text/scanner"
)

type Query struct {
	Slug       string
	Kinds      []string
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

		case "kind":
			// Kind
			colonToken := s.Scan()
			if colonToken == scanner.EOF {
				return nil, errors.New("unexpected EOF when : was expected")
			}

			kindToken := s.Scan()
			if kindToken == scanner.EOF {
				return nil, errors.New("unexpected EOF when a kind value was expected")
			}
			result.Kinds = append(result.Kinds, s.TokenText())

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
