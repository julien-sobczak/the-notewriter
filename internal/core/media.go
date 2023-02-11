package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/clock"
)

type MediaKind int

const (
	KindUnknown  MediaKind = 0
	KindAudio    MediaKind = 1
	KindPicture  MediaKind = 2
	KindVideo    MediaKind = 3
	KindDocument MediaKind = 4
)

var AudioExtensions = []string{".mp3", ".wav"}
var PictureExtensions = []string{".jpeg", ".png", ".gif", ".svg", ".avif"}
var VideoExtensions = []string{".mp4", ".ogg", ".webm"}

type Media struct {
	ID int64

	// Relative path
	RelativePath string

	// Type of media
	Kind MediaKind

	// Media exists on disk
	Dangling bool

	// How many notes references this file
	Links int

	// File extension in lowercase
	Extension string

	// Content last modification date
	MTime time.Time

	// MD5 Checksum
	Hash string

	// 	Size of the file
	Size int64

	// Permission of the file
	Mode fs.FileMode

	// Timestamps to track changes
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     time.Time
	LastCheckedAt time.Time

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
func NewMedia(path string) *Media {
	m := &Media{
		RelativePath: path,
		Kind:         DetectMediaKind(path),
		Extension:    filepath.Ext(path),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		stale:        true,
	}

	abspath := CurrentCollection().GetAbsolutePath(path)
	stat, err := os.Stat(abspath)
	if errors.Is(err, os.ErrNotExist) {
		m.Dangling = true
		return m
	}

	m.Dangling = false
	m.Size = stat.Size()
	m.Hash, _ = hashFromFile(abspath)
	m.MTime = stat.ModTime()
	m.Mode = stat.Mode()

	return m
}

func (m *Media) Update() {
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
			m.Hash, _ = hashFromFile(abspath)
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
	hash, _ := hashFromFile(abspath)
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
}

/* State Management */

func (m *Media) New() bool {
	return m.ID == 0
}

func (m *Media) Updated() bool {
	return m.stale
}

/* Parsing */

// extractMediasFromMarkdown search for medias from a markdown document (can be a file, a note, a flashcard, etc.).
func extractMediasFromMarkdown(fileRelativePath string, content string) ([]*Media, error) {
	var medias []*Media

	// Avoid returning duplicates if a media is included twice
	filepaths := make(map[string]bool)

	regexMedia := regexp.MustCompile(`!\[(.*?)\]\((\S*?)(?:\s+"(.*?)")?\)`)
	matches := regexMedia.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		src := match[2]
		if _, ok := filepaths[src]; ok {
			continue
		}
		relpath, err := CurrentCollection().GetNoteRelativePath(fileRelativePath, src)
		if err != nil {
			return nil, err
		}
		media := NewOrExistingMedia(relpath)
		medias = append(medias, media)
		filepaths[src] = true
	}
	return medias, nil
}

func NewOrExistingMedia(relpath string) *Media {
	media, err := FindMediaByRelativePath(relpath)
	if err != nil {
		log.Fatal(err)
	}
	if media != nil {
		media.Update()
		return media
	}

	media = NewMedia(relpath)
	return media
}

func (m *Media) Check() error {
	db := CurrentDB().Client()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = m.CheckWithTx(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil

}

func (m *Media) CheckWithTx(tx *sql.Tx) error {
	if CurrentConfig().Debug() {
		log.Printf("Checking media %s...", m.RelativePath)
	}
	m.LastCheckedAt = clock.Now()
	query := `
		UPDATE media
		SET last_checked_at = ?
		WHERE id = ?;`
	_, err := tx.Exec(query,
		timeToSQL(m.LastCheckedAt),
		m.ID,
	)

	return err
}

func (m *Media) Save() error {
	if !m.stale {
		return m.Check()
	}

	db := CurrentDB().Client()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = m.SaveWithTx(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	m.stale = false

	return nil
}

func (m *Media) SaveWithTx(tx *sql.Tx) error {
	if !m.stale {
		return m.CheckWithTx(tx)
	}

	now := clock.Now()
	m.UpdatedAt = now
	m.LastCheckedAt = now

	if m.ID != 0 {
		if err := m.UpdateWithTx(tx); err != nil {
			return err
		}
	} else {
		m.CreatedAt = now
		if err := m.InsertWithTx(tx); err != nil {
			return err
		}
	}

	m.stale = false

	return nil
}

func (m *Media) InsertWithTx(tx *sql.Tx) error {
	if CurrentConfig().Debug() {
		log.Printf("Inserting media %s...", m.RelativePath)
	}
	query := `
		INSERT INTO media(
			id,
			relative_path,
			kind,
			dangling,
			extension,
			mtime,
			hashsum,
			links,
			size,
			mode,
			created_at,
			updated_at,
			deleted_at,
			last_checked_at
		)
		VALUES (NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	res, err := tx.Exec(query,
		m.RelativePath,
		m.Kind,
		m.Dangling,
		m.Extension,
		timeToSQL(m.MTime),
		m.Hash,
		m.Links,
		m.Size,
		m.Mode,
		timeToSQL(m.CreatedAt),
		timeToSQL(m.UpdatedAt),
		timeToSQL(m.DeletedAt),
		timeToSQL(m.LastCheckedAt),
	)
	if err != nil {
		return err
	}

	var id int64
	if id, err = res.LastInsertId(); err != nil {
		return err
	}
	m.ID = id

	return nil
}

func (m *Media) UpdateWithTx(tx *sql.Tx) error {
	if CurrentConfig().Debug() {
		log.Printf("Updating media %s...", m.RelativePath)
	}
	query := `
		UPDATE media
		SET
			relative_path = ?,
			kind = ?,
			dangling = ?,
			extension = ?,
			mtime = ?,
			hashsum = ?,
			links = ?,
			size = ?,
			mode = ?,
			created_at = ?,
			updated_at = ?,
			deleted_at = ?,
			last_checked_at = ?
		WHERE id = ?;
	`
	_, err := tx.Exec(query,
		m.RelativePath,
		m.Kind,
		m.Dangling,
		m.Extension,
		timeToSQL(m.MTime),
		m.Hash,
		m.Links,
		m.Size,
		m.Mode,
		timeToSQL(m.UpdatedAt),
		timeToSQL(m.DeletedAt),
		timeToSQL(m.LastCheckedAt),
		m.ID,
	)

	return err
}

func (m *Media) Delete() error {
	db := CurrentDB().Client()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = m.DeleteWithTx(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (m *Media) DeleteWithTx(tx *sql.Tx) error {
	if CurrentConfig().Debug() {
		log.Printf("Deleting media %s...", m.RelativePath)
	}
	query := `DELETE FROM media WHERE id = ?;`
	_, err := tx.Exec(query, m.ID)
	return err
}

// CountMedias returns the total number of medias.
func CountMedias() (int, error) {
	db := CurrentDB().Client()

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM media WHERE deleted_at = ''`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func LoadMediaByID(id int64) (*Media, error) {
	return QueryMedia(`WHERE id = ?`, id)
}

func FindMediaByRelativePath(relativePath string) (*Media, error) {
	return QueryMedia(`WHERE relative_path = ?`, relativePath)
}

func FindMediaByHash(hash string) (*Media, error) {
	return QueryMedia(`WHERE hashsum = ?`, hash)
}

func FindMediasLastCheckedBefore(point time.Time) ([]*Media, error) {
	return QueryMedias(`WHERE last_checked_at < ?`, timeToSQL(point))
}

/* SQL Helpers */

func QueryMedia(whereClause string, args ...any) (*Media, error) {
	db := CurrentDB().Client()

	var m Media
	var createdAt string
	var updatedAt string
	var deletedAt string
	var lastCheckedAt string
	var mTime string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			id,
			relative_path,
			kind,
			dangling,
			extension,
			mtime,
			hashsum,
			links,
			size,
			mode,
			created_at,
			updated_at,
			deleted_at,
			last_checked_at
		FROM media
		%s;`, whereClause), args...).
		Scan(
			&m.ID,
			&m.RelativePath,
			&m.Kind,
			&m.Dangling,
			&m.Extension,
			&mTime,
			&m.Hash,
			&m.Links,
			&m.Size,
			&m.Mode,
			&createdAt,
			&updatedAt,
			&deletedAt,
			&lastCheckedAt,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	m.CreatedAt = timeFromSQL(createdAt)
	m.UpdatedAt = timeFromSQL(updatedAt)
	m.DeletedAt = timeFromSQL(deletedAt)
	m.LastCheckedAt = timeFromSQL(lastCheckedAt)
	m.MTime = timeFromSQL(mTime)

	return &m, nil
}

func QueryMedias(whereClause string, args ...any) ([]*Media, error) {
	db := CurrentDB().Client()

	var medias []*Media

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			id,
			relative_path,
			kind,
			dangling,
			extension,
			mtime,
			hashsum,
			links,
			size,
			mode,
			created_at,
			updated_at,
			deleted_at,
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
		var deletedAt string
		var lastCheckedAt string
		var mTime string

		err = rows.Scan(
			&m.ID,
			&m.RelativePath,
			&m.Kind,
			&m.Dangling,
			&m.Extension,
			&mTime,
			&m.Hash,
			&m.Links,
			&m.Size,
			&m.Mode,
			&createdAt,
			&updatedAt,
			&deletedAt,
			&lastCheckedAt,
		)
		if err != nil {
			return nil, err
		}

		m.CreatedAt = timeFromSQL(createdAt)
		m.UpdatedAt = timeFromSQL(updatedAt)
		m.DeletedAt = timeFromSQL(deletedAt)
		m.LastCheckedAt = timeFromSQL(lastCheckedAt)
		m.MTime = timeFromSQL(mTime)
		medias = append(medias, &m)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return medias, err
}
