package core

import "time"

type MediaKind int

const (
	KindUnknown  MediaKind = 0
	KindAudio    MediaKind = 1
	KindPicture  MediaKind = 2
	KindDocument MediaKind = 3
)

type Media struct {
	ID int64

	// Relative path
	Filepath string

	// Type of media
	Kind MediaKind

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
