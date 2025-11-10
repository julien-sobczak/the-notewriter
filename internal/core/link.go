package core

import (
	"fmt"

	"github.com/julien-sobczak/the-notewriter/pkg/oid"
)

type Link struct {
	// Source
	SourceOID  oid.OID `yaml:"source_oid" json:"source_oid"`
	SourceKind string  `yaml:"source_kind" json:"source_kind"`

	// Target
	TargetOID  oid.OID `yaml:"target_oid" json:"target_oid"`
	TargetKind string  `yaml:"target_kind" json:"target_kind"`

	Type string `yaml:"type" json:"type"`
}



func (r Link) String() string {
	return fmt.Sprintf("link %s[%s] -> %s -> %s[%s]", r.SourceKind, r.SourceOID, r.Type, r.TargetKind, r.TargetOID)
}

func (r *Link) ToYAML() string {
	return ToBeautifulYAML(r)
}

func (r *Link) ToJSON() string {
	return ToBeautifulJSON(r)
}

func (r *Link) ToMarkdown() string {
	return fmt.Sprintf("%s(%s) -> %s(%s)", r.SourceKind, r.SourceOID, r.TargetKind, r.TargetOID)
}

/* Database Management */



func (r *Repository) UpdateLinks(source Object, additionalLinks []*Link) error {
	// We systematically recreate all links to be sure to not have dangling links
	// (= links that no longer exist in notes but are still present in database)

	// First, delete existing links
	CurrentLogger().Debugf("Deleting links from %s...", source.UniqueOID())
	query := `DELETE FROM link WHERE source_oid = ?;`
	res, err := CurrentDB().Client().Exec(query, source.UniqueOID())
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	CurrentLogger().Debugf("Deleted %d rows in table 'link'", rows)

	// Combine links from source object and additional links (use copy to avoid modifying source)
	allLinks := append([]*Link{}, source.Links()...)
	allLinks = append(allLinks, additionalLinks...)

	// Second, create the current links
	for _, link := range allLinks {
		CurrentLogger().Debugf("Inserting link %s...", link)
		query := `
			INSERT INTO link(
				source_oid,
				source_kind,
				target_oid,
				target_kind,
				"type"
			)
			VALUES (?, ?, ?, ?, ?);
		`
		_, err := CurrentDB().Client().Exec(query,
			link.SourceOID,
			link.SourceKind,
			link.TargetOID,
			link.TargetKind,
			link.Type,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// CountLinks returns the total number of links.


func (r *Repository) FindLinks() ([]*Link, error) {
	return QueryLinks(CurrentDB().Client(), "")
}



/* SQL Helpers */



func QueryLinks(db SQLClient, whereClause string, args ...any) ([]*Link, error) {
	var links []*Link

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			source_oid,
			source_kind,
			target_oid,
			target_kind,
			type
		FROM link
		%s;`, whereClause), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var r Link

		err = rows.Scan(
			&r.SourceOID,
			&r.SourceKind,
			&r.TargetOID,
			&r.TargetKind,
			&r.Type,
		)
		if err != nil {
			return nil, err
		}

		links = append(links, &r)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return links, err
}
