package core

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notetaker/internal/helpers"
	"github.com/julien-sobczak/the-notetaker/internal/medias"
	"github.com/julien-sobczak/the-notetaker/pkg/clock"
	"github.com/julien-sobczak/the-notetaker/pkg/text"
	"gopkg.in/yaml.v3"
)

type MediaKind string

const (
	KindUnknown  MediaKind = "unknown"
	KindAudio    MediaKind = "audio"
	KindPicture  MediaKind = "picture"
	KindVideo    MediaKind = "video"
	KindDocument MediaKind = "document"
)

// List of supported audio formats
var AudioExtensions = []string{".mp3", ".wav"}

// List of supported picture formats
var PictureExtensions = []string{".jpeg", ".png", ".gif", ".svg", ".avif"}

// List of supported picture formats
var VideoExtensions = []string{".mp4", ".ogg", ".webm"}

// Maximum width and/or height for preview blobs.
const PreviewMaxWidthOrHeight = 600

// Maximum width and/or height for large blobs.
const LargeMaxWidthOrHeight = 1980

type Media struct {
	OID string `yaml:"oid"`

	// Relative path
	RelativePath string `yaml:"relative_path"`

	// Type of media
	MediaKind MediaKind `yaml:"kind"`

	// Media exists on disk
	Dangling bool `yaml:"dangling"`

	// File extension in lowercase
	Extension string `yaml:"extension"`

	// Content last modification date
	MTime time.Time `yaml:"mtime"`

	// MD5 Checksum
	Hash string `yaml:"hash"`

	// 	Size of the file
	Size int64 `yaml:"size"`

	// Permission of the file
	Mode fs.FileMode `yaml:"mode"`

	// Eager-loaded list of blobs
	BlobRefs []*BlobRef `yaml:"blobs"`

	// Timestamps to track changes
	CreatedAt     time.Time `yaml:"created_at"`
	UpdatedAt     time.Time `yaml:"updated_at"`
	DeletedAt     time.Time `yaml:"deleted_at,omitempty"`
	LastCheckedAt time.Time `yaml:"-"`

	new   bool
	stale bool
}

// DetectMediaKind returns the media kind based on a file path.
func DetectMediaKind(filename string) MediaKind {
	ext := filepath.Ext(filename)
	for _, audioExt := range AudioExtensions {
		if strings.EqualFold(ext, audioExt) {
			return KindAudio
		}
	}
	for _, pictureExt := range PictureExtensions {
		if strings.EqualFold(ext, pictureExt) {
			return KindPicture
		}
	}
	for _, videoExt := range VideoExtensions {
		if strings.EqualFold(ext, videoExt) {
			return KindVideo
		}
	}
	return KindUnknown
}

// NewMedia initializes a new media.
func NewMedia(relpath string) *Media {
	m := &Media{
		OID:          NewOID(),
		RelativePath: relpath,
		MediaKind:    DetectMediaKind(relpath),
		Extension:    filepath.Ext(relpath),
		CreatedAt:    clock.Now(),
		UpdatedAt:    clock.Now(),
		new:          true,
		stale:        true,
	}

	abspath := CurrentCollection().GetAbsolutePath(relpath)
	stat, err := os.Stat(abspath)
	if errors.Is(err, os.ErrNotExist) {
		m.Dangling = true
		return m
	}

	m.Dangling = false
	m.Size = stat.Size()
	m.Hash, _ = helpers.HashFromFile(abspath)
	m.MTime = stat.ModTime()
	m.Mode = stat.Mode()

	m.UpdateBlobs()

	return m
}

func (m *Media) update() {
	abspath := CurrentCollection().GetAbsolutePath(m.RelativePath)
	stat, err := os.Stat(abspath)
	dangling := errors.Is(err, os.ErrNotExist)

	// Special case when file didn't exist or no longer exist
	if m.Dangling != dangling {
		m.Dangling = dangling
		m.stale = true

		if dangling {
			m.Size = 0
			m.Hash = ""
			m.MTime = time.Time{}
			m.Mode = 0
		} else {
			m.Size = stat.Size()
			m.Hash, _ = helpers.HashFromFile(abspath)
			m.MTime = stat.ModTime()
			m.Mode = stat.Mode()
		}
		return
	}

	// Check if file on disk has changed
	size := stat.Size()
	if m.Size != size {
		m.Size = size
		m.stale = true
	}
	hash, _ := helpers.HashFromFile(abspath)
	if m.Hash != hash {
		m.Hash = hash
		m.stale = true
	}
	mTime := stat.ModTime()
	if m.MTime != mTime {
		m.MTime = mTime
		m.stale = true
	}
	mode := stat.Mode()
	if m.Mode != mode {
		m.Mode = mode
		m.stale = true
	}

	if m.stale {
		m.UpdateBlobs()
	}
}

func (m *Media) UpdateBlobs() {
	if CurrentConfig().DryRun {
		return
	}

	src := CurrentCollection().GetAbsolutePath(m.RelativePath)

	tmpDir := CurrentConfig().TempDir()
	converter := CurrentConfig().Converter()

	// Old blobs will be gc later if not referenced.
	m.BlobRefs = nil

	switch m.MediaKind {

	case KindUnknown:
		// Same as document
		fallthrough
	case KindDocument:
		// Nothing to convert
		// Simply copy the content
		blob := MustWriteBlob(src, []string{"original", "lossless"})
		m.BlobRefs = append(m.BlobRefs, blob)

	case KindPicture:
		// Convert to AVIF (widely supported in desktop and mobiles as of 2023)

		dimensions, _ := medias.ReadImageDimensions(src)

		if dimensions.LargerThan(PreviewMaxWidthOrHeight) {
			dest := filepath.Join(tmpDir, filepath.Base(src)+".preview.avif")
			err := converter.ToAVIF(src, dest, medias.ResizeTo(PreviewMaxWidthOrHeight))
			if err != nil {
				log.Fatalf("Unable to generate preview blob from file %q: %v", m.RelativePath, err)
			}
			m.BlobRefs = append(m.BlobRefs, MustWriteBlob(dest, []string{"preview", "lossy"}))
		}

		if dimensions.LargerThan(LargeMaxWidthOrHeight) {
			dest := filepath.Join(tmpDir, filepath.Base(src)+".large.avif")
			err := converter.ToAVIF(src, dest, medias.ResizeTo(LargeMaxWidthOrHeight))
			if err != nil {
				log.Fatalf("Unable to generate preview blob from file %q: %v", m.RelativePath, err)
			}
			m.BlobRefs = append(m.BlobRefs, MustWriteBlob(dest, []string{"large", "lossy"}))
		}

		dest := filepath.Join(tmpDir, filepath.Base(src)+".original.avif")
		err := converter.ToAVIF(src, dest, medias.OriginalSize())
		if err != nil {
			log.Fatalf("Unable to generate preview blob from file %q: %v", m.RelativePath, err)
		}
		m.BlobRefs = append(m.BlobRefs, MustWriteBlob(dest, []string{"original", "lossy"}))

	case KindAudio:
		dest := filepath.Join(tmpDir, filepath.Base(src)+".original.mp3")
		err := converter.ToMP3(src, dest)
		if err != nil {
			log.Fatalf("Unable to generate preview blob from file %q: %v", m.RelativePath, err)
		}
		m.BlobRefs = append(m.BlobRefs, MustWriteBlob(dest, []string{"original", "lossy"}))

	case KindVideo:
		dest := filepath.Join(tmpDir, filepath.Base(src)+".original.webm")
		err := converter.ToWebM(src, dest)
		if err != nil {
			log.Fatalf("Unable to generate preview blob from file %q: %v", m.RelativePath, err)
		}
		m.BlobRefs = append(m.BlobRefs, MustWriteBlob(dest, []string{"original", "lossy"}))

		// and generate a picture from the first frame
		dest = filepath.Join(tmpDir, filepath.Base(src)+".preview.avif")
		err = converter.ToAVIF(src, dest, medias.ResizeTo(PreviewMaxWidthOrHeight))
		if err != nil {
			log.Fatalf("Unable to generate preview blob from file %q: %v", m.RelativePath, err)
		}
		m.BlobRefs = append(m.BlobRefs, MustWriteBlob(dest, []string{"preview", "lossy"}))
	}
}

// MustWriteBlob writes a new blob or fail.
func MustWriteBlob(path string, tags []string) *BlobRef {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Unable to read blob %q: %v", path, err)
	}
	ext := filepath.Ext(path)
	oid := helpers.Hash(data)
	blob := BlobRef{
		OID:      oid,
		MimeType: medias.MimeType(ext),
		Tags:     tags,
	}
	if err := CurrentDB().WriteBlob(blob.OID, data); err != nil {
		log.Fatalf("Unable to write blob from file %q: %v", path, err)
	}
	return &blob
}

/* Object */

func (m *Media) Kind() string {
	return "media"
}

func (m *Media) UniqueOID() string {
	return m.OID
}

func (m *Media) ModificationTime() time.Time {
	return m.UpdatedAt
}

func (m *Media) State() State {
	if !m.DeletedAt.IsZero() {
		return Deleted
	}
	if m.new {
		return Added
	}
	if m.stale {
		return Modified
	}
	return None
}

func (m *Media) ForceState(state State) {
	switch state {
	case Added:
		m.new = true
	case Deleted:
		m.DeletedAt = clock.Now()
	}
	m.stale = true
}
func (m *Media) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(m)
	if err != nil {
		return err
	}
	return nil
}

func (m *Media) Write(w io.Writer) error {
	data, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (m *Media) SubObjects() []StatefulObject {
	return nil
}

func (m *Media) Blobs() []*BlobRef {
	return m.BlobRefs
}

func (m Media) String() string {
	return fmt.Sprintf("media %s [%s]", m.RelativePath, m.OID)
}

/* State Management */

func (m *Media) New() bool {
	return m.new
}

func (m *Media) Updated() bool {
	return m.stale
}

/* Parsing */

// extractMediasFromMarkdown search for medias from a markdown document (can be a file, a note, a flashcard, etc.).
func extractMediasFromMarkdown(fileRelativePath string, fileBody string) []*Media {
	var medias []*Media

	parsedMedias := ParseMedias(fileRelativePath, fileBody)
	for _, parsedMedia := range parsedMedias {
		media := NewOrExistingMedia(parsedMedia.RelativePath)
		medias = append(medias, media)
	}
	return medias
}

type ParsedMedia struct {
	// The path as specified in the file. (Ex: "../medias/pic.png")
	RawPath string
	// The path relative to the root collection directory. (Ex: "references/medias/pic.png")
	RelativePath string
	// The absolute path. (Ex: "$HOME/notes/references/medias/pic.png")
	AbsolutePath string
	// Line number where the link present.
	Line int
}

// ParseMedias extracts raw paths from a file or note body content.
func ParseMedias(fileRelativePath, fileBody string) []*ParsedMedia {
	var medias []*ParsedMedia

	// Avoid returning duplicates if a media is included twice
	filepaths := make(map[string]bool)

	regexMedia := regexp.MustCompile(`!\[(.*?)\]\((\S*?)(?:\s+"(.*?)")?\)`)
	matches := regexMedia.FindAllStringSubmatch(fileBody, -1)
	for _, match := range matches {
		txt := match[0]
		line := text.LineNumber(fileBody, txt)

		rawPath := match[2]
		if _, ok := filepaths[rawPath]; ok {
			continue
		}
		relativePath, err := CurrentCollection().GetNoteRelativePath(fileRelativePath, rawPath)
		if err != nil {
			log.Fatal(err)
		}
		absolutePath := CurrentCollection().GetAbsolutePath(relativePath)

		medias = append(medias, &ParsedMedia{
			RawPath:      rawPath,
			RelativePath: relativePath,
			AbsolutePath: absolutePath,
			Line:         line,
		})
		filepaths[rawPath] = true
	}

	return medias
}

func NewOrExistingMedia(relpath string) *Media {
	media, err := CurrentCollection().FindMediaByRelativePath(relpath)
	if err != nil {
		log.Fatal(err)
	}
	if media != nil {
		media.update()
		return media
	}

	media = NewMedia(relpath)
	return media
}

func (m *Media) Check() error {
	CurrentLogger().Debugf("Checking media %s...", m.RelativePath)
	m.LastCheckedAt = clock.Now()
	query := `
		UPDATE media
		SET last_checked_at = ?
		WHERE oid = ?;`
	_, err := CurrentDB().Client().Exec(query,
		timeToSQL(m.LastCheckedAt),
		m.OID,
	)

	return err
}

func (m *Media) Save() error {
	var err error
	m.UpdatedAt = clock.Now()
	m.LastCheckedAt = clock.Now()
	switch m.State() {
	case Added:
		err = m.Insert()
	case Modified:
		err = m.Update()
	case Deleted:
		err = m.Delete()
	default:
		err = m.Check()
	}
	if err != nil {
		return err
	}
	m.new = false
	m.stale = false
	return nil
}

func (m *Media) Insert() error {
	CurrentLogger().Debugf("Inserting media %s...", m.RelativePath)
	query := `
		INSERT INTO media(
			oid,
			relative_path,
			kind,
			dangling,
			extension,
			mtime,
			hashsum,
			size,
			mode,
			created_at,
			updated_at,
			last_checked_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	_, err := CurrentDB().Client().Exec(query,
		m.OID,
		m.RelativePath,
		m.MediaKind,
		m.Dangling,
		m.Extension,
		timeToSQL(m.MTime),
		m.Hash,
		m.Size,
		m.Mode,
		timeToSQL(m.CreatedAt),
		timeToSQL(m.UpdatedAt),
		timeToSQL(m.LastCheckedAt),
	)
	if err != nil {
		return err
	}

	// Insert blobs
	m.InsertBlobs()

	return nil
}

func (m *Media) InsertBlobs() error {
	if err := m.DeleteBlobs(); err != nil {
		return err
	}
	for _, b := range m.BlobRefs {
		// Blobs are immutable and their OID is determined using a hashsum.
		// Two medias can contains the same content and share the same blobs.

		blob, err := CurrentCollection().FindBlobFromOID(b.OID)
		if err != nil {
			return err
		}
		if blob != nil {
			CurrentLogger().Debugf("Ignoring existing blob %s...", b.OID)
			// Already exists
			continue
		}

		CurrentLogger().Debugf("Inserting blob %s...", b.OID)
		attributes, err := AttributesString(b.Attributes)
		if err != nil {
			return err
		}
		query := `
			INSERT INTO blob(
				oid,
				media_oid,
				mime,
				attributes,
				tags
			)
			VALUES (?, ?, ?, ?, ?);
		`
		_, err = CurrentDB().Client().Exec(query,
			b.OID,
			m.OID,
			b.MimeType,
			attributes,
			strings.Join(b.Tags, ","),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Media) Update() error {
	CurrentLogger().Debugf("Updating media %s...", m.RelativePath)
	query := `
		UPDATE media
		SET
			relative_path = ?,
			kind = ?,
			dangling = ?,
			extension = ?,
			mtime = ?,
			hashsum = ?,
			size = ?,
			mode = ?,
			created_at = ?,
			updated_at = ?,
			last_checked_at = ?
		WHERE oid = ?;
	`
	_, err := CurrentDB().Client().Exec(query,
		m.RelativePath,
		m.MediaKind,
		m.Dangling,
		m.Extension,
		timeToSQL(m.MTime),
		m.Hash,
		m.Size,
		m.Mode,
		timeToSQL(m.CreatedAt),
		timeToSQL(m.UpdatedAt),
		timeToSQL(m.LastCheckedAt),
		m.OID,
	)

	// Insert blobs
	m.InsertBlobs()

	return err
}

func (m *Media) Delete() error {
	if err := m.DeleteBlobs(); err != nil {
		return err
	}
	CurrentLogger().Debugf("Deleting media %s...", m.RelativePath)
	query := `DELETE FROM media WHERE oid = ?;`
	_, err := CurrentDB().Client().Exec(query, m.OID)
	return err
}

func (m *Media) DeleteBlobs() error {
	CurrentLogger().Debugf("Deleting blobs for media %s...", m.OID)
	query := `DELETE FROM blob WHERE media_oid = ?;`
	res, err := CurrentDB().Client().Exec(query, m.OID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	CurrentLogger().Debugf("Deleted %d rows", rows)
	return nil
}

// CountMedias returns the total number of medias.
func (c *Collection) CountMedias() (int, error) {
	var count int
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM media`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (c *Collection) LoadMediaByOID(oid string) (*Media, error) {
	return QueryMedia(CurrentDB().Client(), `WHERE oid = ?`, oid)
}

func (c *Collection) FindMediaByRelativePath(relativePath string) (*Media, error) {
	return QueryMedia(CurrentDB().Client(), `WHERE relative_path = ?`, relativePath)
}

func (c *Collection) FindMediaByHash(hash string) (*Media, error) {
	return QueryMedia(CurrentDB().Client(), `WHERE hashsum = ?`, hash)
}

func (c *Collection) FindMediasLastCheckedBefore(point time.Time) ([]*Media, error) {
	return QueryMedias(CurrentDB().Client(), `WHERE last_checked_at < ?`, timeToSQL(point))
}

func (c *Collection) FindBlobsFromMedia(mediaOID string) ([]*BlobRef, error) {
	return QueryBlobs(CurrentDB().Client(), "WHERE media_oid = ?", mediaOID)
}

func (c *Collection) FindBlobFromOID(oid string) (*BlobRef, error) {
	return QueryBlob(CurrentDB().Client(), "WHERE oid = ?", oid)
}

/* SQL Helpers */

func QueryMedia(db SQLClient, whereClause string, args ...any) (*Media, error) {
	var m Media
	var createdAt string
	var updatedAt string
	var lastCheckedAt string
	var mTime string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			oid,
			relative_path,
			kind,
			dangling,
			extension,
			mtime,
			hashsum,
			size,
			mode,
			created_at,
			updated_at,
			last_checked_at
		FROM media
		%s;`, whereClause), args...).
		Scan(
			&m.OID,
			&m.RelativePath,
			&m.MediaKind,
			&m.Dangling,
			&m.Extension,
			&mTime,
			&m.Hash,
			&m.Size,
			&m.Mode,
			&createdAt,
			&updatedAt,
			&lastCheckedAt,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	m.CreatedAt = timeFromSQL(createdAt)
	m.UpdatedAt = timeFromSQL(updatedAt)
	m.LastCheckedAt = timeFromSQL(lastCheckedAt)
	m.MTime = timeFromSQL(mTime)

	// Load blobs
	blobs, err := CurrentCollection().FindBlobsFromMedia(m.OID)
	if err != nil {
		return nil, err
	}
	m.BlobRefs = blobs

	return &m, nil
}

func QueryMedias(db SQLClient, whereClause string, args ...any) ([]*Media, error) {
	var medias []*Media

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			oid,
			relative_path,
			kind,
			dangling,
			extension,
			mtime,
			hashsum,
			size,
			mode,
			created_at,
			updated_at,
			last_checked_at
		FROM media
		%s;`, whereClause), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var m Media
		var createdAt string
		var updatedAt string
		var lastCheckedAt string
		var mTime string

		err = rows.Scan(
			&m.OID,
			&m.RelativePath,
			&m.MediaKind,
			&m.Dangling,
			&m.Extension,
			&mTime,
			&m.Hash,
			&m.Size,
			&m.Mode,
			&createdAt,
			&updatedAt,
			&lastCheckedAt,
		)
		if err != nil {
			return nil, err
		}

		m.CreatedAt = timeFromSQL(createdAt)
		m.UpdatedAt = timeFromSQL(updatedAt)
		m.LastCheckedAt = timeFromSQL(lastCheckedAt)
		m.MTime = timeFromSQL(mTime)

		// Load blobs
		blobs, err := CurrentCollection().FindBlobsFromMedia(m.OID)
		if err != nil {
			return nil, err
		}
		m.BlobRefs = blobs

		medias = append(medias, &m)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return medias, err
}

func QueryBlob(db SQLClient, whereClause string, args ...any) (*BlobRef, error) {
	var b BlobRef
	var attributesRaw string
	var tagsRaw string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			oid,
			mime,
			attributes,
			tags
		FROM blob
		%s;`, whereClause), args...).
		Scan(
			&b.OID,
			&b.MimeType,
			&attributesRaw,
			&tagsRaw,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	attributes := make(map[string]interface{})
	err := yaml.Unmarshal([]byte(attributesRaw), &attributes)
	if err != nil {
		return nil, err
	}
	b.Attributes = attributes
	b.Tags = strings.Split(tagsRaw, ",")

	return &b, nil
}

func QueryBlobs(db SQLClient, whereClause string, args ...any) ([]*BlobRef, error) {
	var blobs []*BlobRef

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			oid,
			mime,
			attributes,
			tags
		FROM blob
		%s;`, whereClause), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b BlobRef
		var attributesRaw string
		var tagsRaw string

		err = rows.Scan(
			&b.OID,
			&b.MimeType,
			&attributesRaw,
			&tagsRaw,
		)
		if err != nil {
			return nil, err
		}

		attributes := make(map[string]interface{})
		err := yaml.Unmarshal([]byte(attributesRaw), &attributes)
		if err != nil {
			return nil, err
		}
		b.Attributes = attributes
		b.Tags = strings.Split(tagsRaw, ",")
		blobs = append(blobs, &b)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return blobs, err
}

/* Helpers */

// AttributesString formats the current attributes to the YAML front matter format.
func AttributesString(attributes map[string]interface{}) (string, error) {
	var buf bytes.Buffer
	bufEncoder := yaml.NewEncoder(&buf)
	bufEncoder.SetIndent(Indent)
	err := bufEncoder.Encode(attributes)
	if err != nil {
		return "", err
	}
	return CompactYAML(buf.String()), nil
}
