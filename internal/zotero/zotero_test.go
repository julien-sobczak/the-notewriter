package zotero

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZoteroReferenceOnBook(t *testing.T) {
	referenceJSON := `  {
    "key": "TW7KH7DX",
    "version": 0,
    "itemType": "book",
    "creators": [
      {
        "firstName": "Patrick",
        "lastName": "Lencioni",
        "creatorType": "author"
      }
    ],
    "tags": [
      {
        "tag": "Teams in the workplace",
        "type": 1
      }
    ],
    "ISBN": "9780787960759",
    "title": "The five dysfunctions of a team: a leadership fable",
    "edition": "1st ed",
    "place": "San Francisco",
    "publisher": "Jossey-Bass",
    "date": "2002",
    "numPages": "229",
    "callNumber": "HD66 .L456 2002",
    "libraryCatalog": "Library of Congress ISBN",
    "shortTitle": "The five dysfunctions of a team"
  }`
	var reference map[string]interface{}
	err := json.Unmarshal([]byte(referenceJSON), &reference)
	require.NoError(t, err)

	ref := ZoteroReference{
		fields: reference,
	}
	assert.Equal(t, "The five dysfunctions of a team: a leadership fable", ref.Title())
	assert.Equal(t, "The five dysfunctions of a team", ref.ShortTitle())
	assert.Equal(t, []string{"Patrick Lencioni"}, ref.Authors())
	assert.Equal(t, "2002", ref.PublicationYear())
	assert.Contains(t, ref.Attributes(), "ISBN")
}

// TODO add many other examples of various references
