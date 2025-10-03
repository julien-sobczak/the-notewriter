package markdown_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParseMarkdown(t *testing.T) {
	testcases := []struct {
		name   string
		golden string
		test   func(*testing.T, *markdown.File)
	}{
		{
			name:   "Basic",
			golden: "basic",
			test: func(t *testing.T, md *markdown.File) {
				// No front matter is defined
				assert.Empty(t, md.FrontMatter)
				fmMap, err := md.FrontMatter.AsMap()
				require.NoError(t, err)
				assert.Empty(t, fmMap)
				fmNode, err := md.FrontMatter.AsNode()
				require.NoError(t, err)
				assert.Empty(t, fmNode)

				// Body contains the original file content
				content, err := os.ReadFile(md.AbsolutePath)
				require.NoError(t, err)
				assert.Equal(t, markdown.Document(content), md.Body)

				// Check last update date
				updatedAt := md.LastUpdateDate()
				os.WriteFile(md.AbsolutePath, []byte("# Updated"), 0644)
				updatedMD, err := markdown.ParseFile(md.AbsolutePath)
				require.NoError(t, err)
				assert.True(t, updatedAt.Before(updatedMD.LastUpdateDate()))

				// Check sections
				sections, err := md.GetSections()
				require.NoError(t, err)

				sectionTitle := sections[0]
				assert.Equal(t, &markdown.Section{
					Parent:        nil,
					HeadingText:   "Title",
					HeadingLevel:  1,
					ContentText:   "# Title\n\n## Subtitle\n\n### Section A\n\nText from section A\n\n#### Section A.1\n\nText from section A1\n\n### Section B\n\nText from section B\n\n#### Section B.1\n\nText from section B1\n\n#### Section B.2\n\nText from section B2",
					FileLineStart: 1,
					FileLineEnd:   23,
					BodyLineStart: 1,
					BodyLineEnd:   23,
				}, trimSubSections(sectionTitle))

				sectionSubtitle := sections[1]
				assert.Equal(t, &markdown.Section{
					Parent:        sectionTitle,
					HeadingText:   "Subtitle",
					HeadingLevel:  2,
					ContentText:   "## Subtitle\n\n### Section A\n\nText from section A\n\n#### Section A.1\n\nText from section A1\n\n### Section B\n\nText from section B\n\n#### Section B.1\n\nText from section B1\n\n#### Section B.2\n\nText from section B2",
					FileLineStart: 3,
					FileLineEnd:   23,
					BodyLineStart: 3,
					BodyLineEnd:   23,
				}, trimSubSections(sectionSubtitle))

				sectionA := sections[2]
				assert.Equal(t, &markdown.Section{
					Parent:        sectionSubtitle,
					HeadingText:   "Section A",
					HeadingLevel:  3,
					ContentText:   "### Section A\n\nText from section A\n\n#### Section A.1\n\nText from section A1",
					FileLineStart: 5,
					FileLineEnd:   11,
					BodyLineStart: 5,
					BodyLineEnd:   11,
				}, trimSubSections(sectionA))

				sectionA1 := sections[3]
				assert.Equal(t, &markdown.Section{
					Parent:        sectionA,
					HeadingText:   "Section A.1",
					HeadingLevel:  4,
					ContentText:   "#### Section A.1\n\nText from section A1",
					FileLineStart: 9,
					FileLineEnd:   11,
					BodyLineStart: 9,
					BodyLineEnd:   11,
				}, trimSubSections(sectionA1))

				sectionB := sections[4]
				assert.Equal(t, &markdown.Section{
					Parent:        sectionSubtitle,
					HeadingText:   "Section B",
					HeadingLevel:  3,
					ContentText:   "### Section B\n\nText from section B\n\n#### Section B.1\n\nText from section B1\n\n#### Section B.2\n\nText from section B2",
					FileLineStart: 13,
					FileLineEnd:   23,
					BodyLineStart: 13,
					BodyLineEnd:   23,
				}, trimSubSections(sectionB))

				sectionB1 := sections[5]
				assert.Equal(t, &markdown.Section{
					Parent:        sectionB,
					HeadingText:   "Section B.1",
					HeadingLevel:  4,
					ContentText:   "#### Section B.1\n\nText from section B1",
					FileLineStart: 17,
					FileLineEnd:   19,
					BodyLineStart: 17,
					BodyLineEnd:   19,
				}, trimSubSections(sectionB1))

				sectionB2 := sections[6]
				assert.Equal(t, &markdown.Section{
					Parent:        sectionB,
					HeadingText:   "Section B.2",
					HeadingLevel:  4,
					ContentText:   "#### Section B.2\n\nText from section B2",
					FileLineStart: 21,
					FileLineEnd:   23,
					BodyLineStart: 21,
					BodyLineEnd:   23,
				}, trimSubSections(sectionB2))

				// Now check walking the sections
				err = md.WalkSections(func(parent *markdown.Section, current *markdown.Section, children []*markdown.Section) error {

					if current.HeadingText == "Title" {
						assert.Nil(t, parent)
						assert.ElementsMatch(t, []*markdown.Section{sectionSubtitle}, children)
					}

					if current.HeadingText == "Subtitle" {
						assert.Equal(t, sectionTitle, parent)
						assert.ElementsMatch(t, []*markdown.Section{sectionA, sectionB}, children)
					}

					if current.HeadingText == "Section A" {
						assert.Equal(t, sectionSubtitle, parent)
						assert.ElementsMatch(t, []*markdown.Section{sectionA1}, children)
					}

					if current.HeadingText == "Section A.1" {
						assert.Equal(t, sectionA, parent)
						assert.Empty(t, children)
					}

					if current.HeadingText == "Section B" {
						assert.Equal(t, sectionSubtitle, parent)
						assert.ElementsMatch(t, []*markdown.Section{sectionB1, sectionB2}, children)
					}

					if current.HeadingText == "Section B.1" {
						assert.Equal(t, sectionB, parent)
						assert.Empty(t, children)
					}

					if current.HeadingText == "Section B.2" {
						assert.Equal(t, sectionB, parent)
						assert.Empty(t, children)
					}

					return nil
				})
				require.NoError(t, err)
			},
		},

		{
			name:   "Front Matter",
			golden: "front-matter",
			test: func(t *testing.T, md *markdown.File) {
				assert.Equal(t, markdown.FrontMatter("# A comment\ntitle: Title\ntags: [tag1, tag2]\nrating: 3\nlinks:\n- https://github.com\n"), md.FrontMatter)
				fmMap, err := md.FrontMatter.AsMap()
				require.NoError(t, err)
				assert.Equal(t, map[string]any{
					"title":  "Title",
					"tags":   []interface{}{"tag1", "tag2"},
					"rating": 3,
					"links":  []interface{}{"https://github.com"},
				}, fmMap)
				fmNode, err := md.FrontMatter.AsNode()
				require.NoError(t, err)
				expectedMap := &yaml.Node{
					Kind: yaml.MappingNode,
					Tag:  "!!map",
					Content: []*yaml.Node{
						{
							Kind:        yaml.ScalarNode,
							Tag:         "!!str",
							Value:       "title",
							HeadComment: "# A comment",
							Line:        2,
							Column:      1,
						},
						{
							Kind:   yaml.ScalarNode,
							Tag:    "!!str",
							Value:  "Title",
							Line:   2,
							Column: 8,
						},
						{
							Kind:   yaml.ScalarNode,
							Tag:    "!!str",
							Value:  "tags",
							Line:   3,
							Column: 1,
						},
						{
							Kind:  yaml.SequenceNode,
							Style: yaml.FlowStyle,
							Tag:   "!!seq",
							Content: []*yaml.Node{
								{
									Kind:   yaml.ScalarNode,
									Tag:    "!!str",
									Value:  "tag1",
									Line:   3,
									Column: 8,
								},
								{
									Kind:   yaml.ScalarNode,
									Tag:    "!!str",
									Value:  "tag2",
									Line:   3,
									Column: 14,
								},
							},
							Line:   3,
							Column: 7,
						},
						{
							Kind:   yaml.ScalarNode,
							Tag:    "!!str",
							Value:  "rating",
							Line:   4,
							Column: 1,
						},
						{
							Kind:   yaml.ScalarNode,
							Tag:    "!!int",
							Value:  "3",
							Line:   4,
							Column: 9,
						},
						{
							Kind:   yaml.ScalarNode,
							Tag:    "!!str",
							Value:  "links",
							Line:   5,
							Column: 1,
						},
						{
							Kind: yaml.SequenceNode,
							Tag:  "!!seq",
							Content: []*yaml.Node{
								{
									Kind:   yaml.ScalarNode,
									Tag:    "!!str",
									Value:  "https://github.com",
									Line:   6,
									Column: 3,
								},
							},
							Line:   6,
							Column: 1,
						},
					},
					Line:   2,
					Column: 1,
				}
				assert.Equal(t, expectedMap, fmNode)

				// Body and File line numbers differ when a Front Matter is defined
				// 1. Check body on file
				assert.Equal(t, 10, md.BodyLine)
				assert.True(t, strings.HasPrefix(string(md.Body), "# Title")) // Must start with the first non-empty line
				// 2. Check sections
				sections, err := md.GetSections()
				require.NoError(t, err)
				sectionTitle := sections[0]
				assert.Equal(t, 10, sectionTitle.FileLineStart)
				assert.Equal(t, 16, sectionTitle.FileLineEnd)
				assert.Equal(t, 1, sectionTitle.BodyLineStart)
				assert.Equal(t, 7, sectionTitle.BodyLineEnd)
			},
		},

		{
			name:   "Code Blocks",
			golden: "code-block",
			test: func(t *testing.T, md *markdown.File) {
				sections, err := md.GetSections()
				require.NoError(t, err)
				assert.Len(t, sections, 3)

				sectionTitle := sections[0]

				sectionFenced := sections[1]
				assert.Equal(t, &markdown.Section{
					Parent:        sectionTitle,
					HeadingText:   "Fenced Code Block",
					HeadingLevel:  2,
					ContentText:   "## Fenced Code Block\n\n```md\n# Heading inside a block code\n\nSome text\n```",
					FileLineStart: 3,
					FileLineEnd:   9,
					BodyLineStart: 3,
					BodyLineEnd:   9,
				}, trimSubSections(sectionFenced))

				sectionIndent := sections[2]
				assert.Equal(t, &markdown.Section{
					Parent:        sectionTitle,
					HeadingText:   "Indented Code Block",
					HeadingLevel:  2,
					ContentText:   "## Indented Code Block\n\n    # Heading inside a block code\n\n    Some text",
					FileLineStart: 11,
					FileLineEnd:   15,
					BodyLineStart: 11,
					BodyLineEnd:   15,
				}, trimSubSections(sectionIndent))
			},
		},

		// Add more test cases here to enrich Markdown support
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			tr := core.NewTestRepository(t, core.FromGoldenFileNamed(t, "TestMarkdown/"+testcase.golden+".md"))
			md, err := markdown.ParseFile(filepath.Join(tr.Root, testcase.golden+".md"))
			require.NoError(t, err)
			testcase.test(t, md)
		})
	}
}

func TestFile(t *testing.T) {

	t.Run("GetSections", func(t *testing.T) {
		tr := core.NewTestRepository(t)

		tr.WriteFile("valid-hierarchy.md", `
# Title

## Section 1

Text from section 1

### Section 1.1

Text from section 1.1

## Section 2

Text from section 2
`)
		md, err := markdown.ParseFile(filepath.Join(tr.Root, "valid-hierarchy.md"))
		require.NoError(t, err)
		sections, err := md.GetSections()
		require.NoError(t, err)
		require.Len(t, sections, 4)

		// Check sections
		assert.Equal(t, "Title", sections[0].HeadingText.String())
		assert.Equal(t, "Section 1", sections[1].HeadingText.String())
		assert.Equal(t, "Section 1.1", sections[2].HeadingText.String())
		assert.Equal(t, "Section 2", sections[3].HeadingText.String())

		title := sections[0]
		section1 := sections[1]
		section1_1 := sections[2]
		section2 := sections[3]

		assert.Nil(t, title.Parent)
		assert.Equal(t, title, section1.Parent)
		assert.Equal(t, section1, section1_1.Parent)
		assert.Equal(t, title, section2.Parent)

		assert.Equal(t, []*markdown.Section{section1, section2}, title.Subsections)
		assert.Equal(t, []*markdown.Section{section1_1}, section1.Subsections)
		assert.Empty(t, section1_1.Subsections)
		assert.Empty(t, section2.Subsections)
	})
}

/* Test Helpers */

// trimSubSections returns a copy of the section with its children removed (for easier comparison in tests)
func trimSubSections(section *markdown.Section) *markdown.Section {
	if section == nil {
		return nil
	}
	// Duplicate the struct but remove children
	trimmed := *section
	trimmed.Subsections = nil
	return &trimmed
}
