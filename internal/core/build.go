package core

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/julien-sobczak/the-notetaker/pkg/clock"
)

type BuildResult struct {
	files      map[string]State
	notes      map[string]State
	flashcards map[string]State
	links      map[string]State
	reminders  map[string]State
	medias     map[string]State
}

func NewBuildResult() *BuildResult {
	return &BuildResult{
		files:      make(map[string]State),
		notes:      make(map[string]State),
		flashcards: make(map[string]State),
		links:      make(map[string]State),
		reminders:  make(map[string]State),
		medias:     make(map[string]State),
	}
}

func (b *BuildResult) UpdateFile(file *File) {
	if file.New() {
		b.setActionOnFile(file, Added)
		for _, note := range file.GetNotes() {
			b.setActionOnNote(note, Added)
			links, _ := note.GetLinks()
			for _, link := range links {
				b.setActionOnLink(link, Added)
			}
			reminders, _ := note.GetReminders()
			for _, reminder := range reminders {
				b.setActionOnReminder(reminder, Added)
			}
		}
		for _, flashcard := range file.GetFlashcards() {
			b.setActionOnFlashcard(flashcard, Added)
		}
		medias, _ := file.GetMedias()
		for _, media := range medias {
			b.setActionOnMedia(media, Added)
		}
	} else if file.Updated() {
		b.setActionOnFile(file, Modified)
		for _, note := range file.GetNotes() {
			if note.New() {
				b.setActionOnNote(note, Added)
			} else if note.Updated() {
				b.setActionOnNote(note, Modified)
			} else {
				b.setActionOnNote(note, None)
			}
			links, _ := note.GetLinks()
			for _, link := range links {
				if link.New() {
					b.setActionOnLink(link, Added)
				} else if link.Updated() {
					b.setActionOnLink(link, Modified)
				} else {
					b.setActionOnLink(link, None)
				}
			}
			reminders, _ := note.GetReminders()
			for _, reminder := range reminders {
				if reminder.New() {
					b.setActionOnReminder(reminder, Added)
				} else if reminder.Updated() {
					b.setActionOnReminder(reminder, Modified)
				} else {
					b.setActionOnReminder(reminder, None)
				}
			}
		}
		for _, flashcard := range file.GetFlashcards() {
			if flashcard.New() {
				b.setActionOnFlashcard(flashcard, Added)
			} else if flashcard.Updated() {
				b.setActionOnFlashcard(flashcard, Modified)
			} else {
				b.setActionOnFlashcard(flashcard, None)
			}
		}
		medias, _ := file.GetMedias()
		for _, media := range medias {
			if media.New() {
				b.setActionOnMedia(media, Added)
			} else if media.Updated() {
				b.setActionOnMedia(media, Modified)
			} else {
				b.setActionOnMedia(media, None)
			}
		}
	} else {
		b.setActionOnFile(file, None)
		for _, note := range file.GetNotes() {
			b.setActionOnNote(note, None)
			links, _ := note.GetLinks()
			for _, link := range links {
				b.setActionOnLink(link, None)
			}
			reminders, _ := note.GetReminders()
			for _, reminder := range reminders {
				b.setActionOnReminder(reminder, None)
			}
		}
		for _, flashcard := range file.GetFlashcards() {
			b.setActionOnFlashcard(flashcard, None)
		}
		medias, _ := file.GetMedias()
		for _, media := range medias {
			b.setActionOnMedia(media, None)
		}
	}
}

func (b *BuildResult) DeleteFile(file *File) {
	b.setActionOnFile(file, Deleted)
}
func (b *BuildResult) DeleteNote(note *Note) {
	b.setActionOnNote(note, Deleted)
}
func (b *BuildResult) DeleteFlashcard(flashcard *Flashcard) {
	b.setActionOnFlashcard(flashcard, Deleted)
}
func (b *BuildResult) DeleteLink(link *Link) {
	b.setActionOnLink(link, Deleted)
}
func (b *BuildResult) DeleteMedia(media *Media) {
	b.setActionOnMedia(media, Deleted)
}
func (b *BuildResult) DeleteReminder(reminder *Reminder) {
	b.setActionOnReminder(reminder, Deleted)
}

func (b *BuildResult) setActionOnFile(file *File, state State) {
	b.files[file.Wikilink] = state
}
func (b *BuildResult) setActionOnNote(note *Note, state State) {
	b.notes[note.Wikilink] = state
}
func (b *BuildResult) setActionOnFlashcard(flashcard *Flashcard, state State) {
	b.flashcards[flashcard.ShortTitle] = state
}
func (b *BuildResult) setActionOnLink(link *Link, state State) {
	b.links[link.GoName] = state
}
func (b *BuildResult) setActionOnMedia(media *Media, state State) {
	b.medias[media.RelativePath] = state
}
func (b *BuildResult) setActionOnReminder(reminder *Reminder, state State) {
	b.reminders[reminder.DescriptionRaw] = state
}

func (c *Collection) walk(fn func(path string, stat fs.FileInfo) error) {
	config := CurrentConfig()

	CurrentLogger().Infof("Reading %s...\n", config.RootDirectory)

	filepath.WalkDir(config.RootDirectory, func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err) // FIXME not visible in stderr
			return err
		}

		dirname := filepath.Base(path)
		if dirname == ".nt" {
			return fs.SkipDir // NB fs.SkipDir skip the parent dir when path is a file
		}
		if dirname == ".git" {
			return fs.SkipDir
		}

		relpath, err := CurrentCollection().GetFileRelativePath(path)
		if err != nil {
			// ignore the file
			return nil
		}

		if !config.IgnoreFile.Include(relpath) {
			return nil
		}

		// We look for only specific extension
		if !info.IsDir() && !config.ConfigFile.SupportExtension(relpath) {
			// Nothing to do
			return nil
		}

		// Ignore certain file modes like symlinks
		fileInfo, err := os.Lstat(path) // NB: os.Stat follows symlinks
		if err != nil {
			// Ignore the file
			return nil
		}
		if !fileInfo.Mode().IsRegular() {
			// Exclude any file with a mode bit set (device, socket, named pipe, ...)
			// See https://pkg.go.dev/io/fs#FileMode
			return nil
		}

		if err := fn(path, fileInfo); err != nil {
			return err
		}

		return nil
	})
}

func (c *Collection) Build(outputDirectory string) (*BuildResult, error) {
	result := NewBuildResult()
	buildTime := clock.Now()

	c.walk(func(path string, stat fs.FileInfo) error {
		CurrentLogger().Debugf("Processing %s...\n", path) // TODO emit notif for tests? + Parse file
		file, err := NewOrExistingFile(path)
		if err != nil {
			return err
		}
		result.UpdateFile(file)

		err = file.Save()
		if err != nil {
			return err
		}
		return nil
	})

	// Search for deleted files, notes, ...
	links, err := FindLinksLastCheckedBefore(buildTime)
	if err != nil {
		return result, err
	}
	for _, link := range links {
		result.DeleteLink(link)
		if err := link.Delete(); err != nil {
			return result, err
		}
	}
	reminders, err := FindRemindersLastCheckedBefore(buildTime)
	if err != nil {
		return result, err
	}
	for _, reminder := range reminders {
		result.DeleteReminder(reminder)
		if err := reminder.Delete(); err != nil {
			return result, err
		}
	}
	flashcards, err := FindFlashcardsLastCheckedBefore(buildTime)
	if err != nil {
		return result, err
	}
	for _, flashcard := range flashcards {
		result.DeleteFlashcard(flashcard)
		if err := flashcard.Delete(); err != nil {
			return result, err
		}
	}
	medias, err := FindMediasLastCheckedBefore(buildTime)
	if err != nil {
		return result, err
	}
	for _, media := range medias {
		result.DeleteMedia(media)
		if err := media.Delete(); err != nil {
			return result, err
		}
	}
	notes, err := FindNotesLastCheckedBefore(buildTime)
	if err != nil {
		return result, err
	}
	for _, note := range notes {
		result.DeleteNote(note)
		if err := note.Delete(); err != nil {
			return result, err
		}
	}
	files, err := FindFilesLastCheckedBefore(buildTime)
	if err != nil {
		return result, err
	}
	for _, file := range files {
		result.DeleteFile(file)
		if err := file.Delete(); err != nil {
			return result, err
		}
	}

	return result, nil
}

func (c *Collection) Update(buildResult *BuildResult) error {
	// Update all tables and their timestamps
	// + generate replication log + packfiles (only if change, a rebuild must not trigger new files)
	// TODO Mark as stale old records (deleted_at)
	// buildResult.files
	// file.GetNotes()
	// note.GetMedias()
	// media.GetExtension()
	// media.GetHash()
	// media.GetSize()

	// note.GetLinks() // or file.GetLinks()
	// note.AsMarkdown()
	// note.AsHTML()
	// note.AsText()

	return nil
}
