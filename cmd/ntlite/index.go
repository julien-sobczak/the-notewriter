package main

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Lite version of internal/core/index.go

// State describes an object status.
type State string

const (
	None     State = "none"
	Added    State = "added"
	Modified State = "modified"
	Deleted  State = "deleted"
)

/* Index */

type Index struct {
	Objects     []*IndexObject   `yaml:"objects"`
	StagingArea []*StagingObject `yaml:"staging"`
}

type IndexObject struct {
	OID   string    `yaml:"oid"`
	Kind  string    `yaml:"kind"`
	MTime time.Time `yaml:"mtime"`
}

type StagingObject struct {
	IndexObject
	State State      `yaml:"state"`
	Data  ObjectData `yaml:"data"`
}

// ReadIndex loads the index file.
func ReadIndex() *Index {
	path := filepath.Join(CurrentCollection().Path, ".nt/index")
	in, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		// First use
		return &Index{}
	}
	if err != nil {
		log.Fatalf("Unable to open index: %v", err)
	}
	index := new(Index)
	if err := index.Read(in); err != nil {
		log.Fatalf("Unable to read index: %v", err)
	}
	in.Close()
	return index
}

// Save persists the index on disk.
func (i *Index) Save() error {
	path := filepath.Join(CurrentCollection().Path, ".nt/index")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return i.Write(f)
}

// Read reads an index from the file.
func (i *Index) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(&i)
	if err != nil {
		return err
	}
	return nil
}

// Write dumps the index to a file.
func (i *Index) Write(w io.Writer) error {
	data, err := yaml.Marshal(i)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// StageObject registers a changed object into the staging area
func (i *Index) StageObject(obj StatefulObject) error {
	objData, err := NewObjectData(obj)
	if err != nil {
		return err
	}

	// Update staging area
	stagingObject := &StagingObject{
		IndexObject: IndexObject{
			OID:   obj.UniqueOID(),
			Kind:  obj.Kind(),
			MTime: obj.ModificationTime(),
		},
		State: obj.State(),
		Data:  objData,
	}

	i.StagingArea = append(i.StagingArea, stagingObject)

	return nil
}

// ClearStagingArea empties the staging area.
func (i *Index) ClearStagingArea() {
	for _, obj := range i.StagingArea {
		i.Objects = append(i.Objects, &obj.IndexObject)
	}
	i.StagingArea = nil
}

// ObjectData serializes any Object to base64 after zlib compression.
type ObjectData []byte // alias to serialize to YAML easily

// NewObjectData creates a compressed-string representation of the object.
func NewObjectData(obj Object) (ObjectData, error) {
	b := new(bytes.Buffer)
	if err := obj.Write(b); err != nil {
		return nil, err
	}
	in := b.Bytes()

	zb := new(bytes.Buffer)
	w := zlib.NewWriter(zb)
	w.Write(in)
	w.Close()
	return ObjectData(zb.Bytes()), nil
}

func (od ObjectData) MarshalYAML() (interface{}, error) {
	return base64.StdEncoding.EncodeToString(od), nil
}

func (od *ObjectData) UnmarshalYAML(node *yaml.Node) error {
	value := node.Value
	ba, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return err
	}
	*od = ba
	return nil
}

func (od ObjectData) Unmarshal(target interface{}) error {
	if target == nil {
		return fmt.Errorf("cannot unmarshall in nil target")
	}
	src := bytes.NewReader(od)
	dest := new(bytes.Buffer)
	r, err := zlib.NewReader(src)
	if err != nil {
		return err
	}
	io.Copy(dest, r)
	r.Close()

	if f, ok := target.(*File); ok {
		f.Read(dest)
		return nil
	}
	if n, ok := target.(*Note); ok {
		n.Read(dest)
		return nil
	}
	return fmt.Errorf("unsupported type %T", target)
}
