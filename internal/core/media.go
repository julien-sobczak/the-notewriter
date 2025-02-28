package core

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/medias"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
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
var PictureExtensions = []string{".jpg", ".jpeg", ".png", ".gif", ".svg", ".avif"}

// List of supported picture formats
var VideoExtensions = []string{".mp4", ".ogg", ".webm"}

// Maximum width and/or height for preview blobs.
const PreviewMaxWidthOrHeight = 600

// Maximum width and/or height for large blobs.
const LargeMaxWidthOrHeight = 1980

type Media struct {
	OID oid.OID `yaml:"oid" json:"oid"`

	// Pack file where this object belongs
	PackFileOID oid.OID `yaml:"packfile_oid" json:"packfile_oid"`

	// Relative path
	RelativePath string `yaml:"relative_path" json:"relative_path"`

	// Type of media
	MediaKind MediaKind `yaml:"kind" json:"kind"`

	// Media exists on disk
	Dangling bool `yaml:"dangling" json:"dangling"`

	// File extension in lowercase
	Extension string `yaml:"extension" json:"extension"`

	// Content last modification date
	MTime time.Time `yaml:"mtime" json:"mtime"`

	// MD5 Checksum
	Hash string `yaml:"hash" json:"hash"`

	// 	Size of the file
	Size int64 `yaml:"size" json:"size"`

	// Eager-loaded list of blobs
	BlobRefs []*BlobRef `yaml:"blobs" json:"blobs"`

	// Timestamps to track changes
	CreatedAt time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at" json:"updated_at"`
	IndexedAt time.Time `yaml:"indexed_at,omitempty" json:"indexed_at,omitempty"`
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
func NewMedia(packFile *PackFile, parsedMedia *ParsedMedia) (*Media, error) {
	hash := ""
	if !parsedMedia.Dangling {
		hash = parsedMedia.FileHash()
	}

	return &Media{
		OID:          oid.New(),
		PackFileOID:  packFile.OID,
		RelativePath: parsedMedia.RelativePath,
		MediaKind:    parsedMedia.MediaKind,
		Extension:    parsedMedia.Extension,
		Dangling:     parsedMedia.Dangling,
		Size:         parsedMedia.FileSize(),
		MTime:        parsedMedia.FileMTime(),
		Hash:         hash,
		CreatedAt:    packFile.CTime,
		UpdatedAt:    packFile.CTime,
		IndexedAt:    packFile.CTime,
	}, nil
}

func NewOrExistingMedia(packFile *PackFile, parsedMedia *ParsedMedia) (*Media, error) {
	// Try to find an existing object (instead of recreating it from scratch after every change)
	existingMedia, err := CurrentRepository().FindMatchingMedia(parsedMedia)
	if err != nil {
		return nil, err
	}
	if existingMedia != nil {
		existingMedia.update(packFile, parsedMedia)
		return existingMedia, nil
	}
	return NewMedia(packFile, parsedMedia)
}

func (m *Media) update(packFile *PackFile, parsedMedia *ParsedMedia) {
	stale := false

	// Special case when file didn't exist or no longer exist
	if m.Dangling != parsedMedia.Dangling {
		m.Dangling = parsedMedia.Dangling
		stale = true

		if parsedMedia.Dangling {
			m.Size = 0
			m.Hash = ""
			m.MTime = time.Time{}
		} else {
			m.Size = parsedMedia.Size
			m.MTime = parsedMedia.MTime
			m.Hash = parsedMedia.FileHash()
		}
		return
	}

	// Check if file on disk has changed
	if m.Size != parsedMedia.Size {
		m.Size = parsedMedia.Size
		stale = true
	}
	if m.MTime != parsedMedia.MTime {
		m.MTime = parsedMedia.MTime
		stale = true
	}
	hash := parsedMedia.FileHash()
	if m.Hash != hash {
		m.Hash = hash
		stale = true
	}

	m.PackFileOID = packFile.OID
	m.IndexedAt = packFile.CTime

	if stale {
		m.UpdatedAt = m.IndexedAt
	}
}

func (m *Media) GenerateBlobs() {
	if CurrentConfig().DryRun {
		return
	}

	if m.Dangling {
		// No blob for missing media
		return
	}

	src := CurrentRepository().GetAbsolutePath(m.RelativePath)

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
			toAVIF(converter, src, dest, medias.ResizeTo(PreviewMaxWidthOrHeight))
			m.BlobRefs = append(m.BlobRefs, MustWriteBlob(dest, []string{"preview", "lossy"}))
		}

		if dimensions.LargerThan(LargeMaxWidthOrHeight) {
			dest := filepath.Join(tmpDir, filepath.Base(src)+".large.avif")
			toAVIF(converter, src, dest, medias.ResizeTo(LargeMaxWidthOrHeight))
			m.BlobRefs = append(m.BlobRefs, MustWriteBlob(dest, []string{"large", "lossy"}))
		}

		dest := filepath.Join(tmpDir, filepath.Base(src)+".original.avif")
		toAVIF(converter, src, dest, medias.OriginalSize())
		m.BlobRefs = append(m.BlobRefs, MustWriteBlob(dest, []string{"original", "lossy"}))

	case KindAudio:
		dest := filepath.Join(tmpDir, filepath.Base(src)+".original.mp3")
		toMP3(converter, src, dest)
		m.BlobRefs = append(m.BlobRefs, MustWriteBlob(dest, []string{"original", "lossy"}))

	case KindVideo:
		dest := filepath.Join(tmpDir, filepath.Base(src)+".original.webm")
		toWebM(converter, src, dest)
		m.BlobRefs = append(m.BlobRefs, MustWriteBlob(dest, []string{"original", "lossy"}))

		// and generate a picture from the first frame
		dest = filepath.Join(tmpDir, filepath.Base(src)+".preview.avif")
		toAVIF(converter, src, dest, medias.ResizeTo(PreviewMaxWidthOrHeight))
		m.BlobRefs = append(m.BlobRefs, MustWriteBlob(dest, []string{"preview", "lossy"}))
	}
}

func toAVIF(converter medias.Converter, src, dest string, dimensions medias.Dimensions) {
	_, err := os.Stat(dest)
	if os.IsNotExist(err) {
		err := converter.ToAVIF(src, dest, dimensions)
		if err != nil {
			log.Fatalf("Unable to generate preview blob from file %q: %v", src, err)
		}
		return
	}
	if err != nil {
		log.Fatalf("Unable to retrieve stat for file %q: %v", src, err)
	}
}
func toMP3(converter medias.Converter, src, dest string) {
	_, err := os.Stat(dest)
	if os.IsNotExist(err) {
		err := converter.ToMP3(src, dest)
		if err != nil {
			log.Fatalf("Unable to generate preview blob from file %q: %v", src, err)
		}
		return
	}
	if err != nil {
		log.Fatalf("Unable to retrieve stat for file %q: %v", src, err)
	}
}
func toWebM(converter medias.Converter, src, dest string) {
	_, err := os.Stat(dest)
	if os.IsNotExist(err) {
		err := converter.ToWebM(src, dest)
		if err != nil {
			log.Fatalf("Unable to generate preview blob from file %q: %v", src, err)
		}
		return
	}
	if err != nil {
		log.Fatalf("Unable to retrieve stat for file %q: %v", src, err)
	}
}

// MustWriteBlob writes a new blob object or fails.
func MustWriteBlob(path string, tags []string) *BlobRef {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Unable to read blob %q: %v", path, err)
	}
	ext := filepath.Ext(path)
	oid := oid.NewFromBytes(data)
	blob := &BlobRef{
		OID:      oid,
		MimeType: medias.MimeType(ext),
		Tags:     tags,
	}
	if err := CurrentDB().WriteBlobOnDisk(blob.OID, data); err != nil {
		log.Fatalf("Unable to write blob from file %q: %v", path, err)
	}
	return blob
}

/* Object */

func (m *Media) Kind() string {
	return "media"
}

func (m *Media) UniqueOID() oid.OID {
	return m.OID
}

func (m *Media) ModificationTime() time.Time {
	return m.UpdatedAt
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

func (m *Media) Relations() []*Relation {
	// Medias are referenced by notes but don't have relation toward other objects by themselves.
	return nil
}

func (m Media) String() string {
	return fmt.Sprintf("media %s [%s]", m.RelativePath, m.OID)
}

/* Format */

func (m *Media) ToYAML() string {
	return ToBeautifulYAML(m)
}

func (m *Media) ToJSON() string {
	return ToBeautifulJSON(m)
}

func (m *Media) ToMarkdown() string {
	var sb strings.Builder
	sb.WriteString("![](")
	sb.WriteString(string(m.RelativePath))
	sb.WriteString(")")
	return sb.String()
}

/* Database Management */

func (m *Media) Save() error {
	CurrentLogger().Debugf("Saving media %s...", m.RelativePath)

	if err := m.InsertBlobs(); err != nil {
		return err
	}

	query := `
		INSERT INTO media(
			oid,
			packfile_oid,
			relative_path,
			kind,
			dangling,
			extension,
			mtime,
			hashsum,
			size,
			created_at,
			updated_at,
			indexed_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(oid) DO UPDATE SET
			packfile_oid = ?,
			relative_path = ?,
			kind = ?,
			dangling = ?,
			extension = ?,
			mtime = ?,
			hashsum = ?,
			size = ?,
			updated_at = ?,
			indexed_at = ?
		;
	`
	_, err := CurrentDB().Client().Exec(query,
		// Insert
		m.OID,
		m.PackFileOID,
		m.RelativePath,
		m.MediaKind,
		m.Dangling,
		m.Extension,
		timeToSQL(m.MTime),
		m.Hash,
		m.Size,
		timeToSQL(m.CreatedAt),
		timeToSQL(m.UpdatedAt),
		timeToSQL(m.IndexedAt),
		// Update
		m.PackFileOID,
		m.RelativePath,
		m.MediaKind,
		m.Dangling,
		m.Extension,
		timeToSQL(m.MTime),
		m.Hash,
		m.Size,
		timeToSQL(m.UpdatedAt),
		timeToSQL(m.IndexedAt),
	)
	if err != nil {
		return err
	}

	return nil
}

func (m *Media) InsertBlobs() error {
	if err := m.DeleteBlobs(); err != nil {
		return err
	}
	for _, b := range m.BlobRefs {
		// Blobs are immutable and their OID is determined using a hashsum.
		// Two medias can contains the same content and share the same blobs.

		blob, err := CurrentRepository().FindBlobFromOID(b.OID)
		if err != nil {
			return err
		}
		if blob != nil {
			CurrentLogger().Debugf("Ignoring existing blob %s...", b.OID)
			// Already exists
			continue
		}

		CurrentLogger().Debugf("Inserting blob %s...", b.OID)
		attributes, err := b.Attributes.ToJSON()
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

func (m *Media) Delete() error {
	CurrentLogger().Debugf("Deleting media %s...", m.RelativePath)
	if err := m.DeleteBlobs(); err != nil {
		return err
	}
	query := `DELETE FROM media WHERE oid = ? AND packfile_oid = ?;`
	_, err := CurrentDB().Client().Exec(query, m.OID, m.PackFileOID)
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

/* SQL Queries */

// CountMedias returns the total number of medias.
func (r *Repository) CountMedias() (int, error) {
	var count int
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM media`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (r *Repository) LoadMediaByOID(oid oid.OID) (*Media, error) {
	return QueryMedia(CurrentDB().Client(), `WHERE oid = ?`, oid)
}

func (r *Repository) FindMatchingMedia(parsedMedia *ParsedMedia) (*Media, error) {
	// Find by hash (ex: file was renamed)
	media, err := r.FindMediaByHash(parsedMedia.FileHash())
	if err != nil {
		return nil, err
	}
	if media != nil {
		return media, nil
	}

	// Find by relative path
	relativePath := r.GetFileRelativePath(parsedMedia.AbsolutePath)
	return r.FindMediaByRelativePath(relativePath)
}

func (r *Repository) FindMediaByRelativePath(relativePath string) (*Media, error) {
	return QueryMedia(CurrentDB().Client(), `WHERE relative_path = ?`, relativePath)
}

func (r *Repository) FindMediaByHash(hash string) (*Media, error) {
	return QueryMedia(CurrentDB().Client(), `WHERE hashsum = ?`, hash)
}

func (r *Repository) FindMediasLastCheckedBefore(point time.Time) ([]*Media, error) {
	return QueryMedias(CurrentDB().Client(), `WHERE indexed_at < ?`, timeToSQL(point))
}

func (r *Repository) FindBlobsFromMedia(mediaOID oid.OID) ([]*BlobRef, error) {
	return QueryBlobs(CurrentDB().Client(), "WHERE media_oid = ?", mediaOID)
}

func (r *Repository) FindBlobFromOID(oid oid.OID) (*BlobRef, error) {
	return QueryBlob(CurrentDB().Client(), "WHERE oid = ?", oid)
}

/* SQL Helpers */

func QueryMedia(db SQLClient, whereClause string, args ...any) (*Media, error) {
	var m Media
	var createdAt string
	var updatedAt string
	var lastIndexedAt string
	var mTime string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			oid,
			packfile_oid,
			relative_path,
			kind,
			dangling,
			extension,
			mtime,
			hashsum,
			size,
			created_at,
			updated_at,
			indexed_at
		FROM media
		%s;`, whereClause), args...).
		Scan(
			&m.OID,
			&m.PackFileOID,
			&m.RelativePath,
			&m.MediaKind,
			&m.Dangling,
			&m.Extension,
			&mTime,
			&m.Hash,
			&m.Size,
			&createdAt,
			&updatedAt,
			&lastIndexedAt,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	m.CreatedAt = timeFromSQL(createdAt)
	m.UpdatedAt = timeFromSQL(updatedAt)
	m.IndexedAt = timeFromSQL(lastIndexedAt)
	m.MTime = timeFromSQL(mTime)

	// Load blobs
	blobs, err := CurrentRepository().FindBlobsFromMedia(m.OID)
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
			packfile_oid,
			relative_path,
			kind,
			dangling,
			extension,
			mtime,
			hashsum,
			size,
			created_at,
			updated_at,
			indexed_at
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
		var lastIndexedAt string
		var mTime string

		err = rows.Scan(
			&m.OID,
			&m.PackFileOID,
			&m.RelativePath,
			&m.MediaKind,
			&m.Dangling,
			&m.Extension,
			&mTime,
			&m.Hash,
			&m.Size,
			&createdAt,
			&updatedAt,
			&lastIndexedAt,
		)
		if err != nil {
			return nil, err
		}

		m.CreatedAt = timeFromSQL(createdAt)
		m.UpdatedAt = timeFromSQL(updatedAt)
		m.IndexedAt = timeFromSQL(lastIndexedAt)
		m.MTime = timeFromSQL(mTime)

		// Load blobs
		blobs, err := CurrentRepository().FindBlobsFromMedia(m.OID)
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

/* FileObject interface */

func (m *Media) FileRelativePath() string {
	return m.RelativePath
}
func (m *Media) FileMTime() time.Time {
	return m.MTime
}
func (m *Media) FileSize() int64 {
	return m.Size
}
func (m *Media) FileHash() string {
	return m.Hash
}
func (m *Media) Blobs() []*BlobRef {
	return m.BlobRefs
}
