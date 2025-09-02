package main

import (
	"fmt"
	"regexp"
	"strings"
)

// Placeholder represents a URL placeholder variable
type Placeholder struct {
	Name         string   // Variable name (e.g., "page")
	FullMatch    string   // Full placeholder text (e.g., "${page:[issues,pulls]}")
	AllowedValues []string // Specific allowed values, nil if any value allowed
	HasMore      bool     // True if "..." is present, indicating autocomplete suggestions
}

// parsePlaceholders extracts all placeholders from a URL
func parsePlaceholders(url string) []Placeholder {
	// Regex to match ${variable} or ${variable:[value1,value2,...]}
	re := regexp.MustCompile(`\$\{([^}:]+)(?::?\[([^\]]*)\])?\}`)
	matches := re.FindAllStringSubmatch(url, -1)
	
	var placeholders []Placeholder
	if len(matches) == 0 {
		return []Placeholder{} // Return empty slice instead of nil
	}
	
	for _, match := range matches {
		placeholder := Placeholder{
			Name:      match[1],
			FullMatch: match[0],
		}
		
		// Parse allowed values if present
		if len(match) > 2 && match[2] != "" {
			values := strings.Split(match[2], ",")
			for i, value := range values {
				value = strings.TrimSpace(value)
				if value == "..." {
					placeholder.HasMore = true
				} else {
					values[i] = value
				}
			}
			
			// Remove "..." from values list if present
			if placeholder.HasMore && len(values) > 0 && values[len(values)-1] == "..." {
				values = values[:len(values)-1]
			}
			
			placeholder.AllowedValues = values
		}
		
		placeholders = append(placeholders, placeholder)
	}
	
	return placeholders
}

// expandURL replaces placeholders in URL with provided values
func expandURL(url string, values map[string]string) string {
	result := url
	for name, value := range values {
		// Replace all occurrences of this placeholder
		re := regexp.MustCompile(`\$\{` + regexp.QuoteMeta(name) + `(?::\[[^\]]*\])?\}`)
		result = re.ReplaceAllString(result, value)
	}
	return result
}

// getPlaceholderType returns the type of input needed for a placeholder
func (p Placeholder) getPlaceholderType() string {
	if len(p.AllowedValues) > 0 && !p.HasMore {
		return "select"
	} else if len(p.AllowedValues) > 0 && p.HasMore {
		return "autocomplete"
	}
	return "input"
}

// String returns a human-readable description of the placeholder
func (p Placeholder) String() string {
	switch p.getPlaceholderType() {
	case "select":
		return fmt.Sprintf("%s (choose from: %s)", p.Name, strings.Join(p.AllowedValues, ", "))
	case "autocomplete":
		return fmt.Sprintf("%s (suggestions: %s, or enter custom value)", p.Name, strings.Join(p.AllowedValues, ", "))
	default:
		return fmt.Sprintf("%s (enter any value)", p.Name)
	}
}