package core_test

import (
	"os"
	"strings"
	"testing"

	"github.com/julien-sobczak/the-notewriter/internal/core"
	"github.com/julien-sobczak/the-notewriter/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestParseMarkdown(t *testing.T) {
	testcases := []struct {
		name   string
		golden string
		test   func(*testing.T, *core.MarkdownFile)
	}{
		{
			name:   "Basic",
			golden: "basic",
			test: func(t *testing.T, md *core.MarkdownFile) {
				// No front matter is defined
				assert.Empty(t, md.FrontMatter())
				fmMap, err := md.FrontMatterAsMap()
				require.NoError(t, err)
				assert.Empty(t, fmMap)
				fmNode, err := md.FrontMatterAsNode()
				require.NoError(t, err)
				assert.Empty(t, fmNode)

				// Body contains the original file content
				content, err := os.ReadFile(md.AbsolutePath())
				require.NoError(t, err)
				assert.Equal(t, string(content), md.Body())

				// Check last update date
				updatedAt := md.LastUpdateDate()
				os.WriteFile(md.AbsolutePath(), []byte("# Updated"), 0644)
				updatedMD, err := core.ParseMarkdownFile(md.AbsolutePath())
				require.NoError(t, err)
				assert.True(t, updatedAt.Before(updatedMD.LastUpdateDate()))

				// Check sections
				sections, err := md.GetSections()
				require.NoError(t, err)

				sectionTitle := sections[0]
				assert.Equal(t, &core.MarkdownSection{
					Parent:              nil,
					HeadingText:         "Title",
					HeadingLevel:        1,
					ContentText:         "# Title\n\n## Subtitle\n\n### Section A\n\nText from section A\n\n#### Section A.1\n\nText from section A1\n\n### Section B\n\nText from section B\n\n#### Section B.1\n\nText from section B1\n\n#### Section B.2\n\nText from section B2",
					FileLineNumberStart: 1,
					FileLineNumberEnd:   23,
					BodyLineNumberStart: 1,
					BodyLineNumberEnd:   23,
				}, sectionTitle)

				sectionSubtitle := sections[1]
				assert.Equal(t, &core.MarkdownSection{
					Parent:              sectionTitle,
					HeadingText:         "Subtitle",
					HeadingLevel:        2,
					ContentText:         "## Subtitle\n\n### Section A\n\nText from section A\n\n#### Section A.1\n\nText from section A1\n\n### Section B\n\nText from section B\n\n#### Section B.1\n\nText from section B1\n\n#### Section B.2\n\nText from section B2",
					FileLineNumberStart: 3,
					FileLineNumberEnd:   23,
					BodyLineNumberStart: 3,
					BodyLineNumberEnd:   23,
				}, sectionSubtitle)

				sectionA := sections[2]
				assert.Equal(t, &core.MarkdownSection{
					Parent:              sectionSubtitle,
					HeadingText:         "Section A",
					HeadingLevel:        3,
					ContentText:         "### Section A\n\nText from section A\n\n#### Section A.1\n\nText from section A1",
					FileLineNumberStart: 5,
					FileLineNumberEnd:   11,
					BodyLineNumberStart: 5,
					BodyLineNumberEnd:   11,
				}, sectionA)

				sectionA1 := sections[3]
				assert.Equal(t, &core.MarkdownSection{
					Parent:              sectionA,
					HeadingText:         "Section A.1",
					HeadingLevel:        4,
					ContentText:         "#### Section A.1\n\nText from section A1",
					FileLineNumberStart: 9,
					FileLineNumberEnd:   11,
					BodyLineNumberStart: 9,
					BodyLineNumberEnd:   11,
				}, sectionA1)

				sectionB := sections[4]
				assert.Equal(t, &core.MarkdownSection{
					Parent:              sectionSubtitle,
					HeadingText:         "Section B",
					HeadingLevel:        3,
					ContentText:         "### Section B\n\nText from section B\n\n#### Section B.1\n\nText from section B1\n\n#### Section B.2\n\nText from section B2",
					FileLineNumberStart: 13,
					FileLineNumberEnd:   23,
					BodyLineNumberStart: 13,
					BodyLineNumberEnd:   23,
				}, sectionB)

				sectionB1 := sections[5]
				assert.Equal(t, &core.MarkdownSection{
					Parent:              sectionB,
					HeadingText:         "Section B.1",
					HeadingLevel:        4,
					ContentText:         "#### Section B.1\n\nText from section B1",
					FileLineNumberStart: 17,
					FileLineNumberEnd:   19,
					BodyLineNumberStart: 17,
					BodyLineNumberEnd:   19,
				}, sectionB1)

				sectionB2 := sections[6]
				assert.Equal(t, &core.MarkdownSection{
					Parent:              sectionB,
					HeadingText:         "Section B.2",
					HeadingLevel:        4,
					ContentText:         "#### Section B.2\n\nText from section B2",
					FileLineNumberStart: 21,
					FileLineNumberEnd:   23,
					BodyLineNumberStart: 21,
					BodyLineNumberEnd:   23,
				}, sectionB2)

				// Now check walking the sections
				err = md.WalkSections(func(parent *core.MarkdownSection, current *core.MarkdownSection, children []*core.MarkdownSection) error {

					if current.HeadingText == "Title" {
						assert.Nil(t, parent)
						assert.ElementsMatch(t, []*core.MarkdownSection{sectionSubtitle}, children)
					}

					if current.HeadingText == "Subtitle" {
						assert.Equal(t, sectionTitle, parent)
						assert.ElementsMatch(t, []*core.MarkdownSection{sectionA, sectionB}, children)
					}

					if current.HeadingText == "Section A" {
						assert.Equal(t, sectionSubtitle, parent)
						assert.ElementsMatch(t, []*core.MarkdownSection{sectionA1}, children)
					}

					if current.HeadingText == "Section A.1" {
						assert.Equal(t, sectionA, parent)
						assert.Empty(t, children)
					}

					if current.HeadingText == "Section B" {
						assert.Equal(t, sectionSubtitle, parent)
						assert.ElementsMatch(t, []*core.MarkdownSection{sectionB1, sectionB2}, children)
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
			test: func(t *testing.T, md *core.MarkdownFile) {
				assert.Equal(t, "# A comment\ntitle: Title\ntags: [tag1, tag2]\nrating: 3\nlinks:\n- https://github.com\n", md.FrontMatter())
				fmMap, err := md.FrontMatterAsMap()
				require.NoError(t, err)
				assert.Equal(t, map[string]any{
					"title":  "Title",
					"tags":   []interface{}{"tag1", "tag2"},
					"rating": 3,
					"links":  []interface{}{"https://github.com"},
				}, fmMap)
				fmNode, err := md.FrontMatterAsNode()
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
				assert.Equal(t, 10, md.BodyLineNumber())
				assert.True(t, strings.HasPrefix(md.Body(), "# Title")) // Must start with the first non-empty line
				// 2. Check sections
				sections, err := md.GetSections()
				require.NoError(t, err)
				sectionTitle := sections[0]
				assert.Equal(t, 10, sectionTitle.FileLineNumberStart)
				assert.Equal(t, 16, sectionTitle.FileLineNumberEnd)
				assert.Equal(t, 1, sectionTitle.BodyLineNumberStart)
				assert.Equal(t, 7, sectionTitle.BodyLineNumberEnd)
			},
		},

		{
			name:   "Code Blocks",
			golden: "code-block",
			test: func(t *testing.T, md *core.MarkdownFile) {
				sections, err := md.GetSections()
				require.NoError(t, err)
				assert.Len(t, sections, 3)

				sectionTitle := sections[0]

				sectionFenced := sections[1]
				assert.Equal(t, &core.MarkdownSection{
					Parent:              sectionTitle,
					HeadingText:         "Fenced Code Block",
					HeadingLevel:        2,
					ContentText:         "## Fenced Code Block\n\n```md\n# Heading inside a block code\n\nSome text\n```",
					FileLineNumberStart: 3,
					FileLineNumberEnd:   9,
					BodyLineNumberStart: 3,
					BodyLineNumberEnd:   9,
				}, sectionFenced)

				sectionIndent := sections[2]
				assert.Equal(t, &core.MarkdownSection{
					Parent:              sectionTitle,
					HeadingText:         "Indented Code Block",
					HeadingLevel:        2,
					ContentText:         "## Indented Code Block\n\n    # Heading inside a block code\n\n    Some text",
					FileLineNumberStart: 11,
					FileLineNumberEnd:   15,
					BodyLineNumberStart: 11,
					BodyLineNumberEnd:   15,
				}, sectionIndent)
			},
		},

		// Add more test cases here to enrich Markdown support
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			filename := testutil.SetUpFromGoldenFileNamed(t, "markdown/"+testcase.golden+".md")
			md, err := core.ParseMarkdownFile(filename)
			require.NoError(t, err)
			testcase.test(t, md)
		})
	}
}

/* Test Helpers */
