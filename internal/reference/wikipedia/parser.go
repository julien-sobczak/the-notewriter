package wikipedia

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Infobox struct {
	Name       string
	Attributes map[string]any
}

func NewInfobox(name string) *Infobox {
	return &Infobox{
		Name:       name,
		Attributes: make(map[string]any),
	}
}

var regexBirthDate = regexp.MustCompile(`^(?i){{birth[- ]date\|`)
var regexBirthDateAndAge = regexp.MustCompile(`^(?i){{birth[- ](?:date|year)[- ]and[- ]age\|`)
var regexDeathDate = regexp.MustCompile(`^(?i){{death[- ]date\|`)
var regexDeathDateAndAge = regexp.MustCompile(`^(?i){{death[- ](?:date|year)[- ]and[- ]age\|`)
var regexDate = regexp.MustCompile(`^(?i)(\d{1,2}) (January|February|March|April|May|June|July|August|September|October|November|December) (\d{1,4})$`)

var months = map[string]string{
	"January":   "01",
	"February":  "02",
	"March":     "03",
	"April":     "04",
	"May":       "05",
	"June":      "06",
	"July":      "07",
	"August":    "08",
	"September": "09",
	"October":   "10",
	"November":  "11",
	"December":  "12",
}

func parseWikitext(txt string) *Infobox {
	infobox := NewInfobox("")
	r := regexp.MustCompile(`\{\{Infobox\s+(\w+)`)
	indices := r.FindAllStringIndex(txt, -1)
	for _, matchIndices := range indices {
		start := matchIndices[0]
		end := start + 5000 // read "enough" characters to find the closing }}
		if end > len(txt) {
			end = len(txt)
		}

		extract := txt[start:end]
		lines := strings.Split(strings.TrimSuffix(extract, "\n"), "\n")
		i := 1
		for {
			if i >= len(lines) {
				// eof
				break
			}
			line := strings.TrimSuffix(lines[i], "\n")

			if len(strings.TrimSpace(line)) == 0 {
				// blank line
				i++
				continue
			}
			if strings.HasPrefix(line, "}}") {
				// end of infobox
				break
			}
			if strings.HasPrefix(line, "| ") {
				// new attribute
				attributeRegex := regexp.MustCompile(`^\|\s+(\w+)\s*=\s*(.*?)\s*$`)
				res := attributeRegex.FindStringSubmatch(line)
				key, wikiValue := res[1], res[2]

				var parsedValue interface{}

				if strings.HasPrefix(wikiValue, "{{") && !strings.HasSuffix(wikiValue, "}}") {
					// Multiline value
					var sb strings.Builder
					sb.WriteString(wikiValue)
					sb.WriteString("\n")
					i++
					for i < len(lines) {
						line = lines[i]
						sb.WriteString(line)
						sb.WriteString("\n")
						i++
						if strings.HasPrefix(line, "}}") {
							break
						}
					}
					wikiValue = sb.String()
				}

				parsedValue = parseAttributeValue(wikiValue)
				if parsedValue != nil {
					infobox.Attributes[key] = parsedValue
				} else {
					fmt.Printf("Ignoring unknown syntax for atttribute %q: %s", key, wikiValue)
				}
			}
			i++
		}

	}
	return infobox
}

func parseAttributeValue(value string) interface{} {
	value = stripTextFormatting(value)
	value = stripLinks(value)
	value = stripHTMLEntities(value)
	value = stripHTMLComments(value)

	if strings.HasPrefix(value, "{{plainlist|") {
		return parsePlainlist(value)
	} else if strings.HasPrefix(value, "{{flatlist|") {
		return parseFlatlist(value)
	} else if regexBirthDate.MatchString(value) {
		return parseBirthDate(value)
	} else if regexBirthDateAndAge.MatchString(value) {
		return parseBirthDateAndAge(value)
	} else if regexDeathDate.MatchString(value) {
		return parseDeathDate(value)
	} else if regexDeathDateAndAge.MatchString(value) {
		return parseDeathDateAndAge(value)
	} else if regexDate.MatchString(value) {
		return parseDate(value)
	} else if strings.HasPrefix(value, "{{") {
		// Unsupported syntax
		return nil
	}

	return value
}

func parseDate(value string) string {
	res := regexDate.FindStringSubmatch(value)
	day, monthText, year := res[1], res[2], res[3]
	month := months[monthText]
	dayInt, err := strconv.Atoi(day)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s-%s-%02d", year, month, dayInt)
}

func stripTextFormatting(value string) string {
	// italic ''
	// bold '''
	// bold+italic '''''
	r := regexp.MustCompile(`'{2,5}(.*?)'{2,5}`)
	value = r.ReplaceAllString(value, "$1")
	return value
}

func parsePlainlist(value string) []interface{} {
	value = strings.TrimPrefix(value, "{{plainlist|\n")
	var items []interface{}
	for _, item := range strings.Split(value, "\n") {
		if strings.HasPrefix(item, "}}") {
			break
		}
		itemValue := parseAttributeValue(strings.TrimPrefix(item, "* "))
		if itemValue != nil {
			items = append(items, itemValue)
		}
	}
	return items
}

func parseFlatlist(value string) string {
	value = strings.TrimPrefix(value, "{{flatlist|\n")
	var items []string
	for _, item := range strings.Split(value, "\n") {
		if strings.HasPrefix(item, "}}") {
			break
		}
		itemValue := parseAttributeValue(strings.TrimPrefix(item, "* "))
		if itemValue != nil {
			items = append(items, fmt.Sprintf("%s", itemValue))
		}
	}
	return strings.Join(items, ", ")
}

func parseBirthDate(value string) string {
	// https://en.wikipedia.org/wiki/Template:Birth-date
	parameters := extractParametersFromValue(value, regexBirthDate)
	date := extractDateFromParameters(parameters)
	return date
}

func parseBirthDateAndAge(value string) string {
	// https://en.wikipedia.org/wiki/Template:Birth-date_and_age
	parameters := extractParametersFromValue(value, regexBirthDateAndAge)
	date := extractDateFromParameters(parameters)
	return date
}

func parseDeathDate(value string) string {
	// https://en.wikipedia.org/wiki/Template:Death-date
	parameters := extractParametersFromValue(value, regexDeathDate)
	date := extractDateFromParameters(parameters)
	return date
}

func parseDeathDateAndAge(value string) string {
	// https://en.wikipedia.org/wiki/Template:Death-date_and_age
	parameters := extractParametersFromValue(value, regexDeathDateAndAge)
	date := extractDateFromParameters(parameters)
	return date
}

// extractParametersFromValue returns the list of parameters.
// Ex: {{Death date|2016|12|31|df=y}} => []string{"2016", "12", "31", "df=y"}
func extractParametersFromValue(value string, r *regexp.Regexp) []string {
	parameters := strings.Split(strings.TrimSuffix(r.ReplaceAllString(value, ""), "}}"), "|")
	return parameters
}

func extractDateFromParameters(params []string) string {
	// Remove df=? parameter
	var filteredParams []string
	for _, param := range params {
		if strings.HasPrefix(param, "df=") {
			continue
		}
		filteredParams = append(filteredParams, param)
	}

	if len(filteredParams) == 0 {
		return ""
	}

	// Inspect the first parameter to determine the format
	regexMonthFirst := regexp.MustCompile(`^(\w+) (\d{1,2}), (\d{1,4})$`)     // Ex: February 24, 1993
	regexMonthYear := regexp.MustCompile(`^(\w+) (\d{1,4})$`)                 // Ex: September 1941
	regexDayFirst := regexp.MustCompile(`^(\d{1,2}) (\w+) (\d{1,4})$`)        // Ex: 24 February 1993
	regexYearOnly := regexp.MustCompile(`^\d{1,4}$`)                          // Ex: 1941
	regexISO := regexp.MustCompile(`^\d{1,4}(?:(?:-(\d{1,2}))?-(\d{1,2}))?$`) // Ex: 1941-10-13

	firstParam := filteredParams[0]
	secondParam := ""
	if len(filteredParams) >= 2 {
		secondParam = filteredParams[1]
	}
	thirdParam := ""
	if len(filteredParams) >= 3 {
		thirdParam = filteredParams[2]
	}

	if regexMonthFirst.MatchString(firstParam) {
		res := regexMonthFirst.FindStringSubmatch(firstParam)
		monthText, day, year := res[1], res[2], res[3]
		month := months[monthText]
		dayInt, err := strconv.Atoi(day)
		if err != nil {
			return ""
		}
		return fmt.Sprintf("%s-%s-%02d", year, month, dayInt)
	} else if regexMonthYear.MatchString(firstParam) {
		res := regexMonthYear.FindStringSubmatch(firstParam)
		monthText, year := res[1], res[2]
		month := months[monthText]
		return fmt.Sprintf("%s-%s", year, month)
	} else if regexDayFirst.MatchString(firstParam) {
		res := regexDayFirst.FindStringSubmatch(firstParam)
		day, monthText, year := res[1], res[2], res[3]
		month := months[monthText]
		dayInt, err := strconv.Atoi(day)
		if err != nil {
			return ""
		}
		return fmt.Sprintf("%s-%s-%02d", year, month, dayInt)
	} else if regexYearOnly.MatchString(firstParam) {
		date := firstParam
		if len(secondParam) > 0 {
			secondParamInt, err := strconv.Atoi(secondParam)
			if err != nil {
				return ""
			}
			date += fmt.Sprintf("-%02d", secondParamInt)
		}
		if len(thirdParam) > 0 {
			thirdParamInt, err := strconv.Atoi(thirdParam)
			if err != nil {
				return ""
			}
			date += fmt.Sprintf("-%02d", thirdParamInt)
		}
		return date
	} else if regexISO.MatchString(firstParam) {
		return firstParam
	}

	return ""
}

func stripLinks(value string) string {
	// Different kinds of links:
	// See https://www.mediawiki.org/wiki/Help:Links

	rPipeTrick := regexp.MustCompile(`\[\[(?:.*?:)?(.+?)\|\]\]`)
	rPipedLink := regexp.MustCompile(`\[\[.+?\|(.+?)\]\]`)
	rInternalLink := regexp.MustCompile(`\[\[(.+?)\]\]`)
	value = rPipeTrick.ReplaceAllString(value, "$1")
	value = rPipedLink.ReplaceAllString(value, "$1")
	value = rInternalLink.ReplaceAllString(value, "$1")
	return value
}

func stripHTMLEntities(value string) string {
	// See https://www.w3schools.com/html/html_entities.asp
	// Non-exhaustive list. Other characters are simply dropped
	value = strings.ReplaceAll(value, "&nbsp;", " ")  // non-breaking space
	value = strings.ReplaceAll(value, "&#160;", " ")  // non-breaking space
	value = strings.ReplaceAll(value, "&lt;", "<")    // less than
	value = strings.ReplaceAll(value, "&#60;", "<")   // less than
	value = strings.ReplaceAll(value, "&gt;", ">")    // greater than
	value = strings.ReplaceAll(value, "&#62;", ">")   // greater than
	value = strings.ReplaceAll(value, "&amp;", "&")   // ampersand
	value = strings.ReplaceAll(value, "&#38;", "&")   // ampersand
	value = strings.ReplaceAll(value, "&quot;", "\"") // double quotation mark
	value = strings.ReplaceAll(value, "&#34;", "\"")  // double quotation mark
	value = strings.ReplaceAll(value, "&apos;", "'")  // single quotation mark
	value = strings.ReplaceAll(value, "&#39;", "'")   // single quotation mark
	value = strings.ReplaceAll(value, "&cent;", "¢")  // cent
	value = strings.ReplaceAll(value, "&#162;", "¢")  // cent
	value = strings.ReplaceAll(value, "&pound;", "£") // pound
	value = strings.ReplaceAll(value, "&#163;", "£")  // pound
	value = strings.ReplaceAll(value, "&yen;", "¥")   // yen
	value = strings.ReplaceAll(value, "&#165;", "¥")  // yen
	value = strings.ReplaceAll(value, "&euro;", "€")  // euro
	value = strings.ReplaceAll(value, "&#8364;", "€") // euro
	value = strings.ReplaceAll(value, "&copy;", "©")  // copyright
	value = strings.ReplaceAll(value, "&#169;", "©")  // copyright
	value = strings.ReplaceAll(value, "&reg;", "®")   // registered trademark
	value = strings.ReplaceAll(value, "&#174;", "®")  // registered trademark

	// Remove unknown entites
	r := regexp.MustCompile(`&\S+;`)
	value = r.ReplaceAllString(value, "")

	return value
}

func stripHTMLComments(value string) string {
	r := regexp.MustCompile(`<!--.+?-->`)
	value = r.ReplaceAllString(value, "")
	return value
}
