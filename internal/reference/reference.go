package reference

type Attribute struct {
	Key   string
	Value interface{}
}

// Reference represents a single reference.
type Reference interface {
	Attributes() []Attribute // differs according the reference kind.
}

// Manager retrieves the metadata for references.
type Manager interface {

	// Search returns the best matching reference.
	Search(query string) (Reference, error)
}
