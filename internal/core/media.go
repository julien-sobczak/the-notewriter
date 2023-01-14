package core

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
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
var PictureExtensions = []string{".jpeg", ".png", ".gif"}
var VideoExtensions = []string{".mp4", ".ogg", ".webm"}

type Media struct {
	ID int64

	// Relative path
	Filepath string

	// Type of media
	Kind MediaKind

	// Media exists on disk
	Dangling bool

	// How many notes references this file
	Links *int

	// File extension in lowercase
	Extension string

	// Content last modification date
	MTime time.Time

	// MD5 Checksum
	Hash string

	// 	Size of the file
	Size int64

	// Timestamps to track changes
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
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
		Filepath:  path,
		Kind:      DetectMediaKind(path),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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
		relpath, err := CurrentCollection().GetRelativePath(fileRelativePath, src)
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
	// TODO
	return nil
}

func (m *Media) SaveWithTx(tx *sql.Tx) error {
	// TODO
	return nil
}
