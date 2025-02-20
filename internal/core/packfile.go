package core

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	godiffpatch "github.com/sourcegraph/go-diff-patch"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"gopkg.in/yaml.v3"
)

/*
 * ObjectData
 */

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
	if f, ok := target.(*Flashcard); ok {
		f.Read(dest)
		return nil
	}
	if f, ok := target.(*Study); ok {
		f.Read(dest)
		return nil
	}
	if m, ok := target.(*Media); ok {
		m.Read(dest)
		return nil
	}
	if l, ok := target.(*GoLink); ok {
		l.Read(dest)
		return nil
	}
	if r, ok := target.(*Reminder); ok {
		r.Read(dest)
		return nil
	}

	return fmt.Errorf("unsupported type %T", target)
}

/*
 * PackFile
 */

// NilPackFile implements the Null Object Pattern for PackFile.
var NilPackFile = &PackFile{
	OID:              oid.Nil,
	FileRelativePath: "",
	FileMTime:        time.Time{},
	FileSize:         0,
	CTime:            time.Time{},
	PackObjects:      nil,
	BlobRefs:         nil,
}

type PackFile struct {
	OID              oid.OID       `yaml:"oid" json:"oid"`
	FileRelativePath string        `yaml:"file_relative_path" json:"file_relative_path"`
	FileMTime        time.Time     `yaml:"file_mtime" json:"file_mtime"`
	FileSize         int64         `yaml:"file_size" json:"file_size"`
	CTime            time.Time     `yaml:"ctime" json:"ctime"`
	PackObjects      []*PackObject `yaml:"objects" json:"objects"`
	BlobRefs         []*BlobRef    `yaml:"blobs" json:"blobs"`
}

type PackObject struct {
	OID         oid.OID    `yaml:"oid" json:"oid"`
	Kind        string     `yaml:"kind" json:"kind"`
	CTime       time.Time  `yaml:"ctime" json:"ctime"`
	Description string     `yaml:"desc" json:"desc"`
	Data        ObjectData `yaml:"data" json:"data"`
}

// ReadObject recreates the core object from a commit object.
func (p *PackObject) ReadObject() Object {
	switch p.Kind {
	case "file":
		file := new(File)
		p.Data.Unmarshal(file)
		return file
	case "flashcard":
		flashcard := new(Flashcard)
		p.Data.Unmarshal(flashcard)
		return flashcard
	case "study":
		study := new(Study)
		p.Data.Unmarshal(study)
		return study
	case "note":
		note := new(Note)
		p.Data.Unmarshal(note)
		return note
	case "link":
		link := new(GoLink)
		p.Data.Unmarshal(link)
		return link
	case "media":
		media := new(Media)
		p.Data.Unmarshal(media)
		return media
	case "reminder":
		reminder := new(Reminder)
		p.Data.Unmarshal(reminder)
		return reminder
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

// Ref returns a ref to the pack file.
func (p *PackFile) Ref() PackFileRef {
	return PackFileRef{
		RelativePath: p.FileRelativePath,
		OID:          p.OID,
		CTime:        p.CTime,
	}
}

// GetPackObject retrieves an object from a pack file.
func (p *PackFile) GetPackObject(oid oid.OID) (*PackObject, bool) {
	for _, object := range p.PackObjects {
		if object.OID == oid {
			return object, true
		}
	}
	return nil, false
}

// AppendPackObject registers a new object inside the pack file.
func (p *PackFile) AppendPackObject(obj *PackObject) {
	p.PackObjects = append(p.PackObjects, obj)
}

// MustAppendObject registers a new object inside the pack file or panic.
func (p *PackFile) MustAppendObject(obj Object) {
	if err := p.AppendObject(obj); err != nil {
		panic(err)
	}
}

// AppendObject registers a new object inside the pack file.
func (p *PackFile) AppendObject(obj Object) error {
	data, err := NewObjectData(obj)
	if err != nil {
		return err
	}
	p.PackObjects = append(p.PackObjects, &PackObject{
		OID:         obj.UniqueOID(),
		Kind:        obj.Kind(),
		CTime:       obj.ModificationTime(),
		Description: obj.String(),
		Data:        data,
	})
	return nil
}

// AppendBlob registers a new blob inside the pack file.
func (p *PackFile) AppendBlob(blob *BlobRef) error {
	p.BlobRefs = append(p.BlobRefs, blob)
	return nil
}

// AppendBlobs registers new blobs inside the pack file.
func (p *PackFile) AppendBlobs(blobs []*BlobRef) error {
	p.BlobRefs = append(p.BlobRefs, blobs...)
	return nil
}

// UnmarshallObject extract a single object from a commit.
func (p *PackFile) UnmarshallObject(oid oid.OID, target interface{}) error {
	for _, objEdit := range p.PackObjects {
		if objEdit.OID == oid {
			return objEdit.Data.Unmarshal(target)
		}
	}
	return fmt.Errorf("no object with OID %q", oid)
}

// FindFirstBlobWithMimeType returns the first blob with the given mime type.
func (p *PackFile) FindFirstBlobWithMimeType(mimeType string) *BlobRef {
	for _, blob := range p.BlobRefs {
		if blob.MimeType == mimeType {
			return blob
		}
	}
	return nil
}

/* Object */

func (p *PackFile) UniqueOID() oid.OID {
	return p.OID
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

// Save writes a new pack file inside .nt/objects.
func (p *PackFile) Save() error {
	if CurrentConfig().DryRun {
		return nil
	}
	return p.SaveTo(PackFilePath(p.OID))
}

// ObjectPath returns the absolute path to the pack file in .nt/objects/ directory.
func (p *PackFile) ObjectPath() string {
	return PackFilePath(p.OID)
}

// ObjectRelativePath returns the relative path to the pack file inside .nt/ directory.
func (p *PackFile) ObjectRelativePath() string {
	return PackFileRelativePath(p.OID)
}

// PackFilePath returns the path to the pack file in .nt/objects/ directory.
func PackFilePath(oid oid.OID) string {
	return filepath.Join(CurrentConfig().RootDirectory, ".nt", PackFileRelativePath(oid))
}

// PackFileRelativePath returns the path to the pack file in .nt/objects/ directory.
func PackFileRelativePath(oid oid.OID) string {
	return "objects/" + oid.RelativePath() + ".pack"
}

// SaveTo writes a new pack file to the given location.
func (p *PackFile) SaveTo(path string) error {
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

/* Diff */

type ObjectDiff struct {
	Before ParsedObject
	After  ParsedObject
}

func (o *ObjectDiff) Kind() string {
	return o.AfterOrBefore().Kind()
}

func (o *ObjectDiff) RelativePath() string {
	return o.AfterOrBefore().FileRelativePath()
}

// Modified returns true if the object has been modified.
func (o *ObjectDiff) Modified() bool {
	return o.Before != nil && o.After != nil
}

// Added returns true if the object has been added.
func (o *ObjectDiff) Added() bool {
	return o.Before == nil && o.After != nil
}

// Deleted returns true if the object has been deleted.
func (o *ObjectDiff) Deleted() bool {
	return o.Before != nil && o.After == nil
}

// Patch returns a diff patch for the object.
func (o *ObjectDiff) Patch() string {
	changeDescription := fmt.Sprintf("%s [%s]", o.RelativePath(), o.Kind())
	before := ""
	after := ""
	if o.Before != nil {
		before = o.Before.ToYAML()
	}
	if o.After != nil {
		after = o.After.ToYAML()
	}
	patch := godiffpatch.GeneratePatch(
		changeDescription,
		before,
		after,
	)
	return patch
}

// AfterOrBefore returns the first non-nil object prefering the after one.
func (o *ObjectDiff) AfterOrBefore() ParsedObject {
	if o.After != nil {
		return o.After
	}
	return o.Before
}

// BeforeOrAfter returns the first non-nil object prefering the before one.
func (o *ObjectDiff) BeforeOrAfter() ParsedObject {
	if o.Before != nil {
		return o.Before
	}
	return o.After
}

// ObjectDiffs represents a list of ObjectDiff with helper methods to extract diff objects.
type ObjectDiffs []*ObjectDiff

// FindMedia returns the diff for a media.
func (d ObjectDiffs) FindMedia(relativePath string) *ObjectDiff {
	for _, diff := range d {
		if diff.Kind() == "media" && diff.RelativePath() == relativePath {
			return diff
		}
	}
	return nil
}

// FindFileByTitle returns the diff for a file with a given title.
func (d ObjectDiffs) FindFileByTitle(relativePath string, title string) *ObjectDiff {
	for _, diff := range d {
		if diff.Kind() == "file" && diff.RelativePath() == relativePath {
			file := diff.AfterOrBefore().(*File)
			if file.Title.String() == title {
				return diff
			}
		}
	}
	return nil
}

// FindNoteByTitle returns the diff for a note with a given title.
func (d ObjectDiffs) FindNoteByTitle(relativePath string, title string) *ObjectDiff {
	for _, diff := range d {
		if diff.Kind() == "note" && diff.RelativePath() == relativePath {
			note := diff.AfterOrBefore().(*Note)
			if note.Title.String() == title {
				return diff
			}
		}
	}
	return nil
}

// FindGoLinkByName returns the diff for a go link with a given name.
func (d ObjectDiffs) FindGoLinkByName(relativePath string, name string) *ObjectDiff {
	for _, diff := range d {
		if diff.Kind() == "link" && diff.RelativePath() == relativePath {
			goLink := diff.AfterOrBefore().(*GoLink)
			if goLink.GoName == name {
				return diff
			}
		}
	}
	return nil
}

// FindFlashcardByShortTitle returns the diff for a flashcard with a given title.
func (d ObjectDiffs) FindFlashcardByShortTitle(relativePath string, shortTitle string) *ObjectDiff {
	for _, diff := range d {
		if diff.Kind() == "flashcard" && diff.RelativePath() == relativePath {
			flashcard := diff.AfterOrBefore().(*Flashcard)
			if flashcard.ShortTitle.String() == shortTitle {
				return diff
			}
		}
	}
	return nil
}

// FindReminderWithTag returns the diff for a reminder matching a given tag.
func (d ObjectDiffs) FindReminderWithTag(relativePath string, tag string) *ObjectDiff {
	for _, diff := range d {
		if diff.Kind() == "reminder" && diff.RelativePath() == relativePath {
			reminder := diff.AfterOrBefore().(*Reminder)
			if reminder.Tag == tag {
				return diff
			}
		}
	}
	return nil
}

// Diff compares two pack files and returns the differences.
func (p *PackFile) Diff(other *PackFile) ObjectDiffs {
	var result ObjectDiffs

	for _, beforePackObject := range p.PackObjects {
		beforeObject := beforePackObject.ReadObject()
		beforeParsedObject, ok := beforeObject.(ParsedObject)
		if !ok {
			// We diff only objects extracted from Markdown files
			continue
		}
		afterPackObject, ok := other.GetPackObject(beforePackObject.OID)
		if ok {
			if beforePackObject.CTime.Equal(afterPackObject.CTime) {
				// Ignore object not changed between two pack files
				continue
			}

			afterObject := afterPackObject.ReadObject()
			if afterParsedObject, ok := afterObject.(ParsedObject); ok {
				// Compare both
				result = append(result, &ObjectDiff{
					Before: beforeParsedObject,
					After:  afterParsedObject,
				})
			}
		} else {
			// Deleted
			result = append(result, &ObjectDiff{
				Before: beforeParsedObject,
				After:  nil,
			})
		}
	}
	for _, afterPackObject := range other.PackObjects {
		_, ok := p.GetPackObject(afterPackObject.OID)
		if ok {
			// Already compared above
			continue
		}

		afterObject := afterPackObject.ReadObject()
		if afterParsedObject, ok := afterObject.(ParsedObject); ok {
			// Added
			result = append(result, &ObjectDiff{
				Before: nil,
				After:  afterParsedObject,
			})

		}
	}

	return result
}

/* Interface Dumpable */

func (p *PackFile) ToYAML() string {
	return ToBeautifulYAML(p)
}

func (p *PackFile) ToJSON() string {
	return ToBeautifulJSON(p)
}

func (p *PackFile) ToMarkdown() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# PackFile %s\n\n", p.OID))
	sb.WriteString("## Objects\n\n")
	for _, obj := range p.PackObjects {
		sb.WriteString(fmt.Sprintf("* %s: %s `@oid: %s`\n", obj.Kind, obj.Description, obj.OID))
	}
	return sb.String()
}

/*
 * PackFileRef
 */

type PackFileRef struct {
	OID          oid.OID   `yaml:"oid" json:"oid"`
	RelativePath string    `yaml:"relative_path" json:"relative_path"`
	CTime        time.Time `yaml:"ctime" json:"ctime"`
}

// ObjectOID returns the OID of the blob.
func (b PackFileRef) ObjectOID() oid.OID {
	return b.OID
}

// ObjectPath returns the absolute path to the pack file in .nt/objects/ directory.
func (p PackFileRef) ObjectPath() string {
	return PackFilePath(p.OID)
}

// ObjectRelativePath returns the relative path to the pack file inside .nt/ directory.
func (p PackFileRef) ObjectRelativePath() string {
	return PackFileRelativePath(p.OID)
}

// Convenient type to add methods
type PackFileRefs []PackFileRef

// OIDs returns the list of OIDs.
func (p PackFileRefs) OIDs() []oid.OID {
	var results []oid.OID
	for _, packFileRef := range p {
		results = append(results, packFileRef.OID)
	}
	return results
}

/* PackFile creation */

func NewPackFileFromParsedFile(parsedFile *ParsedFile) (*PackFile, error) {
	// Use the hash of the parsed file as OID (if the file changes = new oid.OID)
	packFileOID := oid.MustParse(parsedFile.Hash())

	// Check first if a previous execution already created the pack file
	// (ex: the command was aborted with Ctrl+C and restarted)
	existingPackFile, err := CurrentDB().ReadPackFileOnDisk(packFileOID)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if existingPackFile != nil {
		return existingPackFile, nil
	}

	packFile := &PackFile{
		OID: packFileOID,

		// Init file properties
		FileRelativePath: parsedFile.RelativePath,
		FileMTime:        parsedFile.Markdown.MTime,
		FileSize:         parsedFile.Markdown.Size,

		// Init pack file properties
		CTime: clock.Now(),
	}

	// Create objects
	var objects []Object

	// Process the File
	file, err := NewOrExistingFile(packFile, parsedFile)
	if err != nil {
		return nil, err
	}
	objects = append(objects, file)
	file.GenerateBlobs()

	// Process the Note(s)
	for _, parsedNote := range parsedFile.Notes {
		note, err := NewOrExistingNote(packFile, file, parsedNote)
		if err != nil {
			return nil, err
		}
		objects = append(objects, note)

		// Process the Flashcard
		if parsedNote.Flashcard != nil {
			parsedFlashcard := parsedNote.Flashcard
			flashcard, err := NewOrExistingFlashcard(packFile, file, note, parsedFlashcard)
			if err != nil {
				return nil, err
			}
			objects = append(objects, flashcard)
		}

		// Process the Reminder(s)
		for _, parsedReminder := range parsedNote.Reminders {
			reminder, err := NewOrExistingReminder(packFile, note, parsedReminder)
			if err != nil {
				return nil, err
			}
			objects = append(objects, reminder)
		}

		// Process the Golink(s)
		for _, parsedGoLink := range parsedNote.GoLinks {
			goLink, err := NewOrExistingGoLink(packFile, note, parsedGoLink)
			if err != nil {
				return nil, err
			}
			objects = append(objects, goLink)
		}
	}

	// Fill the pack file
	for _, obj := range objects {
		if statefulObj, ok := obj.(StatefulObject); ok {
			if err := packFile.AppendObject(statefulObj); err != nil {
				return nil, err
			}
		}
		if fileObj, ok := obj.(FileObject); ok {
			if err := packFile.AppendBlobs(fileObj.Blobs()); err != nil {
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

func NewPackFileFromParsedMedia(parsedMedia *ParsedMedia) (*PackFile, error) {
	packFileOID := oid.New()
	if !parsedMedia.Dangling {
		// Use the hash of the raw original media as OID (if the media is even slightly edited = new oid.OID)
		packFileOID = oid.MustParse(parsedMedia.FileHash())
	}

	// Check first if a previous execution already created the pack file
	// (ex: the command was aborted with Ctrl+C and restarted)
	existingPackFile, err := CurrentDB().ReadPackFileOnDisk(packFileOID)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if existingPackFile != nil {
		return existingPackFile, nil
	}

	packFile := &PackFile{
		OID: packFileOID,

		// Init file properties
		FileRelativePath: parsedMedia.RelativePath,
		FileMTime:        parsedMedia.MTime,
		FileSize:         parsedMedia.Size,

		// Init pack file properties
		CTime: clock.Now(),
	}

	// Process the Media
	media, err := NewOrExistingMedia(packFile, parsedMedia)
	if err != nil {
		return nil, err
	}
	media.GenerateBlobs()

	// Fill the pack file
	if err := packFile.AppendObject(media); err != nil {
		return nil, err
	}
	if err := packFile.AppendBlobs(media.Blobs()); err != nil {
		return nil, err
	}

	// Save the pack file on disk
	if err := packFile.Save(); err != nil {
		return nil, err
	}

	return packFile, nil
}
