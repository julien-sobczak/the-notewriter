package core

import (
	"database/sql"
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

func NewLinkFromObjects(objA, objB Object, relationship string) *Link {
	return NewLink(objA.UniqueOID(), objA.Kind(), objB.UniqueOID(), objB.Kind(), relationship)
}

// NewLink instantiates a new link.
func NewLink(oidA oid.OID, kindA string, oidB oid.OID, kindB string, relationship string) *Link {
	r := &Link{
		SourceOID:  oidA,
		SourceKind: kindA,
		TargetOID:  oidB,
		TargetKind: kindB,
		Type:       relationship,
	}
	return r
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

func (r *Repository) DeleteLinks(obj Object) error {
	if obj.UniqueOID() == "" {
		// No link was saved
		return nil
	}
	CurrentLogger().Debugf("Deleting links from/to %s...", obj.UniqueOID())
	query := `DELETE FROM link WHERE source_oid = ? or target_oid = ?;`
	res, err := CurrentDB().Client().Exec(query, obj.UniqueOID(), obj.UniqueOID())
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	CurrentLogger().Debugf("Deleted %d rows in table 'link'", rows)
	return nil
}

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
func (r *Repository) CountLinks() (int, error) {
	var count int
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM link`).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repository) FindLinks() ([]*Link, error) {
	return QueryLinks(CurrentDB().Client(), "")
}

func (r *Repository) FindLinksTo(oid oid.OID) ([]*Link, error) {
	return QueryLinks(CurrentDB().Client(), `WHERE target_oid = ?`, oid)
}

func (r *Repository) FindLinksFrom(oid oid.OID) ([]*Link, error) {
	return QueryLinks(CurrentDB().Client(), `WHERE source_oid = ?`, oid)
}

/* SQL Helpers */

func QueryLink(db SQLClient, whereClause string, args ...any) (*Link, error) {
	var r Link

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			source_oid,
			source_kind,
			target_oid,
			target_kind,
			type
		FROM link
		%s;`, whereClause), args...).
		Scan(
			&r.SourceOID,
			&r.SourceKind,
			&r.TargetOID,
			&r.TargetKind,
			&r.Type,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &r, nil
}

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
