package main

import (
	"bufio"
	"log"
	"regexp"
	"strings"
)

type Affirmation struct {
	Description string
}

type Prompt struct {
	Description string
	Attributes  map[string]string
	Tags        []string
}

func ParseAffirmations(md string) ([]*Affirmation, error) {
	var results []*Affirmation

	// Extract the affirmations by reading the Markdown document line by line
	scanner := bufio.NewScanner(strings.NewReader(md))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "* ") {
			// Not a list item
			continue
		}
		line = strings.TrimPrefix(line, "* ")
		results = append(results, &Affirmation{
			Description: line,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func ParsePrompts(md string) ([]*Prompt, error) {
	var results []*Prompt

	// Extract the prompts by reading the Markdown document line by line
	scanner := bufio.NewScanner(strings.NewReader(md))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "* ") {
			// Not a list item
			continue
		}
		line = strings.TrimPrefix(line, "* ")

		tags, attributes, description := extractAndStripTagsAndAttributes(line)

		results = append(results, &Prompt{
			Description: description,
			Attributes:  attributes,
			Tags:        tags,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func MustParsePrompts(md string) []*Prompt {
	prompts, err := ParsePrompts(md)
	if err != nil {
		log.Fatal(err)
	}
	return prompts
}

func MustParseAffirmations(md string) []*Affirmation {
	affirmations, err := ParseAffirmations(md)
	if err != nil {
		log.Fatal(err)
	}
	return affirmations
}

// TODO use Markdown.Document new abstraction instead
func extractAndStripTagsAndAttributes(text string) ([]string, map[string]string, string) {
	tagRe := regexp.MustCompile("`#(\\w+)`")
	attrRe := regexp.MustCompile("`@(\\w+):\\s*([^@#]+)`")

	tags := tagRe.FindAllStringSubmatch(text, -1)
	attributes := attrRe.FindAllStringSubmatch(text, -1)

	var extractedTags []string
	for _, tag := range tags {
		if len(tag) > 1 {
			extractedTags = append(extractedTags, tag[1])
		}
	}

	extractedAttributes := make(map[string]string)
	for _, attr := range attributes {
		if len(attr) > 2 {
			extractedAttributes[attr[1]] = strings.TrimSpace(attr[2])
		}
	}

	// Strip tags and attributes from the text
	strippedText := tagRe.ReplaceAllString(text, "")
	strippedText = attrRe.ReplaceAllString(strippedText, "")

	return extractedTags, extractedAttributes, strippedText
}
