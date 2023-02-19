package core

type Blob struct {
	// OID to locate the blob file in .nt/objects
	OID        string
	Attributes map[string]interface{}
	Tags       []string
}

func (b *Blob) Hash() string {
	// TODO
	return ""
}

type Object interface {
	OID() string
	ToObject() string
	Blobs() []Blob // OID size tags
}



// Same for other objects

// Command Add
// _file_ = read the file
// _path_ = traverse the path
// . = traverse the work tree
// Same as current Build() but:
// - create blobs in `.nt/objects/`
// - append to `.nt/index`:
