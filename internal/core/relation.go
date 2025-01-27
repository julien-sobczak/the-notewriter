package core

import (
	"database/sql"
	"fmt"

	"github.com/julien-sobczak/the-notewriter/pkg/oid"
)

type Relation struct {
	// Source
	SourceOID  oid.OID `yaml:"source_oid" json:"source_oid"`
	SourceKind string  `yaml:"source_kind" json:"source_kind"`

	// Target
	TargetOID  oid.OID `yaml:"target_oid" json:"target_oid"`
	TargetKind string  `yaml:"target_kind" json:"target_kind"`

	Type string `yaml:"type" json:"type"`
}

func NewRelationFromObjects(objA, objB Object, relationType string) *Relation {
	return NewRelation(objA.UniqueOID(), objA.Kind(), objB.UniqueOID(), objB.Kind(), relationType)
}

// NewRelation instantiates a new relation.
func NewRelation(oidA oid.OID, kindA string, oidB oid.OID, kindB string, relationType string) *Relation {
	r := &Relation{
		SourceOID:  oidA,
		SourceKind: kindA,
		TargetOID:  oidB,
		TargetKind: kindB,
		Type:       relationType,
	}
	return r
}

func (r Relation) String() string {
	return fmt.Sprintf("relation %s[%s] -> %s -> %s[%s]", r.SourceKind, r.SourceOID, r.Type, r.TargetKind, r.TargetOID)
}

func (r *Relation) ToYAML() string {
	return ToBeautifulYAML(r)
}

func (r *Relation) ToJSON() string {
	return ToBeautifulJSON(r)
}

func (r *Relation) ToMarkdown() string {
	return fmt.Sprintf("%s(%s) -> %s(%s)", r.SourceKind, r.SourceOID, r.TargetKind, r.TargetOID)
}

/* Database Management */

func (r *Repository) DeleteRelations(obj Object) error {
	if obj.UniqueOID() == "" {
		// No relation was saved
		return nil
	}
	CurrentLogger().Debugf("Deleting relations from/to %s...", obj.UniqueOID())
	query := `DELETE FROM relation WHERE source_oid = ? or target_oid = ?;`
	res, err := CurrentDB().Client().Exec(query, obj.UniqueOID(), obj.UniqueOID())
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	CurrentLogger().Debugf("Deleted %d rows in table 'relation'", rows)
	return nil
}

func (r *Repository) UpdateRelations(source Object) error {
	// We systematically recreate all relations to be sure to not have dangling relations
	// (= relations that no longer exist in notes but are still present in database)

	// First, delete existing relations
	CurrentLogger().Debugf("Deleting relations from %s...", source.UniqueOID())
	query := `DELETE FROM relation WHERE source_oid = ?;`
	res, err := CurrentDB().Client().Exec(query, source.UniqueOID())
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	CurrentLogger().Debugf("Deleted %d rows in table 'relation'", rows)

	// Second, create the current relations
	for _, relation := range source.Relations() {
		CurrentLogger().Debugf("Inserting relation %s...", relation)
		query := `
			INSERT INTO relation(
				source_oid,
				source_kind,
				target_oid,
				target_kind,
				"type"
			)
			VALUES (?, ?, ?, ?, ?);
		`
		_, err := CurrentDB().Client().Exec(query,
			relation.SourceOID,
			relation.SourceKind,
			relation.TargetOID,
			relation.TargetKind,
			relation.Type,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// CountRelations returns the total number of relations.
func (r *Repository) CountRelations() (int, error) {
	var count int
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM relation`).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repository) FindRelations() ([]*Relation, error) {
	return QueryRelations(CurrentDB().Client(), "")
}

func (r *Repository) FindRelationsTo(oid oid.OID) ([]*Relation, error) {
	return QueryRelations(CurrentDB().Client(), `WHERE target_oid = ?`, oid)
}

func (r *Repository) FindRelationsFrom(oid oid.OID) ([]*Relation, error) {
	return QueryRelations(CurrentDB().Client(), `WHERE source_oid = ?`, oid)
}

/* SQL Helpers */

func QueryRelation(db SQLClient, whereClause string, args ...any) (*Relation, error) {
	var r Relation

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			source_oid,
			source_kind,
			target_oid,
			target_kind,
			type
		FROM relation
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

func QueryRelations(db SQLClient, whereClause string, args ...any) ([]*Relation, error) {
	var relations []*Relation

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			source_oid,
			source_kind,
			target_oid,
			target_kind,
			type
		FROM relation
		%s;`, whereClause), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var r Relation

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

		relations = append(relations, &r)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return relations, err
}
