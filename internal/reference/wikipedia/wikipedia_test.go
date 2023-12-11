package wikipedia

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		} else if r.URL.Path == "/w/api.php" && r.URL.Query().Get("action") == "parse" && r.URL.Query().Get("pageid") == "21492751" {
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
		} else if r.URL.Path == "/w/api.php" && r.URL.Query().Get("action") == "parse" && r.URL.Query().Get("pageid") == "32300871" {
			// Ex: https://en.wikipedia.org/w/api.php?action=parse&pageid=32300871&contentmodel=wikitext&prop=wikitext&format=json
			fmt.Fprintln(w, `
{
  "parse":{
    "title":"Nelson Mandela",
    "pageid":21492751,
    "wikitext":{
      "*":"{{Short description|None}}\n{{EngvarB|date=August 2014}}\n{{Use dmy dates|date=August 2014}}\n{{use South African English|date=November 2017}}\n{{Infobox administration\n| image = Mandela 1991.jpg\n| name =  Presidency of Nelson Mandela\n| term_start = 10 May 1994\n| term_end = 14 June 1999\n| president = Nelson Mandela\n| president_link = President of South Africa\n| cabinet = [[Cabinet of Nelson Mandela]]\n| election = [[1994 South African general election|1994]]\n| seat = [[Mahlamba Ndlopfu]], [[Pretoria]]<br />[[Genadendal Residence]], [[Cape Town]]\n| party = [[African National Congress]]\n| predecessor = ''[[F. W. de Klerk#State presidency|de Klerk state presidency]]''\n| successor = [[Thabo Mbeki|Mbeki]] presidency\n| seal = Coat of arms of South"
    }
  }
}`)
		}
	}))
	defer ts.Close()

	manager := NewManager()
	manager.BaseURL = ts.URL

	results, err := manager.Search("Nelson Mandela")
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Check first result
	firstResult := results[0]
	actual := firstResult.Attributes()
	expected := map[string]any{
		"name":        "Nelson Mandela",
		"pageId":      21492751,
		"url":         "https://en.wikipedia.org/wiki/Nelson_Mandela",
		"birth_date":  "1918-07-18",
		"birth_place": "Mvezo, Union of South Africa",
		"death_date":  "2013-12-05",
		"death_place": "Johannesburg, South Africa",
		"office":      "President of South Africa",
	}
	assert.Equal(t, expected, actual)

	// Check second result
	secondResult := results[1]
	actual = secondResult.Attributes()
	expected = map[string]any{
		"cabinet":        "Cabinet of Nelson Mandela",
		"election":       "1994",
		"image":          "Mandela 1991.jpg",
		"name":           "Presidency of Nelson Mandela",
		"pageId":         32300871,
		"party":          "African National Congress",
		"predecessor":    "de Klerk state presidency",
		"president":      "Nelson Mandela",
		"president_link": "President of South Africa",
		"seal":           "Coat of arms of South",
		"seat":           "Mahlamba Ndlopfu, Pretoria<br />Genadendal Residence, Cape Town",
		"successor":      "Mbeki presidency",
		"term_end":       "1999-06-14",
		"term_start":     "1994-05-10",
		"url":            "https://en.wikipedia.org/wiki/Nelson_Mandela",
	}
	assert.Equal(t, expected, actual)
}
