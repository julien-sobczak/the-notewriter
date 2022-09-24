package core

// Reference represents a single reference.
type Reference interface {
	Attributes() AttributeList // differa according the reference kind.
}

// ReferenceManager retrieves the metadata for references.
type ReferenceManager interface {

	// Search returns the best matching reference.
	Search(query string) (Reference, error)
}
