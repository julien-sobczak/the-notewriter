# Developer's Guide

## Wikipedia

Metadata about persons are retrieved using Wikipedia API's

Ex: Nelson Mandela

```shell
$ wget https://en.wikipedia.org/w/api.php?action=query&list=search&srsearch=Nelson%20Mandela&utf8=&format=json
{
    "batchcomplete":"",
    "continue":{"sroffset":10,"continue":"-||"},
    "query":{
        "searchinfo":{"totalhits":6806},
        "search":[
            {"ns":0,"title":"Nelson Mandela","pageid":21492751,"size":199842,"wordcount":23891,"snippet":"...","timestamp":"2022-09-09T12:42:59Z"},
            {"ns":0,"title":"Death and state funeral of Nelson Mandela","pageid":41284488,"size":133945,"wordcount":13656,"snippet":"...","timestamp":"2022-09-15T15:28:34Z"},
        ]
    }
}
```

The parameter `srsearch` is explained in the [module `query` API documentation](https://en.wikipedia.org/w/api.php?action=help&modules=query)

Once we choose a `pageid`, we can retrieve the page content using the [module `parse` API](https://en.wikipedia.org/w/api.php?action=help&modules=parse):

```
# HTML
$ wget https://en.wikipedia.org/w/api.php?action=parse&pageid=21492751&contentmodel=wikitext&prop=wikitext
{
    "parse": {
        "title": "Nelson Mandela",
        "pageid": 21492751,
        "text": {
            "*": "<div class=\"mw-parser-output\">...</div>"
        }
    }
}

# Wikitext
$ wget https://en.wikipedia.org/w/api.php?action=parse&pageid=21492751&contentmodel=wikitext&prop=wikitext
{
    "parse": {
        "title": "Nelson Mandela",
        "pageid": 21492751,
        "wikitext": {
            "*": "{{Short description|First president of South Africa from 1994 to 1999}}...\n[[Category:Xhosa people]]\n[[Category:International Sim\u00f3n Bol\u00edvar Prize recipients]]"
        }
    }
}
```

Metadata is present in [Infobox](https://en.wikipedia.org/wiki/Help:Infobox) using this syntax:

```
{{Infobox person
|name    =
|image   =
|caption =
...
|website =
}}
```

Ex: Nelson Mandela (continued)

In Wikitext:

```
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
```

In HTML:

```html
<table class=\"infobox vcard\">
  <tbody>
    <tr>
      <th colspan=\"2\" class=\"infobox-above\"><div class=\"honorific-prefix\"><a href=\"/wiki/His_Excellency\" class=\"mw-redirect\" title=\"His Excellency\">His Excellency</a></div>
      <div class=\"fn\">Nelson Mandela</div>
      <div class=\"honorific-suffix\"><span class=\"noexcerpt nowraplinks\"><a href=\"/wiki/Order_of_Mapungubwe\" title=\"Order of Mapungubwe\">OMP</a>&#32;<a href=\"/wiki/Star_for_Bravery_in_Gold\" title=\"Star for Bravery in Gold\">SBG</a>&#32;<a href=\"/wiki/Star_for_Bravery_in_Silver\" title=\"Star for Bravery in Silver\">SBS</a>&#32;<a href=\"/wiki/Conspicuous_Leadership_Star\" title=\"Conspicuous Leadership Star\">CLS</a>&#32;<a href=\"/wiki/Decoration_for_Merit_in_Gold\" title=\"Decoration for Merit in Gold\">DMG</a>&#32;<a href=\"/wiki/Merit_Medal_in_Silver\" title=\"Merit Medal in Silver\">MMS</a>&#32;<a href=\"/wiki/Merit_Medal_in_Bronze\" title=\"Merit Medal in Bronze\">MMB</a></span></div></th>
    </tr>
    <tr>
      <td colspan=\"2\" class=\"infobox-image\">
        <a href=\"/wiki/File:Nelson_Mandela_1994.jpg\" class=\"image\"><img alt=\"Portrait photograph of a 76-year-old President Mandela\" src=\"//upload.wikimedia.org/wikipedia/commons/thumb/0/02/Nelson_Mandela_1994.jpg/220px-Nelson_Mandela_1994.jpg\" decoding=\"async\" width=\"220\" height=\"285\" srcset=\"//upload.wikimedia.org/wikipedia/commons/thumb/0/02/Nelson_Mandela_1994.jpg/330px-Nelson_Mandela_1994.jpg 1.5x, //upload.wikimedia.org/wikipedia/commons/thumb/0/02/Nelson_Mandela_1994.jpg/440px-Nelson_Mandela_1994.jpg 2x\" data-file-width=\"1500\" data-file-height=\"1940\" /></a>
        <div class=\"infobox-caption\">Mandela in Washington, D.C., 1994</div>
      </td>
    </tr>
    <tr>
      <td colspan=\"2\" class=\"infobox-full-data\">
        <link rel=\"mw-deduplicated-inline-style\" href=\"mw-data:TemplateStyles:r1066479718\"/>
      </td>
    </tr>
    <tr>
      <th colspan=\"2\" class=\"infobox-header\">1st&#32;<a href=\"/wiki/President_of_South_Africa\" title=\"President of South Africa\">President of South Africa</a></th>
    </tr>
    <tr>
      <td colspan=\"2\" class=\"infobox-full-data\"><span class=\"nowrap\"><b>In office</b></span><br />10 May 1994&#160;\u2013&#32;14 June 1999</td>
      <link rel=\"mw-deduplicated-inline-style\" href=\"mw-data:TemplateStyles:r1066479718\"/>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\"><a href=\"/wiki/Deputy_President_of_South_Africa\" title=\"Deputy President of South Africa\">Deputy</a></th>
      <td class=\"infobox-data\">
        <div class=\"plainlist\">
          \n
          <ul>
            <li>Thabo Mbeki</li>
            \n
            <li>F. W. de Klerk</li>
          </ul>
          \n
        </div>
      </td>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\"><span class=\"nowrap\">Preceded by</span></th>
      <td class=\"infobox-data\"><a href=\"/wiki/F._W._de_Klerk\" title=\"F. W. de Klerk\">F. W. de Klerk</a> <span class=\"avoidwrap\">(as <a href=\"/wiki/State_President_of_South_Africa\" title=\"State President of South Africa\">State President</a>)</span></td>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\"><span class=\"nowrap\">Succeeded by</span></th>
      <td class=\"infobox-data\"><a href=\"/wiki/Thabo_Mbeki\" title=\"Thabo Mbeki\">Thabo Mbeki</a></td>
      <link rel=\"mw-deduplicated-inline-style\" href=\"mw-data:TemplateStyles:r1066479718\"/>
    </tr>
    <tr>
      <th colspan=\"2\" class=\"infobox-header\">19th&#32;<a href=\"/wiki/Secretary-General_of_the_Non-Aligned_Movement\" class=\"mw-redirect\" title=\"Secretary-General of the Non-Aligned Movement\">Secretary-General of the Non-Aligned Movement</a></th>
    </tr>
    <tr>
      <td colspan=\"2\" class=\"infobox-full-data\"><span class=\"nowrap\"><b>In office</b></span><br />2 September 1998&#160;\u2013&#32;14 June 1999</td>
      <link rel=\"mw-deduplicated-inline-style\" href=\"mw-data:TemplateStyles:r1066479718\"/>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\"><span class=\"nowrap\">Preceded by</span></th>
      <td class=\"infobox-data\"><a href=\"/wiki/Andr%C3%A9s_Pastrana_Arango\" title=\"Andr\u00e9s Pastrana Arango\">Andr\u00e9s Pastrana Arango</a></td>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\"><span class=\"nowrap\">Succeeded by</span></th>
      <td class=\"infobox-data\">Thabo Mbeki</td>
      <link rel=\"mw-deduplicated-inline-style\" href=\"mw-data:TemplateStyles:r1066479718\"/>
    </tr>
    <tr>
      <th colspan=\"2\" class=\"infobox-header\">11th&#32;<a href=\"/wiki/President_of_the_African_National_Congress\" class=\"mw-redirect\" title=\"President of the African National Congress\">President of the African National Congress</a></th>
    </tr>
    <tr>
      <td colspan=\"2\" class=\"infobox-full-data\"><span class=\"nowrap\"><b>In office</b></span><br />7 July 1991&#160;\u2013&#32;20 December 1997</td>
      <link rel=\"mw-deduplicated-inline-style\" href=\"mw-data:TemplateStyles:r1066479718\"/>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\"><a href=\"/wiki/Deputy_President_of_the_African_National_Congress\" class=\"mw-redirect\" title=\"Deputy President of the African National Congress\">Deputy</a></th>
      <td class=\"infobox-data\">
        <div class=\"plainlist\">
          \n
          <ul>
            <li><a href=\"/wiki/Walter_Sisulu\" title=\"Walter Sisulu\">Walter Sisulu</a></li>
            \n
            <li>Thabo Mbeki</li>
          </ul>
          \n
        </div>
      </td>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\"><span class=\"nowrap\">Preceded by</span></th>
      <td class=\"infobox-data\"><a href=\"/wiki/Oliver_Tambo\" title=\"Oliver Tambo\">Oliver Tambo</a></td>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\"><span class=\"nowrap\">Succeeded by</span></th>
      <td class=\"infobox-data\">Thabo Mbeki</td>
      <link rel=\"mw-deduplicated-inline-style\" href=\"mw-data:TemplateStyles:r1066479718\"/>
    </tr>
    <tr>
      <th colspan=\"2\" class=\"infobox-header\">4th&#32;<a href=\"/wiki/Deputy_President_of_the_African_National_Congress\" class=\"mw-redirect\" title=\"Deputy President of the African National Congress\">Deputy President of the African National Congress</a></th>
    </tr>
    <tr>
      <td colspan=\"2\" class=\"infobox-full-data\"><span class=\"nowrap\"><b>In office</b></span><br />25 June 1985&#160;\u2013&#32;6 July 1991</td>
      <link rel=\"mw-deduplicated-inline-style\" href=\"mw-data:TemplateStyles:r1066479718\"/>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\"><span class=\"nowrap\">Preceded by</span></th>
      <td class=\"infobox-data\">Oliver Tambo</td>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\"><span class=\"nowrap\">Succeeded by</span></th>
      <td class=\"infobox-data\">Walter Sisulu</td>
    </tr>
    <tr>
      <td colspan=\"2\">\n</td>
    </tr>
    <tr>
      <th colspan=\"2\" class=\"infobox-header\">Personal details</th>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\">Born</th>
      <td class=\"infobox-data\">
        <div class=\"nickname\">Rolihlahla Mandela</div>
        <br /><span>(<span class=\"bday\">1918-07-18</span>)</span>18 July 1918<br /><a href=\"/wiki/Mvezo\" title=\"Mvezo\">Mvezo</a>, <a href=\"/wiki/Union_of_South_Africa\" title=\"Union of South Africa\">Union of South Africa</a>
      </td>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\">Died</th>
      <td class=\"infobox-data\">5 December 2013<span>(2013-12-05)</span> (aged&#160;95)<br /><a href=\"/wiki/Johannesburg\" title=\"Johannesburg\">Johannesburg</a>, South&#160;Africa</td>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\">Resting place</th>
      <td class=\"infobox-data label\">Mandela Graveyard, <span class=\"avoidwrap\"><a href=\"/wiki/Qunu\" title=\"Qunu\">Qunu</a>, Eastern Cape</span></td>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\">Political party</th>
      <td class=\"infobox-data\"><a href=\"/wiki/African_National_Congress\" title=\"African National Congress\">African National Congress</a></td>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\">Other political<br />affiliations</th>
      <td class=\"infobox-data\"><a href=\"/wiki/South_African_Communist_Party\" title=\"South African Communist Party\">South African Communist</a></td>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\">Spouses</th>
      <td class=\"infobox-data\">
        <div class=\"plainlist\">
          \n
          <ul>
            <li class=\"mw-empty-elt\"></li>
          </ul>
          \n
          <div>
            <div><a href=\"/wiki/Evelyn_Mase\" title=\"Evelyn Mase\">Evelyn Ntoko Mase</a></div>
            \n
            <div>&#8203;</div>
            &#32;
            <div>&#8203;</div>
            &#40;<abbr title=\"married\">m.</abbr>&#160;<span title=\"5 October 1944\" class=\"rt-commentedText\">1944</span>&#59;&#32;<abbr title=\"divorced\">div.</abbr>&#160;<span title=\"19 March 1958\" class=\"rt-commentedText\">1958</span>&#41;<wbr />&#8203;
          </div>
          \n
          <ul>
            <li class=\"mw-empty-elt\"></li>
          </ul>
          \n
          <div>
            <div><a href=\"/wiki/Winnie_Madikizela-Mandela\" title=\"Winnie Madikizela-Mandela\">Winnie Madikizela</a></div>
            \n
            <div>&#8203;</div>
            &#32;
            <div>&#8203;</div>
            &#40;<abbr title=\"married\">m.</abbr>&#160;<span title=\"14 June 1958\" class=\"rt-commentedText\">1958</span>&#59;&#32;<abbr title=\"divorced\">div.</abbr>&#160;<span title=\"19 March 1996\" class=\"rt-commentedText\">1996</span>&#41;<wbr />&#8203;
          </div>
          \n
          <ul>
            <li class=\"mw-empty-elt\"></li>
          </ul>
          \n
          <div>
            <div><a href=\"/wiki/Gra%C3%A7a_Machel\" title=\"Gra\u00e7a Machel\">Gra\u00e7a Machel</a><br /></div>
            \n
            <div>&#8203;</div>
            &#32;
            <div>&#8203;</div>
            &#40;<abbr title=\"married\">m.</abbr>&#160;<span title=\"18 July 1998\" class=\"rt-commentedText\">1998</span>&#41;<wbr />&#8203;
          </div>
          \n
        </div>
      </td>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\">Children</th>
      <td class=\"infobox-data\">7, including <a href=\"/wiki/Makgatho_Mandela\" title=\"Makgatho Mandela\">Makgatho</a>, <a href=\"/wiki/Makaziwe_Mandela\" title=\"Makaziwe Mandela\">Makaziwe</a>, <a href=\"/wiki/Zenani_Mandela-Dlamini\" title=\"Zenani Mandela-Dlamini\">Zenani</a>, <a href=\"/wiki/Zindzi_Mandela\" title=\"Zindzi Mandela\">Zindziswa</a> and <a href=\"/wiki/Josina_Z._Machel\" title=\"Josina Z. Machel\">Josina</a> (step-daughter)</td>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\"><a href=\"/wiki/Alma_mater\" title=\"Alma mater\">Alma mater</a></th>
      <td class=\"infobox-data\">
        <div class=\"plainlist\">
          \n
          <ul>
            <li><a href=\"/wiki/University_of_Fort_Hare\" title=\"University of Fort Hare\">University of Fort Hare</a></li>
            \n
            <li><a href=\"/wiki/University_of_London\" title=\"University of London\">University of London</a></li>
            \n
            <li><a href=\"/wiki/University_of_South_Africa\" title=\"University of South Africa\">University of South Africa</a></li>
            \n
            <li><a href=\"/wiki/University_of_the_Witwatersrand\" title=\"University of the Witwatersrand\">University of the Witwatersrand</a></li>
          </ul>
          \n
        </div>
      </td>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\">Occupation</th>
      <td class=\"infobox-data\">
        <div class=\"hlist hlist-separated\">
          \n
          <ul>
            <li>Activist</li>
            \n
            <li>politician</li>
            \n
            <li>philanthropist</li>
            \n
            <li>lawyer</li>
          </ul>
          \n
        </div>
      </td>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\">Known for</th>
      <td class=\"infobox-data\"><a href=\"/wiki/Internal_resistance_to_apartheid\" title=\"Internal resistance to apartheid\">Internal resistance to apartheid</a></td>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\">Awards</th>
      <td class=\"infobox-data\">
        <div class=\"plainlist\">
          \n
          <ul>
            <li><a href=\"/wiki/Sakharov_Prize\" title=\"Sakharov Prize\">Sakharov Prize</a> (1988)</li>
            \n
            <li><a href=\"/wiki/Bharat_Ratna\" title=\"Bharat Ratna\">Bharat Ratna</a> (1990)</li>
            \n
            <li><a href=\"/wiki/Nishan-e-Pakistan\" title=\"Nishan-e-Pakistan\">Nishan-e-Pakistan</a> (1992)</li>
            \n
            <li><a href=\"/wiki/Nobel_Peace_Prize\" title=\"Nobel Peace Prize\">Nobel Peace Prize</a> (1993)</li>
            \n
            <li><a href=\"/wiki/Lenin_Peace_Prize\" title=\"Lenin Peace Prize\">Lenin Peace Prize</a> (1990)</li>
            \n
            <li><a href=\"/wiki/Presidential_Medal_of_Freedom\" title=\"Presidential Medal of Freedom\">Presidential Medal of Freedom</a> (2002)</li>
            \n
            <li><i>(<a href=\"/wiki/List_of_awards_and_honours_received_by_Nelson_Mandela\" title=\"List of awards and honours received by Nelson Mandela\">more...</a>)</i></li>
          </ul>
          \n
        </div>
      </td>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\">Website</th>
      <td class=\"infobox-data\"><span class=\"official-website\"><span class=\"url\"><a rel=\"nofollow\" class=\"external text\" href=\"http://nelsonmandela.org\">Foundation</a></span></span></td>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\">Nicknames</th>
      <td class=\"infobox-data\">
        <div class=\"hlist hlist-separated\">
          \n
          <ul>
            <li><a href=\"/wiki/Madiba\" class=\"mw-redirect\" title=\"Madiba\">Madiba</a></li>
            \n
            <li>Dalibunga</li>
          </ul>
          \n
        </div>
      </td>
    </tr>
    <tr>
      <td colspan=\"2\" class=\"infobox-full-data\">
        <link rel=\"mw-deduplicated-inline-style\" href=\"mw-data:TemplateStyles:r1066479718\"/>
        <b>Writing career</b>
      </td>
    </tr>
    <tr>
      <th scope=\"row\" class=\"infobox-label\">Notable works</th>
      <td class=\"infobox-data\"><i><a href=\"/wiki/Long_Walk_to_Freedom\" title=\"Long Walk to Freedom\">Long Walk to Freedom</a></i></td>
    </tr>
    <tr>
      <td colspan=\"2\">\n</td>
    </tr>
    <tr>
      <td colspan=\"2\" class=\"infobox-below\">
      <div></div>
      </td>
    </tr>
  </tbody>
</table>
```

_The Notetaker_ uses the librairies [goquery](https://github.com/PuerkitoBio/goquery) to extract metadata and [Survey](https://github.com/AlecAivazis/survey) to let the user choose the attributes to keep.

### Why privilege Wikitext over HTML?

Wikitext is more structured. Ex:

```html
<td class="infobox-data">
  <div style="display:inline" class="nickname">Rolihlahla Mandela</div>
  <br>
  <span style="display:none">(<span class="bday">1918-07-18</span>)
  </span>18 July 1918<br><a href="/wiki/Mvezo" title="Mvezo">Mvezo</a>,
  <a href="/wiki/Union_of_South_Africa" title="Union of South Africa">Union of South Africa</a>
</td>
```

Versus:

```
{{Infobox officeholder
| birth_name       = Rolihlahla Mandela
| birth_date       = {{Birth date|df=y|1918|7|18}}
| birth_place      = [[Mvezo]], [[Union of South Africa]]
}}

```

## Unit Testing

The code relies extensively on global variable (for example, to retrieve the current collection, the current database client or the current time). Global variables prevent to run tests in parallel. For a small project like this, I chose to favor code readability over efficiency. 
