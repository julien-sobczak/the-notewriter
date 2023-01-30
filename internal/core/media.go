package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
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

// extractMediasFromMarkdown search for medias from a markdown document (can be a file, a note, a flashcard, etc.).
func extractMediasFromMarkdown(fileRelativePath string, content string) ([]*Media, error) {
	var medias []*Media

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
		media := NewMedia(relpath)
		medias = append(medias, media)
		filepaths[src] = true
	}
	return medias, nil
}

func (m *Media) Save() error {
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

	return nil
}

func (m *Media) SaveWithTx(tx *sql.Tx) error {
	now := clock.Now()
	m.UpdatedAt = now
	m.LastCheckedAt = now

	if m.ID != 0 {
		return m.UpdateWithTx(tx)
	} else {
		m.CreatedAt = now
		return m.InsertWithTx(tx)
	}
}

func (m *Media) InsertWithTx(tx *sql.Tx) error {
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
		)
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
