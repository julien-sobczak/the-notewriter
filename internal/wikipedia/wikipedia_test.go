package wikipedia

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AlecAivazis/survey/v2"
	score "github.com/AlecAivazis/survey/v2/core"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/julien-sobczak/the-notetaker/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.nhat.io/surveyexpect"
	"go.nhat.io/surveyexpect/options"
)

func init() {
	// disable color output for all prompts to simplify testing
	score.DisableColor = true
}

func TestSearch(t *testing.T) {
	// Ex: https://en.wikipedia.org/wiki/Nelson_Mandela

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/w/api.php" && r.URL.Query().Get("action") == "query" {
			// Ex: https://en.wikipedia.org/w/api.php?action=query&list=search&srsearch=Nelson%20Mandela&utf8=&format=json
			fmt.Fprintln(w, `
{
  "batchcomplete":"",
  "continue":{
    "sroffset":10,
    "continue":"-||"
  },
  "query":{
    "searchinfo":{
      "totalhits":6806
    },
    "search":[
      {
        "ns":0,
        "title":"Nelson Mandela",
        "pageid":21492751,
        "size":199842,
        "wordcount":23891,
        "snippet":"<span class=\"searchmatch\">Nelson</span> <span class=\"searchmatch\">Mandela</span> (/mænˈdɛlə/; Xhosa: [xolíɬaɬa <span class=\"searchmatch\">mandɛ̂ːla</span>]; 18 July 1918 – 5 December 2013) was a South African anti-apartheid activist who served",
        "timestamp":"2022-09-09T12:42:59Z"
      },
      {
        "ns":0,
        "title":"Presidency of Nelson Mandela",
        "pageid":32300871,
        "size":12395,
        "wordcount":1197,
        "snippet":"The presidency of <span class=\"searchmatch\">Nelson</span> <span class=\"searchmatch\">Mandela</span> began on 10 May 1994, when <span class=\"searchmatch\">Nelson</span> <span class=\"searchmatch\">Mandela</span>, an anti-apartheid activist, leader of Umkhonto We Sizwe, lawyer, and former",
        "timestamp":"2022-09-17T20:29:13Z"
      }
    ]
  }
}`)
		} else if r.URL.Path == "/w/api.php" && r.URL.Query().Get("action") == "parse" {
			// Ex: https://en.wikipedia.org/w/api.php?action=parse&pageid=21492751&contentmodel=wikitext&prop=wikitext&format=json
			fmt.Fprintln(w, `
{
  "parse":{
    "title":"Nelson Mandela",
    "pageid":21492751,
    "wikitext":{
      "*":"{{Short description|First president of South Africa from 1994 to 1999}}\n{{Infobox officeholder\n| office           = President of South Africa\n| birth_date       = {{Birth date|df=y|1918|7|18}}\n| birth_place      = [[Mvezo]], [[Union of South Africa]]\n| death_date       = {{death date and age|df=y|2013|12|05|1918|7|18}}\n| death_place      = [[Johannesburg]], South&nbsp;Africa\n}}\n'''Nelson Rolihlahla Mandela''' ({{IPAc-en|m|\u00e6|n|\u02c8|d|\u025b|l|\u0259}};<ref>{{cite web| title=Mandela| url=http://www.collinsdictionary.com/dictionary/english/mandela| work=[[Collins English Dictionary]]| access-date=17 December 2013 |archive-url=https://web.archive.org/web/20160405011219/http://www.collinsdictionary.com/dictionary/english/mandela |archive-date=5 April 2016 |url-status=live}}</ref> {{IPA-xh|xol\u00ed\u026ca\u026ca mand\u025b\u0302\u02d0la|lang}}; 18 July 1918&nbsp;\u2013 5 December 2013) was a South African [[Internal resistance to apartheid|anti-apartheid]] activist<!-- NOTE: The lead sentence should stick to what he was primarily known for. The infobox is there to include additional occupations. --> who served as the [[President of South Africa|first president of South Africa]] from 1994 to 1999."
    }
  }
}`)
		}
	}))
	defer ts.Close()

	s := surveyexpect.Expect(func(s *surveyexpect.Survey) {
		s.ExpectSelect("Which page?  [Use arrows to move, type to filter]").
			ExpectOptions(
				"> Nelson Mandela",
				"Presidency of Nelson Mandela",
			).
			Enter()
		s.ExpectMultiSelect("Which attributes?  [Use arrows to move, space to select, <right> to all, <left> to none, type to filter]").
			ExpectOptions(
				"> [ ]  office: President of South Africa",
				"[ ]  birth_date: 1918-07-18",
				"[ ]  birth_place: Mvezo, Union of South Africa",
				"[ ]  death_date: 2013-12-05",
				"[ ]  death_place: Johannesburg, South Africa",
			).
			MoveDown().
			Select().
			MoveDown().
			Select().
			Enter()
	})(t)
	s.Start(func(stdio terminal.Stdio) {
		manager := NewReferenceManager()
		manager.BaseURL = ts.URL
		manager.Stdio = &stdio

		reference, err := manager.Search("Nelson Mandela")
		require.NoError(t, err)
		file := core.NewFileFromAttributes(reference.Attributes())
		frontMatter, err := file.FrontMatterString()
		require.NoError(t, err)
		assert.Equal(t,
			strings.TrimSpace(`
name: Nelson Mandela
pageId: 21492751
url: https://en.wikipedia.org/wiki/Nelson_Mandela
birth_date: "1918-07-18"
birth_place: Mvezo, Union of South Africa
`),
			strings.TrimSpace(frontMatter))
	})

}

/* Learning tests */

func TestDemoSurveyExpect(t *testing.T) {
	s := surveyexpect.Expect(func(s *surveyexpect.Survey) {
		s.ExpectPassword("Enter a password:").
			Answer("secret")
		s.ExpectSelect("Which digit?  [Use arrows to move, type to filter]").
			Enter()
		s.ExpectMultiSelect("Which color?  [Use arrows to move, space to select, <right> to all, <left> to none, type to filter]").
			Select().
			Enter()
	})(t)
	s.Start(func(stdio terminal.Stdio) {
		p := &survey.Password{Message: "Enter a password:"}
		var answer1 string
		err := survey.AskOne(p, &answer1, options.WithStdio(stdio))
		assert.NoError(t, err)
		assert.Equal(t, "secret", answer1)

		var answer2 string
		prompt1 := &survey.Select{
			Message: "Which digit?",
			Options: []string{"one", "two"},
		}
		err = survey.AskOne(prompt1, &answer2, survey.WithValidator(survey.Required), options.WithStdio(stdio))
		assert.NoError(t, err)
		assert.Equal(t, "one", answer2)

		var answers3 []string
		prompt2 := &survey.MultiSelect{
			Message: "Which color?",
			Options: []string{"blue", "yellow"},
		}
		err = survey.AskOne(prompt2, &answers3, survey.WithValidator(survey.Required), options.WithStdio(stdio))
		assert.NoError(t, err)
		assert.Equal(t, []string{"blue"}, answers3)
	})
}
