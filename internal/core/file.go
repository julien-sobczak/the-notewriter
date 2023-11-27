package core

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/helpers"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

// Default indentation in front matter
const Indent int = 2

type Attribute struct {
	Key   string
	Value interface{}
}

type File struct {
	// A unique identifier among all files
	OID string `yaml:"oid"`
	// A unique human-friendly slug
	Slug string `yaml:"slug"`

	// Optional parent file (= index.md)
	ParentFileOID string `yaml:"file_oid,omitempty"`
	ParentFile    *File  `yaml:"-"` // Lazy-loaded

	// A relative path to the collection directory
	RelativePath string `yaml:"relative_path"`
	// The full wikilink to this file (without the extension)
	Wikilink string `yaml:"wikilink"`

	// The FrontMatter for the note file
	FrontMatter *yaml.Node `yaml:"front_matter"`

	// Merged attributes
	Attributes map[string]interface{} `yaml:"attributes,omitempty"`

	// Original title of the main heading without leading # characters
	Title string `yaml:"title,omitempty"`
	// Short title of the main heading without the kind prefix if present
	ShortTitle string `yaml:"short_title,omitempty"`

	Body     string  `yaml:"body"`
	BodyLine int     `yaml:"body_line"`
	notes    []*Note `yaml:"-"`

	// Permission of the file (required to save back)
	Mode fs.FileMode `yaml:"mode"`
	// Size of the file (can be useful to detect changes)
	Size int64 `yaml:"size"`
	// Hash of the content (can be useful to detect changes too)
	Hash string `yaml:"hash"`
	// Content last modification date
	MTime time.Time `yaml:"mtime"`

	CreatedAt     time.Time `yaml:"created_at"`
	UpdatedAt     time.Time `yaml:"updated_at"`
	DeletedAt     time.Time `yaml:"deleted_at,omitempty"`
	LastCheckedAt time.Time `yaml:"-"`

	new   bool
	stale bool
}

func NewOrExistingFile(parent *File, path string) (*File, error) {
	relpath, err := CurrentCollection().GetFileRelativePath(path)
	if err != nil {
		log.Fatal(err)
	}
	existingFile, err := CurrentCollection().FindFileByRelativePath(relpath)
	if err != nil {
		log.Fatal(err)
	}

	if existingFile != nil {
		existingFile.update(parent)
		return existingFile, nil
	}

	return NewFileFromPath(parent, path)
}

/* Creation */

func NewEmptyFile(name string) *File {
	return &File{
		OID:          NewOID(),
		Slug:         "",
		stale:        true,
		new:          true,
		Wikilink:     name,
		RelativePath: name,
		Attributes:   make(map[string]interface{}),
	}
}

func NewFileFromAttributes(parent *File, name string, attributes []Attribute) *File {
	file := NewEmptyFile(name)
	if parent != nil {
		file.ParentFile = parent
		file.ParentFileOID = parent.OID
		file.Attributes = parent.GetAttributes()
	}
	for _, attribute := range attributes {
		file.SetAttribute(attribute.Key, attribute.Value)
	}
	return file
}

func NewFileFromParsedFile(parent *File, parsedFile *ParsedFile) *File {
	file := &File{
		OID:          NewOID(),
		Slug:         parsedFile.Slug,
		RelativePath: parsedFile.RelativePath,
		Wikilink:     text.TrimExtension(parsedFile.RelativePath),
		Mode:         parsedFile.LStat.Mode(),
		Size:         parsedFile.LStat.Size(),
		Hash:         helpers.Hash(parsedFile.Bytes),
		MTime:        parsedFile.LStat.ModTime(),
		Attributes:   make(map[string]interface{}),
		FrontMatter:  parsedFile.FrontMatter,
		Body:         parsedFile.Body,
		BodyLine:     parsedFile.BodyLine,
		CreatedAt:    clock.Now(),
		UpdatedAt:    clock.Now(),
		stale:        true,
		new:          true,
	}
	if parent != nil {
		file.ParentFileOID = parent.OID
		file.ParentFile = parent
	}
	newAttributes := parsedFile.FileAttributes
	if parent != nil {
		newAttributes = file.mergeAttributes(parent.GetAttributes(), newAttributes)
	}
	file.Attributes = newAttributes

	return file
}

func NewFileFromPath(parent *File, filepath string) (*File, error) {
	parsedFile, err := ParseFile(filepath)
	if err != nil {
		return nil, err
	}
	return NewFileFromParsedFile(parent, parsedFile), nil
}

func (f *File) mergeAttributes(attributes ...map[string]interface{}) map[string]interface{} {
	// File attributes are always inheritable to top level-notes
	// (NB: `source` is configured to be non-inheritable).
	//
	// Ex:
	//   ---
	//   source: XXX
	//   ---
	//   # Example
	//   ## Note: Parent
	//   ### Note: Child
	//
	// Is the same as:
	//
	//   # Example
	//   ## Note: Parent
	//   `@source:XXX`
	//   ### Note: Child
	return MergeAttributes(attributes...)
}

/* Object */

func (f *File) Kind() string {
	return "file"
}

func (f *File) UniqueOID() string {
	return f.OID
}

func (f *File) Refresh() (bool, error) {
	// No dependencies = no need to refresh
	return false, nil
}

func (f *File) State() State {
	if !f.DeletedAt.IsZero() {
		return Deleted
	}
	if f.new {
		return Added
	}
	if f.stale {
		return Modified
	}
	return None
}

func (f *File) ForceState(state State) {
	switch state {
	case Added:
		f.new = true
	case Deleted:
		f.DeletedAt = clock.Now()
	}
	f.stale = true
}

func (f *File) SetAlive() {
	f.DeletedAt = clock.Now()
	f.stale = true
}

func (f *File) ModificationTime() time.Time {
	return f.MTime
}

func (f *File) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(f)
	if err != nil {
		return err
	}
	return nil
}

func (f *File) Write(w io.Writer) error {
	data, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (f *File) SubObjects() []StatefulObject {
	var objs []StatefulObject
	for _, object := range f.GetNotes() {
		objs = append(objs, object)
		objs = append(objs, object.SubObjects()...)
	}
	for _, object := range f.GetFlashcards() {
		objs = append(objs, object)
		objs = append(objs, object.SubObjects()...)
	}
	// Medias are already saved through files
	// for _, object := range f.GetMedias() {
	// 	objs = append(objs, object)
	// 	objs = append(objs, object.SubObjects()...)
	// }
	return objs
}

func (f *File) Blobs() []*BlobRef {
	// Use Media.Blobs() instead
	return nil
}

func (f *File) Relations() []*Relation {
	// We consider only relations related to notes
	return nil
}

func (f File) String() string {
	return fmt.Sprintf("file %q [%s]", f.RelativePath, f.OID)
}

/* Update */

func (f *File) update(parent *File) error {
	abspath := CurrentCollection().GetAbsolutePath(f.RelativePath)
	parsedFile, err := ParseFile(abspath)
	if err != nil {
		return err
	}

	newAttributes := parsedFile.FileAttributes
	if parent != nil {
		newAttributes = f.mergeAttributes(parent.GetAttributes(), newAttributes)
	}

	// Check if attributes have changed
	if !reflect.DeepEqual(newAttributes, f.Attributes) {
		f.stale = true
		f.Attributes = newAttributes
	}

	// Check if local file has changed
	if f.MTime != parsedFile.LStat.ModTime() || f.Size != parsedFile.LStat.Size() {
		// file change
		f.stale = true

		f.Mode = parsedFile.LStat.Mode()
		f.Size = parsedFile.LStat.Size()
		f.Hash = helpers.Hash(parsedFile.Bytes)
		if parsedFile.FrontMatter.Kind > 0 {
			f.FrontMatter = parsedFile.FrontMatter
		} else {
			f.FrontMatter = nil
		}
		f.Attributes = parsedFile.FileAttributes
		if parent != nil {
			f.Attributes = f.mergeAttributes(parent.GetAttributes(), f.Attributes)
		}
		f.MTime = parsedFile.LStat.ModTime()
		f.Body = parsedFile.Body
		f.BodyLine = parsedFile.BodyLine
	}

	return nil
}

/* State Management */

func (f *File) New() bool {
	return f.new
}

func (f *File) Updated() bool {
	return f.stale
}

/* Front Matter */

// AbsoluteBodyLine returns the line number in the file by taking into consideration the front matter.
func (f *File) AbsoluteBodyLine(bodyLine int) int {
	return f.BodyLine + bodyLine - 1
}

// FrontMatterString formats the current attributes to the YAML front matter format.
func (f *File) FrontMatterString() (string, error) {
	var buf bytes.Buffer
	bufEncoder := yaml.NewEncoder(&buf)
	bufEncoder.SetIndent(Indent)
	err := bufEncoder.Encode(f.FrontMatter)
	if err != nil {
		return "", err
	}
	return CompactYAML(buf.String()), nil
}

// GetAttributes returns all file-specific and inherited attributes.
func (f *File) GetAttributes() map[string]interface{} {
	return f.Attributes
}

// GetAttribute extracts a single attribute value at the top.
func (f *File) GetAttribute(key string) interface{} {
	value, ok := f.Attributes[key]
	if !ok {
		return nil
	}
	return value
}

// SetAttribute overrides or defines a single attribute.
func (f *File) SetAttribute(key string, value interface{}) {
	if f.FrontMatter == nil {
		var frontMatterContent []*yaml.Node
		f.FrontMatter = &yaml.Node{
			Kind:    yaml.MappingNode,
			Content: frontMatterContent,
		}
	}

	found := false
	for i := 0; i < len(f.FrontMatter.Content)/2; i++ {
		keyNode := f.FrontMatter.Content[i*2]
		valueNode := f.FrontMatter.Content[i*2+1]
		if keyNode.Value != key {
			continue
		}

		found = true

		newValueNode := ToSafeYAMLNode(value)
		if newValueNode.Kind == yaml.ScalarNode {
			valueNode.Value = newValueNode.Value
		} else if newValueNode.Kind == yaml.DocumentNode {
			valueNode.Content = newValueNode.Content[0].Content
		} else {
			valueNode.Content = newValueNode.Content
		}
	}

	if !found {
		f.FrontMatter.Content = append(f.FrontMatter.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: key,
		})
		newValueNode := ToSafeYAMLNode(value)
		switch newValueNode.Kind {
		case yaml.DocumentNode:
			f.FrontMatter.Content = append(f.FrontMatter.Content, newValueNode.Content[0])
		case yaml.ScalarNode:
			f.FrontMatter.Content = append(f.FrontMatter.Content, newValueNode)
		default:
			fmt.Printf("Unexpected type %v\n", newValueNode.Kind)
			os.Exit(1)
		}
	}

	// Don't forget to append in parsed attributes too
	newAttributes := map[string]interface{}{key: value}
	newAttributes = CastAttributes(newAttributes, GetSchemaAttributeTypes())
	f.Attributes = MergeAttributes(f.Attributes, newAttributes)
}

// GetTags returns all defined tags.
func (f *File) GetTags() []string {
	value := f.GetAttribute("tags")
	if tag, ok := value.(string); ok {
		return []string{tag}
	}
	if tags, ok := value.([]string); ok {
		return tags
	}
	if rawTags, ok := value.([]interface{}); ok {
		var tags []string
		for _, rawTag := range rawTags {
			if tag, ok := rawTag.(string); ok {
				tags = append(tags, tag)
			}
		}
		return tags
	}
	return nil
}

// HasTag returns if a file has a given tag.
func (f *File) HasTag(name string) bool {
	return slices.Contains(f.GetTags(), name)
}

/* Content */

func (f *File) GetNotes() []*Note {
	if f.notes != nil {
		return f.notes
	}

	parsedNotes := ParseNotes(f.Body, f.Slug)

	if len(parsedNotes) == 0 {
		return nil
	}

	// Collect parent indices
	parentNoteIndices := make(map[int]int)
	for i, currentNote := range parsedNotes {
		found := false
		for j, prevNote := range parsedNotes[0:i] {
			if prevNote.Level == currentNote.Level-1 {
				found = true
				parentNoteIndices[i] = j
			}
		}
		if !found {
			parentNoteIndices[i] = -1
		}
	}

	// We sort notes to process them according their dependencies.
	// For example, if a note includes another note in the same file
	// (NB: external dependencies are addressed elsewhere when processing files),
	// we must return the included note first for it to be saved first in database,
	// so that when we will build the final note content for the other note,
	// the dependency will be found in database.
	var sortedParsedNotes []*ParsedNote
	addedNoteIndices := make(map[int]bool)
	addedSections := make(map[string]bool)
	changedDuringIteration := false
	for len(addedNoteIndices) < len(parsedNotes) { // until all notes are added or no more notes can be added due to transitive dependency
		for i, note := range parsedNotes {
			if addedNoteIndices[i] {
				// Already added
				continue
			}

			var internalWikilinks []*Wikilink
			for _, wikilink := range note.Wikilinks() {
				if wikilink.Internal() {
					internalWikilinks = append(internalWikilinks, wikilink)
				}
			}

			// A note can be added iff:
			// - no parent OR the parent note has already been added
			// - no internal link OR all notes referenced by internal links has been added first
			parentSatisfied := parentNoteIndices[i] == -1 || addedNoteIndices[parentNoteIndices[i]]
			internalLinksSatisfied := true
			for _, wikilink := range internalWikilinks {
				if _, ok := addedSections[wikilink.Section()]; !ok {
					internalLinksSatisfied = false
				}
			}

			if parentSatisfied && internalLinksSatisfied {
				addedNoteIndices[i] = true
				addedSections[note.Title] = true
				sortedParsedNotes = append(sortedParsedNotes, note)
				changedDuringIteration = true
			}
		}
		if !changedDuringIteration {
			// cyclic dependency found
			CurrentLogger().Info("Cyclic dependency between notes detected. Incomplete note(s) can result.")
			// Add remaining notes without taking care of dependencies...
			for i, note := range parsedNotes {
				if addedNoteIndices[i] {
					// Already added
					continue
				}
				sortedParsedNotes = append(sortedParsedNotes, note)
			}
			break
		}
		changedDuringIteration = false
	}

	// All notes collected until now
	var notes []*Note

	for i, currentNote := range sortedParsedNotes {
		var parent *Note
		if parentNoteIndices[i] != -1 {
			parent = notes[parentNoteIndices[i]]
		}
		note := NewOrExistingNote(f, parent, currentNote)
		if note.HasTag("ignore") {
			// Do not add notes marked as ignorable
			continue
		}
		notes = append(notes, note)
	}

	if len(notes) > 0 {
		f.notes = notes
	}
	return f.notes
}

// ParsedNote represents a single raw note inside a file.
type ParsedNote struct {
	Level          int
	Kind           NoteKind
	Slug           string
	Title          string
	ShortTitle     string
	Line           int
	Body           string
	NoteAttributes map[string]interface{}
	NoteTags       []string
}

// MustParseNote is pratical in unit test to setup a new note.
func MustParseNote(noteContent string, fileSlug string) *ParsedNote {
	notes := ParseNotes(noteContent, fileSlug)
	if len(notes) != 1 {
		log.Fatalf("Must only contain a single note. Found %d note(s)", len(notes))
	}
	return notes[0]
}

// ParseNotes extracts the notes from a file body.
func ParseNotes(fileBody string, fileSlug string) []*ParsedNote {
	type Section struct {
		level      int
		kind       NoteKind
		title      string
		shortTitle string
		lineNumber int
	}
	var sections []*Section

	// Extract all sections
	lines := strings.Split(fileBody, "\n")

	// Check if the file contains typed notes.
	// If so, it means the top heading (= the title of file) does not represent a free note.
	// Otherwise, we will add this top heading as a standalone note.
	ignoreTopHeading := false
	insideCodeBlock := false
	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			insideCodeBlock = !insideCodeBlock
		}
		if insideCodeBlock {
			// Ignore possible Markdown heading in code blocks
			continue
		}
		if ok, longTitle, level := markdown.IsHeading(line); ok {
			if ok, kind, _ := isSupportedNote(longTitle); ok {
				if level != 1 && kind != KindFree {
					ignoreTopHeading = true
					break
				}
			}
		}
	}

	// Current line number during the parsing
	var lineNumber int
	insideNote := false
	insideCodeBlock = false
	for _, line := range lines {
		lineNumber++
		if strings.HasPrefix(line, "```") {
			insideCodeBlock = !insideCodeBlock
		}
		if insideCodeBlock {
			// Ignore possible Markdown heading in code blocks
			continue
		}
		if ok, title, level := markdown.IsHeading(line); ok {
			if level == 1 && ignoreTopHeading {
				continue
			}
			lastLevel := 0
			if len(sections) > 0 {
				lastLevel = sections[len(sections)-1].level
			}
			if level <= lastLevel {
				insideNote = false
			}
			ok, kind, shortTitle := isSupportedNote(title)
			if ok {
				sections = append(sections, &Section{
					level:      level,
					kind:       kind,
					title:      title,
					shortTitle: shortTitle,
					lineNumber: lineNumber,
				})
				insideNote = true
			} else { // block inside a note or a free note?
				if !insideNote { // new free note
					sections = append(sections, &Section{
						level:      level,
						kind:       KindFree,
						title:      title,
						shortTitle: shortTitle,
						lineNumber: lineNumber,
					})
					insideNote = true
				}
			}
		}
	}

	// Iterate over sections and use line numbers to split the raw content into notes
	if len(sections) == 0 {
		return nil
	}

	// All notes collected until now
	var notes []*ParsedNote
	for i, section := range sections {
		var nextSection *Section
		if i < len(sections)-1 {
			nextSection = sections[i+1]
		}

		lineStart := section.lineNumber + 1
		lineEnd := -1 // EOF
		if nextSection != nil {
			lineEnd = nextSection.lineNumber - 1
		}

		noteContent := text.ExtractLines(fileBody, lineStart, lineEnd)
		if text.IsBlank(noteContent) {
			// skip sections without text (= category to organize notes, not really free notes)
			continue
		}

		tags, attributes := ExtractBlockTagsAndAttributes(noteContent)

		// Determine slug from attribute or define a default one otherwise
		slug := markdown.Slug(fileSlug, string(section.kind), section.shortTitle)
		if value, ok := attributes["slug"]; ok {
			if v, ok := value.(string); ok {
				slug = v
			}
		}

		parsedNote := &ParsedNote{
			Level:          section.level,
			Kind:           section.kind,
			Slug:           slug,
			Title:          section.title,
			ShortTitle:     section.shortTitle,
			Line:           section.lineNumber,
			NoteAttributes: CastAttributes(attributes, GetSchemaAttributeTypes()),
			NoteTags:       tags,
			Body:           noteContent,
		}
		notes = append(notes, parsedNote)
	}

	return notes
}

// Hash returns the current hash to use when searching for an existing note in database to avoid recreating it.
func (n *ParsedNote) Hash() string {
	raw := strings.TrimSpace(n.Body)
	hash := helpers.Hash([]byte(raw))
	return hash
}

// Wikilinks returns the wikilinks present in the note.
func (n *ParsedNote) Wikilinks() []*Wikilink {
	return ParseWikilinks(n.Body)
}

// FindNoteByKindAndShortTitle searches for a given note based on its kind and title.
func (f *File) FindNoteByKindAndShortTitle(kind NoteKind, shortTitle string) *Note {
	for _, note := range f.GetNotes() {
		if note.NoteKind == kind && note.ShortTitle == shortTitle {
			return note
		}
	}
	return nil
}

// FindFlashcardByTitle searches for a given flashcard based on its title.
func (f *File) FindFlashcardByTitle(shortTitle string) *Flashcard {
	for _, flashcard := range f.GetFlashcards() {
		if flashcard.ShortTitle == shortTitle {
			return flashcard
		}
	}
	return nil
}

// GetFlashcards extracts flashcards from the file.
func (f *File) GetFlashcards() []*Flashcard {
	var flashcards []*Flashcard
	for _, note := range f.GetNotes() {
		if note.NoteKind != KindFlashcard {
			continue
		}

		flashcard := NewOrExistingFlashcard(f, note)
		flashcards = append(flashcards, flashcard)
	}
	return flashcards
}

// GetMedias extracts medias from the file.
func (f *File) GetMedias() []*Media {
	return extractMediasFromMarkdown(f.RelativePath, f.Body)
}

/* Parsing */

type ParsedFile struct {
	// The paths to the file
	AbsolutePath string
	RelativePath string

	// Stat
	Stat  fs.FileInfo
	LStat fs.FileInfo

	// The raw file bytes
	Bytes []byte

	// Main Heading
	Slug       string
	Title      string
	ShortTitle string

	// The YAML Front Matter
	FrontMatter *yaml.Node
	// File attributes extracted from the Front Matter
	FileAttributes map[string]interface{}

	// The body (= content minus the front matter)
	Body     string
	BodyLine int
}

// ParseFile contains the main logic to parse a raw note file.
func ParseFile(path string) (*ParsedFile, error) {
	CurrentLogger().Debugf("Parsing file %s...", path)

	relativePath, err := CurrentCollection().GetFileRelativePath(path)
	if err != nil {
		return nil, err
	}
	absolutePath := CurrentCollection().GetAbsolutePath(relativePath)

	lstat, err := os.Lstat(absolutePath)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(absolutePath)
	if err != nil {
		return nil, err
	}

	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var rawFrontMatter bytes.Buffer
	var rawContent bytes.Buffer
	frontMatterStarted := false
	frontMatterEnded := false
	bodyStarted := false
	bodyStartLineNumber := 0
	for i, line := range strings.Split(strings.TrimSuffix(string(contentBytes), "\n"), "\n") {
		if strings.HasPrefix(line, "---") {
			if bodyStarted {
				// Flashcard Front/Back line separator
				rawContent.WriteString(line)
				rawContent.WriteString("\n")
			} else if !frontMatterStarted {
				frontMatterStarted = true
			} else if !frontMatterEnded {
				frontMatterEnded = true
			}
			continue
		}

		if frontMatterStarted && !frontMatterEnded {
			rawFrontMatter.WriteString(line)
			rawFrontMatter.WriteString("\n")
		} else {
			if !text.IsBlank(line) && !bodyStarted {
				bodyStarted = true
				bodyStartLineNumber = i + 1
			}
			rawContent.WriteString(line)
			rawContent.WriteString("\n")
		}
	}

	var frontMatter = new(yaml.Node)
	err = yaml.Unmarshal(rawFrontMatter.Bytes(), frontMatter)
	if err != nil {
		return nil, err
	}
	if frontMatter.Kind > 0 { // Happen when no Front Matter is present
		frontMatter = frontMatter.Content[0]
	}

	var attributes = make(map[string]interface{})
	err = yaml.Unmarshal(rawFrontMatter.Bytes(), attributes)
	if err != nil {
		return nil, err
	}

	body := strings.TrimSpace(rawContent.String())
	// Extract title
	title := ""
	for _, line := range strings.Split(body, "\n") {
		if ok, longTitle, _ := markdown.IsHeading(line); ok {
			title = longTitle
		}
	}
	_, _, shortTitle := isSupportedNote(title)

	// Extract/Generate slug
	slug := markdown.Slug(text.TrimExtension(filepath.Base(relativePath)))
	if value, ok := attributes["slug"]; ok {
		if v, ok := value.(string); ok {
			slug = v
		}
	}

	return &ParsedFile{
		AbsolutePath:   absolutePath,
		RelativePath:   relativePath,
		Stat:           stat,
		LStat:          lstat,
		Slug:           slug,
		Title:          title,
		ShortTitle:     shortTitle,
		Bytes:          contentBytes,
		FrontMatter:    frontMatter,
		FileAttributes: CastAttributes(attributes, GetSchemaAttributeTypes()),
		Body:           body,
		BodyLine:       bodyStartLineNumber,
	}, nil
}

// GetTags returns all defined tags on file.
func (f *ParsedFile) GetTags() []string {
	value, ok := f.FileAttributes["tags"]
	if !ok {
		return nil
	}
	if tag, ok := value.(string); ok {
		return []string{tag}
	}
	if tags, ok := value.([]string); ok {
		return tags
	}
	if rawTags, ok := value.([]interface{}); ok {
		var tags []string
		for _, rawTag := range rawTags {
			if tag, ok := rawTag.(string); ok {
				tags = append(tags, tag)
			}
		}
		return tags
	}
	return nil
}

// HasTag returns if the file has specifically a given tag.
func (f *ParsedFile) HasTag(tagName string) bool {
	return slices.Contains(f.GetTags(), tagName)
}

// Content returns the raw file content.
func (f *ParsedFile) Content() string {
	return string(f.Bytes)
}

// Wikilinks returns the wikilinks present inside a file.
func (f *ParsedFile) Wikilinks() []*Wikilink {
	return ParseWikilinks(f.Content())
}

// AbsoluteBodyLine returns the line number in the file by taking into consideration the front matter.
func (f *ParsedFile) AbsoluteBodyLine(bodyLine int) int {
	return f.BodyLine + bodyLine - 1
}

/* Data Management */

func (f *File) SaveOnDisk() error {
	// Persist to disk
	frontMatter, err := f.FrontMatterString()
	if err != nil {
		return err
	}
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(frontMatter)
	sb.WriteString("---\n")
	sb.WriteString(f.Body)

	if f.RelativePath == "" {
		return errors.New("unable to save file as no path is defined")
	}
	rawContent := []byte(sb.String())
	absolutePath := CurrentCollection().GetAbsolutePath(f.RelativePath)
	os.WriteFile(absolutePath, rawContent, f.Mode)

	// Refresh file-specific attributes
	stat, err := os.Lstat(absolutePath)
	if err != nil {
		return err
	}
	f.Size = stat.Size()
	f.Mode = stat.Mode()
	f.Hash = helpers.Hash(rawContent)

	return nil
}

func (f *File) Check() error {
	client := CurrentDB().Client()
	CurrentLogger().Debugf("Checking file %s...", f.RelativePath)
	f.LastCheckedAt = clock.Now()
	query := `
		UPDATE file
		SET last_checked_at = ?
		WHERE oid = ?;`
	if _, err := client.Exec(query, timeToSQL(f.LastCheckedAt), f.OID); err != nil {
		return err
	}
	query = `
		UPDATE note
		SET last_checked_at = ?
		WHERE file_oid = ?;`
	if _, err := client.Exec(query, timeToSQL(f.LastCheckedAt), f.OID); err != nil {
		return err
	}
	query = `
		UPDATE flashcard
		SET last_checked_at = ?
		WHERE file_oid = ?;`
	if _, err := client.Exec(query, timeToSQL(f.LastCheckedAt), f.OID); err != nil {
		return err
	}
	query = `
		UPDATE reminder
		SET last_checked_at = ?
		WHERE file_oid = ?;`
	if _, err := client.Exec(query, timeToSQL(f.LastCheckedAt), f.OID); err != nil {
		return err
	}

	return nil
}

func (f *File) Save() error {
	var err error
	f.UpdatedAt = clock.Now()
	f.LastCheckedAt = clock.Now()
	switch f.State() {
	case Added:
		err = f.Insert()
	case Modified:
		err = f.Update()
	case Deleted:
		err = f.Delete()
	default:
		err = f.Check()
	}
	if err != nil {
		return err
	}
	f.new = false
	f.stale = false
	return nil
}

func (f *File) Insert() error {
	CurrentLogger().Debugf("Inserting file %s...", f.RelativePath)
	query := `
		INSERT INTO file(
			oid,
			file_oid,
			slug,
			relative_path,
			wikilink,
			front_matter,
			attributes,
			title,
			short_title,
			body,
			body_line,
			created_at,
			updated_at,
			last_checked_at,
			mtime,
			size,
			hashsum,
			mode
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	frontMatter, err := f.FrontMatterString()
	if err != nil {
		return err
	}
	attributesJSON, err := AttributesJSON(f.Attributes)
	if err != nil {
		return err
	}

	_, err = CurrentDB().Client().Exec(query,
		f.OID,
		f.ParentFileOID,
		f.Slug,
		f.RelativePath,
		f.Wikilink,
		frontMatter,
		attributesJSON,
		f.Title,
		f.ShortTitle,
		f.Body,
		f.BodyLine,
		timeToSQL(f.CreatedAt),
		timeToSQL(f.UpdatedAt),
		timeToSQL(f.LastCheckedAt),
		timeToSQL(f.MTime),
		f.Size,
		f.Hash,
		f.Mode,
	)
	if err != nil {
		return err
	}

	return nil
}

func (f *File) Update() error {
	CurrentLogger().Debugf("Updating file %s...", f.RelativePath)
	query := `
		UPDATE file
		SET
		    file_oid = ?,
		    slug = ?,
			relative_path = ?,
			wikilink = ?,
			front_matter = ?,
			attributes = ?,
			title = ?,
			short_title = ?,
			body = ?,
			body_line = ?,
			updated_at = ?,
			last_checked_at = ?,
			mtime = ?,
			size = ?,
			hashsum = ?,
			mode = ?
		WHERE oid = ?;
	`
	frontMatter, err := f.FrontMatterString()
	if err != nil {
		return err
	}
	attributesJSON, err := AttributesJSON(f.Attributes)
	if err != nil {
		return err
	}
	_, err = CurrentDB().Client().Exec(query,
		f.ParentFileOID,
		f.Slug,
		f.RelativePath,
		f.Wikilink,
		frontMatter,
		attributesJSON,
		f.Title,
		f.ShortTitle,
		f.Body,
		f.BodyLine,
		timeToSQL(f.UpdatedAt),
		timeToSQL(f.LastCheckedAt),
		timeToSQL(f.MTime),
		f.Size,
		f.Hash,
		f.Mode,
		f.OID,
	)
	return err
}

func (f *File) Delete() error {
	CurrentLogger().Debugf("Deleting file %s...", f.RelativePath)
	query := `DELETE FROM file WHERE oid = ?;`
	_, err := CurrentDB().Client().Exec(query, f.OID)
	return err
}

func (c *Collection) LoadFileByOID(oid string) (*File, error) {
	return QueryFile(CurrentDB().Client(), `WHERE oid = ?`, oid)
}

func (c *Collection) FindFileByRelativePath(relativePath string) (*File, error) {
	return QueryFile(CurrentDB().Client(), `WHERE relative_path = ?`, relativePath)
}

func (c *Collection) FindFilesByRelativePathPrefix(relativePathPrefix string) ([]*File, error) {
	return QueryFiles(CurrentDB().Client(), `WHERE relative_path LIKE ?`, relativePathPrefix+"%")
}

func (c *Collection) FindFileByWikilink(wikilink string) (*File, error) {
	return QueryFile(CurrentDB().Client(), `WHERE wikilink LIKE ?`, "%"+text.TrimExtension(wikilink))
}

func (c *Collection) FindFilesByWikilink(wikilink string) ([]*File, error) {
	return QueryFiles(CurrentDB().Client(), `WHERE wikilink LIKE ?`, "%"+text.TrimExtension(wikilink))
}

func (c *Collection) FindFilesLastCheckedBefore(point time.Time, path string) ([]*File, error) {
	if path == "." {
		path = ""
	}
	return QueryFiles(CurrentDB().Client(), `WHERE last_checked_at < ? AND relative_path LIKE ?`, timeToSQL(point), path+"%")
}

// CountFiles returns the total number of files.
func (c *Collection) CountFiles() (int, error) {
	db := CurrentDB().Client()

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM file`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

/* SQL Helpers */

func QueryFile(db SQLClient, whereClause string, args ...any) (*File, error) {
	var f File
	var rawFrontMatter string
	var createdAt string
	var updatedAt string
	var lastCheckedAt string
	var mTime string
	var attributesRaw string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			oid,
			file_oid,
			slug,
			relative_path,
			wikilink,
			front_matter,
			attributes,
			title,
			short_title,
			body,
			body_line,
			created_at,
			updated_at,
			last_checked_at,
			mtime,
			size,
			hashsum,
			mode
		FROM file
		%s;`, whereClause), args...).
		Scan(
			&f.OID,
			&f.ParentFileOID,
			&f.Slug,
			&f.RelativePath,
			&f.Wikilink,
			&rawFrontMatter,
			&attributesRaw,
			&f.Title,
			&f.ShortTitle,
			&f.Body,
			&f.BodyLine,
			&createdAt,
			&updatedAt,
			&lastCheckedAt,
			&mTime,
			&f.Size,
			&f.Hash,
			&f.Mode,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	var frontMatter yaml.Node
	err := yaml.Unmarshal([]byte(rawFrontMatter), &frontMatter)
	if err != nil {
		return nil, err
	}
	if frontMatter.Kind > 0 { // Happen when no Front Matter is present
		f.FrontMatter = frontMatter.Content[0]
	}

	attributes, err := UnmarshalAttributes(attributesRaw)
	if err != nil {
		return nil, err
	}

	f.Attributes = attributes
	f.CreatedAt = timeFromSQL(createdAt)
	f.UpdatedAt = timeFromSQL(updatedAt)
	f.LastCheckedAt = timeFromSQL(lastCheckedAt)
	f.MTime = timeFromSQL(mTime)

	return &f, nil
}

func QueryFiles(db SQLClient, whereClause string, args ...any) ([]*File, error) {
	var files []*File

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			oid,
			file_oid,
			slug,
			relative_path,
			wikilink,
			front_matter,
			attributes,
			title,
			short_title,
			body,
			body_line,
			created_at,
			updated_at,
			last_checked_at,
			mtime,
			size,
			hashsum,
			mode
		FROM file
		%s;`, whereClause), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var f File
		var rawFrontMatter string
		var createdAt string
		var updatedAt string
		var lastCheckedAt string
		var mTime string
		var attributesRaw string

		err = rows.Scan(
			&f.OID,
			&f.ParentFileOID,
			&f.Slug,
			&f.RelativePath,
			&f.Wikilink,
			&rawFrontMatter,
			&attributesRaw,
			&f.Title,
			&f.ShortTitle,
			&f.Body,
			&f.BodyLine,
			&createdAt,
			&updatedAt,
			&lastCheckedAt,
			&mTime,
			&f.Size,
			&f.Hash,
			&f.Mode,
		)
		if err != nil {
			return nil, err
		}

		var frontMatter yaml.Node
		err := yaml.Unmarshal([]byte(rawFrontMatter), &frontMatter)
		if err != nil {
			return nil, err
		}
		if frontMatter.Kind > 0 { // Happen when no Front Matter is present
			f.FrontMatter = frontMatter.Content[0]
		}

		attributes, err := UnmarshalAttributes(attributesRaw)
		if err != nil {
			return nil, err
		}

		f.Attributes = attributes
		f.CreatedAt = timeFromSQL(createdAt)
		f.UpdatedAt = timeFromSQL(updatedAt)
		f.LastCheckedAt = timeFromSQL(lastCheckedAt)
		f.MTime = timeFromSQL(mTime)

		files = append(files, &f)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return files, err
}

/* Format */

func (f *File) FormatToJSON() string {
	type FileRepresentation struct {
		OID                string                 `json:"oid"`
		Slug               string                 `json:"slug"`
		RelativePath       string                 `json:"relativePath"`
		Wikilink           string                 `json:"wikilink"`
		Attributes         map[string]interface{} `json:"attributes"`
		ShortTitleRaw      string                 `json:"shortTitleRaw"`
		ShortTitleMarkdown string                 `json:"shortTitleMarkdown"`
		ShortTitleHTML     string                 `json:"shortTitleHTML"`
		ShortTitleText     string                 `json:"shortTitleText"`
		Body               string                 `json:"body"`
		CreatedAt          time.Time              `json:"createdAt"`
		UpdatedAt          time.Time              `json:"updatedAt"`
		DeletedAt          *time.Time             `json:"deletedAt"`
	}
	repr := FileRepresentation{
		OID:                f.OID,
		Slug:               f.Slug,
		RelativePath:       f.RelativePath,
		Wikilink:           f.Wikilink,
		ShortTitleRaw:      f.ShortTitle,
		ShortTitleMarkdown: markdown.ToMarkdown(f.ShortTitle),
		ShortTitleHTML:     markdown.ToHTML(f.ShortTitle),
		ShortTitleText:     markdown.ToText(f.ShortTitle),
		Attributes:         f.GetAttributes(),
		Body:               f.Body,
		CreatedAt:          f.CreatedAt,
		UpdatedAt:          f.UpdatedAt,
	}
	if !f.DeletedAt.IsZero() {
		repr.DeletedAt = &f.DeletedAt
	}
	output, _ := json.MarshalIndent(repr, "", " ")
	return string(output)
}

func (f *File) FormatToYAML() string {
	b := new(strings.Builder)
	f.Write(b)
	return b.String()
}

func (f *File) FormatToMarkdown() string {
	var sb strings.Builder
	frontMatter, err := f.FrontMatterString()
	if err != nil {
		sb.WriteString(frontMatter)
	}
	sb.WriteRune('\n')
	sb.WriteRune('\n')
	sb.WriteString(f.Body)
	return sb.String()
}

func (f *File) FormatToHTML() string {
	var sb strings.Builder
	sb.WriteString(markdown.ToHTML(f.Body))
	return sb.String()
}

func (f *File) FormatToText() string {
	var sb strings.Builder
	sb.WriteString(markdown.ToText(f.Body))
	return sb.String()
}
