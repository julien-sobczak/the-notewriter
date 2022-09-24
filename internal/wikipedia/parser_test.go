package wikipedia

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestParserWithInfobox(t *testing.T) {

	nelsonMandelaInfobox := `
{{Infobox officeholder
| honorific_prefix = [[His Excellency]]
| honorific_suffix = {{post-nominals|country=ZAR|OMP|SBG|SBS|CLS|DMG|MMS|MMB|size=100%}}
| image            = Nelson Mandela 1994.jpg
| alt              = Portrait photograph of a 76-year-old President Mandela
| caption          = Mandela in Washington, D.C., 1994
| order            = 1st
| office           = President of South Africa
| term_start       = 10 May 1994
| term_end         = 14 June 1999
| deputy           = {{plainlist|
* Thabo Mbeki
* F. W. de Klerk
}}
| predecessor      = [[F. W. de Klerk]] {{avoid wrap|(as [[State President of South Africa|State President]])}}
| successor        = [[Thabo Mbeki]]
| order3           = 11th
| office3          = President of the African National Congress
| deputy3          = {{plainlist|
* [[Walter Sisulu]]
* Thabo Mbeki
}}
| term_start3      = 7 July 1991
| term_end3        = 20 December 1997
| predecessor3     = [[Oliver Tambo]]
| successor3       = Thabo Mbeki
| order4           = 4th
| office4          = Deputy President of the African National Congress
| term_start4      = 25 June 1985
| term_end4        = 6 July 1991
| predecessor4     = Oliver Tambo
| successor4       = Walter Sisulu
| order2           = 19th
| office2          = Secretary-General of the Non-Aligned Movement
| term_start2      = 2 September 1998
| term_end2        = 14 June 1999
| predecessor2     = [[Andr\u00e9s Pastrana Arango]]
| successor2       = Thabo Mbeki
| birth_name       = Rolihlahla Mandela
| birth_date       = {{Birth date|df=y|1918|7|18}}
| birth_place      = [[Mvezo]], [[Union of South Africa]]
| death_date       = {{death date and age|df=y|2013|12|05|1918|7|18}}
| death_place      = [[Johannesburg]], South&nbsp;Africa
| resting_place    = Mandela Graveyard, {{avoid wrap|[[Qunu]], Eastern Cape}}
| party            = [[African National Congress]]
| otherparty       = [[South African Communist Party|South African Communist]]
| spouse           = {{plainlist|
* {{marriage|[[Evelyn Mase|Evelyn Ntoko Mase]]|5 October 1944|19 March 1958|reason = divorced}}
* {{marriage|[[Winnie Madikizela-Mandela|Winnie Madikizela]]|14 June 1958|19 March 1996|reason = divorced}}
* {{marriage|[[Gra\u00e7a Machel]]<br />|18 July 1998|<!-- Omission per Template:Marriage instructions -->}}
}}
| children         = 7, including {{enum|[[Makgatho Mandela|Makgatho]]|[[Makaziwe Mandela|Makaziwe]]|[[Zenani Mandela-Dlamini|Zenani]]|[[Zindzi Mandela|Zindziswa]]|[[Josina Z. Machel|Josina]] (step-daughter)}}
| alma_mater       = {{plainlist|
* [[University of Fort Hare]]
* [[University of London]]
* [[University of South Africa]]
* [[University of the Witwatersrand]]
}}
| occupation       = {{flatlist|
* Activist
* politician
* philanthropist
* lawyer
}}
| website          = {{official website|nelsonmandela.org|name=Foundation}}
| nickname         = {{flatlist|
* [[Madiba]]
* Dalibunga
}}
| known_for        = [[Internal resistance to apartheid]]
| awards           = {{plainlist|
* [[Sakharov Prize]] (1988)
* [[Bharat Ratna]] (1990)
* [[Nishan-e-Pakistan]] (1992)
* [[Nobel Peace Prize]] (1993)
* [[Lenin Peace Prize]] (1990)
* [[Presidential Medal of Freedom]] (2002)
* ''([[List of awards and honours received by Nelson Mandela|more...]])''
}}
| module           = {{Infobox writer | embed=yes
| notableworks     = ''[[Long Walk to Freedom]]''
}}
}}
`
	infobox := parseWikitext(nelsonMandelaInfobox)
	if len(infobox.Attributes) == 0 {
		t.Fail()
	}

	bytes, err := yaml.Marshal(infobox.Attributes)
	if err != nil {
		t.Fatalf("Unable to marshall: %v", err)
	}
	t.Log("\n---\n" + string(bytes) + "---")
	t.Fail()

	// TODO finish test
	// TODO convert struct to options for promptui + output FrontMatter
}

func TestStripLinks(t *testing.T) {
	var tests = []struct {
		name string // name
		wiki string // input
		text string // expected result
	}{
		{"Internal Link 1", "[[Main Page]]", "Main Page"},
		{"Internal Link 2", "[[Help:Contents]]", "Help:Contents"},
		{"Piped link 1", "[[Help:Editing pages|editing help]]", "editing help"},
		{"Piped link 2", "[[#See also|different text]]", "different text"},
		{"Pipe trick 1", "[[Manual:Extensions|]]", "Extensions"},
		{"Pipe trick 1", "[[User:John Doe|]]", "John Doe"},
		{"Word-ending links", "[[Help]]s", "Helps"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := stripLinks(tt.wiki)
			if actual != tt.text {
				t.Errorf("Expected %s, actual %s", tt.text, actual)
			}
		})
	}
}

func TestParseAttributeValue(t *testing.T) {
	var tests = []struct {
		name     string      // name
		rawValue string      // input
		expected interface{} // expected result
	}{

		{
			"string",
			"10 May 1994",
			"10 May 1994",
		},

		{
			"italic",
			"''italic''",
			"italic",
		},

		{
			"bold",
			"'''bold'''",
			"bold",
		},

		{
			"italic and bold",
			"'''''italic + bold'''''",
			"italic + bold",
		},

		{
			"link",
			"[[His Excellency]]",
			"His Excellency",
		},

		{
			"plainlist",
			`{{plainlist|
* Thabo Mbeki
* F. W. de Klerk
}}`,
			[]string{"Thabo Mbeki", "F. W. de Klerk"},
		},

		{
			"flatlist",
			`{{flatlist|
* Thabo Mbeki
* F. W. de Klerk
}}`,
			"Thabo Mbeki, F. W. de Klerk",
		},

		{
			"flatlist with a link",
			`{{flatlist|
* [[Thabo Mbeki]]
* F. W. de Klerk
}}`,
			"Thabo Mbeki, F. W. de Klerk",
		},

		{
			"birth_date 1",
			"{{Birth date|df=y|1918|7|18}}",
			"1918-07-18",
		},
		{
			"birth_date 2",
			"{{Birth date|2016|12|31|df=y}}",
			"2016-12-31",
		},
		{
			"birth_date 3",
			"{{birth-date|December 7, 1941}}",
			"1941-12-07",
		},
		{
			"birth_date 4",
			"{{birth-date|7 December 1941}}",
			"1941-12-07",
		},
		{
			"birth_date 5",
			"{{Birth-date|31 December 2016}}",
			"2016-12-31",
		},

		{
			"birth date and age 1",
			"{{birth date and age|df=y|1918|7|18}}",
			"1918-07-18",
		},
		{
			"birth date and age 2",
			"{{Birth-date and age|1941}}",
			"1941",
		},
		{
			"birth date and age 3",
			"{{Birth-date and age|September 1941}}",
			"1941-09",
		},
		{
			"birth date and age 4",
			"{{Birth year and age|1941|9}}",
			"1941-09",
		},
		{
			"birth date and age 5",
			"{{Birth-date and age|April 12, 1941}}",
			"1941-04-12",
		},
		{
			"birth date and age 6",
			"{{Birth-date and age|12 April 1941}}",
			"1941-04-12",
		},
		{
			"birth date and age 7",
			"{{Birth-date and age|1941-04-12|Twelfth of April, 1941}}",
			"1941-04-12",
		},

		{
			"death_date 1",
			"{{Death date|df=y|1918|7|18}}",
			"1918-07-18",
		},
		{
			"death_date 2",
			"{{Death date|2016|12|31|df=y}}",
			"2016-12-31",
		},
		{
			"death_date 3",
			"{{death-date|December 7, 1941}}",
			"1941-12-07",
		},
		{
			"death_date 4",
			"{{death-date|7 December 1941}}",
			"1941-12-07",
		},
		{
			"death_date 5",
			"{{Death-date|31 December 2016}}",
			"2016-12-31",
		},

		{
			"death date and age 1",
			"{{death date and age|df=y|2013|12|05|1918|7|18}}",
			"2013-12-05",
		},
		{
			"death date and age 2",
			"{{Death-date and age|1941}}",
			"1941",
		},
		{
			"death date and age 3",
			"{{Death-date and age|September 1941}}",
			"1941-09",
		},
		{
			"death date and age 4",
			"{{Death year and age|1941|9}}",
			"1941-09",
		},
		{
			"death date and age 5",
			"{{Death-date and age|April 12, 1941}}",
			"1941-04-12",
		},
		{
			"death date and age 6",
			"{{Death-date and age|12 April 1941}}",
			"1941-04-12",
		},
		{
			"death date and age 7",
			"{{Death-date and age|1941-04-12|Twelfth of April, 1941}}",
			"1941-04-12",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := parseAttributeValue(tt.rawValue)
			assert.Equal(t, tt.expected, actual)
		})
	}

}
