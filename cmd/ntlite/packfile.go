package main

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"gopkg.in/yaml.v3"
)

/* ObjectData */

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

/* PackFile */

type PackFile struct {
	OID              oid.OID       `yaml:"oid" json:"oid"`
	FileRelativePath string        `yaml:"file_relative_path" json:"file_relative_path"`
	FileMTime        time.Time     `yaml:"file_mtime" json:"file_mtime"`
	FileSize         int64         `yaml:"file_size" json:"file_size"`
	CTime            time.Time     `yaml:"ctime" json:"ctime"`
	PackObjects      []*PackObject `yaml:"objects" json:"objects"`
}

type PackObject struct {
	OID   oid.OID    `yaml:"oid" json:"oid"`
	Kind  string     `yaml:"kind" json:"kind"`
	CTime time.Time  `yaml:"ctime" json:"ctime"`
	Data  ObjectData `yaml:"data" json:"data"`
}

// AppendObject registers a new object inside the pack file.
func (p *PackFile) AppendObject(obj Object) error {
	data, err := NewObjectData(obj)
	if err != nil {
		return err
	}
	p.PackObjects = append(p.PackObjects, &PackObject{
		OID:   obj.UniqueOID(),
		Kind:  obj.Kind(),
		CTime: obj.ModificationTime(),
		Data:  data,
	})
	return nil
}

// ReadObject recreates the core object from a commit object.
func (p *PackObject) ReadObject() Object {
	switch p.Kind {
	case "file":
		file := new(File)
		p.Data.Unmarshal(file)
		return file
	case "note":
		note := new(Note)
		p.Data.Unmarshal(note)
		return note
	}
	return nil
}

// LoadPackFileFromPath reads a pack file file on disk.
func LoadPackFileFromPath(path string) (*PackFile, error) {
	in, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	result := new(PackFile)
	if err := result.Read(in); err != nil {
		return nil, err
	}
	in.Close()
	return result, nil
}

// Read populates a pack file from an object file.
func (p *PackFile) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(&p)
	if err != nil {
		return err
	}
	return nil
}

// Write dumps a pack file to an object file.
func (p *PackFile) Write(w io.Writer) error {
	data, err := yaml.Marshal(p)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// SaveTo writes a new pack file to the given location.
func (p *PackFile) Save() error {
	path := filepath.Join(CurrentRepository().Path, ".nt", "objects/"+p.OID.RelativePath(".pack"))
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return p.Write(f)
}

func NewPackFileFromParsedFile(parsedFile *ParsedFile) (*PackFile, error) {
	// Use the hash of the parsed file as OID (if the file changes = new oid.OID)
	packFileOID := oid.MustParse(Hash([]byte(parsedFile.Markdown.Content)))

	packFile := &PackFile{
		OID: packFileOID,

		// Init file properties
		FileRelativePath: parsedFile.RelativePath,
		FileMTime:        parsedFile.Markdown.MTime,
		FileSize:         parsedFile.Markdown.Size,

		// Init pack file properties
		CTime: time.Now(),
	}

	// Create objects
	var objects []Object

	// Process the File
	file, err := NewOrExistingFile(packFile, parsedFile)
	if err != nil {
		return nil, err
	}
	objects = append(objects, file)

	// Process the Note(s)
	for _, parsedNote := range parsedFile.Notes {
		note, err := NewOrExistingNote(packFile, file, parsedNote)
		if err != nil {
			return nil, err
		}
		objects = append(objects, note)
	}

	// Fill the pack file
	for _, obj := range objects {
		if statefulObj, ok := obj.(StatefulObject); ok {
			if err := packFile.AppendObject(statefulObj); err != nil {
				return nil, err
			}
		}
	}

	// Save the pack file on disk
	if err := packFile.Save(); err != nil {
		return nil, err
	}

	return packFile, nil
}
