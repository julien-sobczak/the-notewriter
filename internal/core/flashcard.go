package core

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"gopkg.in/yaml.v3"
)

const DefaultEaseFactor = 2.5  // Same as Anki
const MinEaseFactor = 1.3      // Same as Anki
const DefaultFirstInterval = 1 // day

type CardType int

const (
	CardNew        CardType = 0
	CardLearning   CardType = 1
	CardReview     CardType = 2
	CardRelearning CardType = 3
)

type QueueType int

const (
	QueueSuspend  QueueType = -1 // leeches as manual suspension is not supported
	QueueNew      QueueType = 0  // new (never shown)
	QueueLearn    QueueType = 1  // learning/relearning
	QueueReview   QueueType = 2  // review (as for type)
	QueueDayLearn QueueType = 3  // in learning, next review in at least a day after the previous review
)

type Flashcard struct {
	OID string `yaml:"oid" json:"oid"`

	// File
	FileOID string `yaml:"file_oid" json:"file_oid"`
	File    *File  `yaml:"-" json:"-"` // Lazy-loaded

	// Note representing the flashcard
	NoteOID string `yaml:"note_oid" json:"note_oid"`
	Note    *Note  `yaml:"-" json:"-"` // Lazy-loaded

	// The filepath of the file containing the note (denormalized field)
	RelativePath string `yaml:"relative_path" json:"relative_path"`

	// The slug of the note (denornalized field)
	Slug string `yaml:"slug" json:"slug"`

	// Short title of the note (denormalized field)
	ShortTitle markdown.Document `yaml:"short_title" json:"short_title"`

	// List of tags
	Tags TagSet `yaml:"tags,omitempty" json:"tags,omitempty"`

	// Fields in Markdown (best for editing)
	Front markdown.Document `yaml:"front" json:"front"`
	Back  markdown.Document `yaml:"back" json:"back"`

	// Timestamps to track changes
	CreatedAt     time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt     time.Time `yaml:"updated_at" json:"updated_at"`
	DeletedAt     time.Time `yaml:"deleted_at,omitempty" json:"deleted_at,omitempty"`
	LastCheckedAt time.Time `yaml:"-" json:"-"`

	// SRS
	DueAt     time.Time      `yaml:"due_at,omitempty" json:"due_at,omitempty"`
	StudiedAt time.Time      `yaml:"studied_at,omitempty" json:"studied_at,omitempty"`
	Settings  map[string]any `yaml:"settings,omitempty" json:"settings,omitempty"`

	new   bool
	stale bool
}

type Study struct {
	OID       string    `yaml:"oid" json:"oid"`               // Not persisted in database but can be useful to deduplicate, etc.
	StartedAt time.Time `yaml:"started_at" json:"started_at"` // Timestamp when the first card was revealed
	EndedAt   time.Time `yaml:"ended_at" json:"ended_at"`     // Timestamp when the last card was completed
	Reviews   []*Review `yaml:"reviews" json:"reviews"`

	new   bool // FIXME useful?
	stale bool // FIXME useful?
}

/* Format */

func (s *Study) ToYAML() string {
	return ToBeautifulYAML(s)
}

func (s *Study) ToJSON() string {
	return ToBeautifulJSON(s)
}

func (s *Study) ToMarkdown() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d reviews:", len(s.Reviews)))
	for _, review := range s.Reviews {
		sb.WriteString(fmt.Sprintf("- Flashcard %s: %s", review.FlashcardOID, review.Feedback))
		sb.WriteRune('\n')
	}
	return sb.String()
}

type Feedback string

const (
	// Anki-inspired feedbacks
	FeedbackEasy  Feedback = "easy"
	FeedbackGood  Feedback = "good"
	FeedbackAgain Feedback = "again"
	FeedbackHard  Feedback = "hard"
	// Special feedbacks
	FeedbackTooEasy Feedback = "too-easy" // Used to bury a card to max interval
	FeedbackTooHard Feedback = "too-hard" // Used to relearn a card from scratch
)

type Review struct {
	FlashcardOID string         `yaml:"flashcard_oid" json:"flashcard_oid"`
	Feedback     Feedback       `yaml:"feedback" json:"feedback"`
	DurationInMs int            `yaml:"duration_ms" json:"duration_ms"`
	CompletedAt  time.Time      `yaml:"completed_at" json:"completed_at"`
	DueAt        time.Time      `yaml:"due_at" json:"due_at"`
	Settings     map[string]any `yaml:"settings" json:"settings"` // Include algorithm-specific attributes (like the e-factor in SM-2)
}

func NewOrExistingFlashcard(file *File, note *Note, parsedFlashcard *ParsedFlashcard) (*Flashcard, error) {

	// Try to find an existing note (instead of recreating it from scratch after every change)
	existingFlashcard, err := CurrentRepository().FindMatchingFlashcard(note, parsedFlashcard)
	if err != nil {
		return nil, err
	}

	if existingFlashcard != nil {
		existingFlashcard.update(file, note, parsedFlashcard)
		return existingFlashcard, nil
	}

	return NewFlashcard(file, note, parsedFlashcard)
}

// NewFlashcard initializes a new flashcard.
func NewFlashcard(file *File, note *Note, parsedFlashcard *ParsedFlashcard) (*Flashcard, error) {
	f := &Flashcard{
		OID: NewOID(),

		// File-specific attributes
		FileOID:      file.OID,
		File:         file,
		RelativePath: note.RelativePath,

		// Note-specific attributes
		NoteOID:    note.OID,
		Note:       note,
		Slug:       note.Slug,
		ShortTitle: note.ShortTitle,
		Tags:       note.GetTags(),

		// Flashcard-specific attributes
		Front: parsedFlashcard.Front,
		Back:  parsedFlashcard.Back,

		// SRS-specific attributes
		// Wait for first study to initialize SRS fields

		// Timestamps
		CreatedAt: clock.Now(),
		UpdatedAt: clock.Now(),

		new:   true,
		stale: true,
	}

	return f, nil
}

/* Object */

func (f *Flashcard) Kind() string {
	return "flashcard"
}

func (f *Flashcard) UniqueOID() string {
	return f.OID
}

func (f *Flashcard) ModificationTime() time.Time {
	return f.UpdatedAt
}

func (f *Flashcard) Refresh() (bool, error) {
	// TODO: No need to refresh?
	return false, nil
}

func (f *Flashcard) Stale() bool {
	return f.stale
}

func (f *Flashcard) State() State {
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

func (f *Flashcard) ForceState(state State) {
	switch state {
	case Added:
		f.new = true
	case Deleted:
		f.DeletedAt = clock.Now()
	}
	f.stale = true
}

func (f *Flashcard) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(f)
	if err != nil {
		return err
	}
	return nil
}

func (f *Flashcard) Write(w io.Writer) error {
	data, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (f *Flashcard) Blobs() []*BlobRef {
	// Use Media.Blobs() instead
	return nil
}

func (f *Flashcard) Relations() []*Relation {
	return nil
}

func (f Flashcard) String() string {
	return fmt.Sprintf("flashcard %q [%s]", f.ShortTitle, f.OID)
}

/* Format */

func (f *Flashcard) ToYAML() string {
	return ToBeautifulYAML(f)
}

func (f *Flashcard) ToJSON() string {
	return ToBeautifulJSON(f)
}

func (f *Flashcard) ToMarkdown() string {
	var sb strings.Builder
	sb.WriteString(string(f.Front))
	sb.WriteString("\n\n---\n\n")
	sb.WriteString(string(f.Back))
	return sb.String()
}

/* Update */

func (f *Flashcard) update(file *File, note *Note, parsedFlashcard *ParsedFlashcard) {
	if f.ShortTitle != note.ShortTitle {
		f.ShortTitle = note.ShortTitle
		f.stale = true
	}

	if f.FileOID != file.OID {
		f.FileOID = file.OID
		f.File = file
		f.stale = true
	}

	if f.NoteOID != note.OID {
		f.NoteOID = note.OID
		f.Note = note
		f.stale = true
	}

	if f.Slug != note.Slug {
		f.Slug = note.Slug
		f.stale = true
	}

	if !reflect.DeepEqual(f.Tags, note.GetTags()) {
		f.Tags = note.GetTags()
		f.stale = true
	}

	if f.Front != parsedFlashcard.Front {
		f.Front = parsedFlashcard.Front
		f.stale = true
	}

	if f.Back != parsedFlashcard.Back {
		f.Back = parsedFlashcard.Back
		f.stale = true
	}
}

/* State Management */

func (f *Flashcard) New() bool {
	return f.new
}

func (f *Flashcard) Updated() bool {
	return f.stale
}

func (f *Flashcard) Check() error {
	CurrentLogger().Debugf("Checking flashcard %s...", f.ShortTitle)
	f.LastCheckedAt = clock.Now()
	query := `
		UPDATE flashcard
		SET last_checked_at = ?
		WHERE oid = ?;`
	_, err := CurrentDB().Client().Exec(query,
		timeToSQL(f.LastCheckedAt),
		f.OID,
	)

	return err
}

func (f *Flashcard) Save() error {
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

func (f *Flashcard) Insert() error {
	CurrentLogger().Debugf("Inserting flashcard %s...", f.ShortTitle)
	query := `
		INSERT INTO flashcard(
			oid,
			file_oid,
			note_oid,
			relative_path,
			short_title,
			slug,
			tags,
			front,
			back,
			created_at,
			updated_at,
			last_checked_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
		`
	_, err := CurrentDB().Client().Exec(query,
		f.OID,
		f.FileOID,
		f.NoteOID,
		f.RelativePath,
		f.ShortTitle,
		f.Slug,
		strings.Join(f.Tags, ","),
		f.Front,
		f.Back,
		timeToSQL(f.CreatedAt),
		timeToSQL(f.UpdatedAt),
		timeToSQL(f.LastCheckedAt))
	if err != nil {
		return err
	}

	return nil
}

func (f *Flashcard) Update() error {
	CurrentLogger().Debugf("Updating flashcard %s...", f.ShortTitle)
	query := `
		UPDATE flashcard
		SET
			file_oid = ?,
			note_oid = ?,
			relative_path = ?,
			short_title = ?,
			slug = ?,
			tags = ?,
			front = ?,
			back = ?,
			updated_at = ?,
			last_checked_at = ?
		WHERE oid = ?;
		`
	_, err := CurrentDB().Client().Exec(query,
		f.FileOID,
		f.NoteOID,
		f.RelativePath,
		f.ShortTitle,
		f.Slug,
		strings.Join(f.Tags, ","),
		f.Front,
		f.Back,
		timeToSQL(f.UpdatedAt),
		timeToSQL(f.LastCheckedAt),
		f.OID)

	return err
}

func (f *Flashcard) Delete() error {
	f.ForceState(Deleted)
	CurrentLogger().Debugf("Deleting flashcard %s...", f.ShortTitle)
	query := `DELETE FROM flashcard WHERE oid = ?;`
	_, err := CurrentDB().Client().Exec(query, f.OID)
	return err
}

func SettingsJSON(settings map[string]any) (string, error) {
	var buf bytes.Buffer
	bufEncoder := json.NewEncoder(&buf)
	err := bufEncoder.Encode(settings)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// CountFlashcards returns the total number of flashcards.
func (r *Repository) CountFlashcards() (int, error) {
	db := CurrentDB().Client()

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM flashcard`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (r *Repository) FindMatchingFlashcard(note *Note, parsedFlashcard *ParsedFlashcard) (*Flashcard, error) {
	// Search by slug
	flashcard, err := r.LoadFlashcardBySlug(parsedFlashcard.Slug)
	if err != nil {
		return nil, err
	}
	if flashcard != nil {
		return flashcard, nil
	}

	// Search by note OID
	flashcard, err = r.LoadFlashcardByNoteOID(note.OID)
	if err != nil {
		log.Fatal(err)
	}
	if flashcard != nil {
		return flashcard, nil
	}

	return nil, nil
}

func (r *Repository) LoadFlashcardByOID(oid string) (*Flashcard, error) {
	return QueryFlashcard(CurrentDB().Client(), `WHERE oid = ?`, oid)
}

func (r *Repository) LoadFlashcardBySlug(slug string) (*Flashcard, error) {
	return QueryFlashcard(CurrentDB().Client(), `WHERE slug = ?`, slug)
}

func (r *Repository) LoadFlashcardByNoteOID(noteID string) (*Flashcard, error) {
	return QueryFlashcard(CurrentDB().Client(), `WHERE note_oid = ?`, noteID)
}

func (r *Repository) FindFlashcardByShortTitle(shortTitle string) (*Flashcard, error) {
	return QueryFlashcard(CurrentDB().Client(), `WHERE short_title = ?`, shortTitle)
}

func (r *Repository) FindFlashcardByHash(hash string) (*Flashcard, error) {
	return QueryFlashcard(CurrentDB().Client(), `WHERE hash = ?`, hash)
}

func (r *Repository) FindFlashcardsLastCheckedBefore(point time.Time, path string) ([]*Flashcard, error) {
	if path == "." {
		path = ""
	}
	return QueryFlashcards(CurrentDB().Client(), `WHERE last_checked_at < ? AND relative_path LIKE ?`, timeToSQL(point), path+"%")
}

/* SQL Helpers */

func QueryFlashcard(db SQLClient, whereClause string, args ...any) (*Flashcard, error) {
	var f Flashcard
	var tagsRaw string
	var settingsRaw sql.NullString
	var dueAt sql.NullString
	var studiedAt sql.NullString
	var createdAt string
	var updatedAt string
	var lastCheckedAt string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			oid,
			file_oid,
			note_oid,
			relative_path,
			short_title,
			slug,
			tags,
			front,
			back,
			due_at,
			studied_at,
			settings,
			created_at,
			updated_at,
			last_checked_at
		FROM flashcard
		%s;`, whereClause), args...).
		Scan(
			&f.OID,
			&f.FileOID,
			&f.NoteOID,
			&f.RelativePath,
			&f.ShortTitle,
			&f.Slug,
			&tagsRaw,
			&f.Front,
			&f.Back,
			&dueAt,
			&studiedAt,
			&settingsRaw,
			&createdAt,
			&updatedAt,
			&lastCheckedAt,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	var settings map[string]any
	if settingsRaw.Valid {
		err := yaml.Unmarshal([]byte(settingsRaw.String), &settings)
		if err != nil {
			return nil, err
		}
	}

	f.Tags = strings.Split(tagsRaw, ",")
	f.Settings = settings
	f.DueAt = timeFromNullableSQL(dueAt)
	f.StudiedAt = timeFromNullableSQL(studiedAt)
	f.CreatedAt = timeFromSQL(createdAt)
	f.UpdatedAt = timeFromSQL(updatedAt)
	f.LastCheckedAt = timeFromSQL(lastCheckedAt)

	return &f, nil
}

func QueryFlashcards(db SQLClient, whereClause string, args ...any) ([]*Flashcard, error) {
	var flashcards []*Flashcard

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			oid,
			file_oid,
			note_oid,
			relative_path,
			short_title,
			slug,
			tags,
			front,
			back,
			due_at,
			studied_at,
			settings,
			created_at,
			updated_at,
			last_checked_at
		FROM flashcard
		%s;`, whereClause), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var f Flashcard
		var tagsRaw string
		var settingsRaw sql.NullString
		var dueAt sql.NullString
		var studiedAt sql.NullString
		var createdAt string
		var updatedAt string
		var lastCheckedAt string

		err = rows.Scan(
			&f.OID,
			&f.FileOID,
			&f.NoteOID,
			&f.RelativePath,
			&f.ShortTitle,
			&f.Slug,
			&tagsRaw,
			&f.Front,
			&f.Back,
			&dueAt,
			&studiedAt,
			&settingsRaw,
			&createdAt,
			&updatedAt,
			&lastCheckedAt,
		)
		if err != nil {
			return nil, err
		}

		var settings map[string]any
		if settingsRaw.Valid {
			err := yaml.Unmarshal([]byte(settingsRaw.String), &settings)
			if err != nil {
				return nil, err
			}
		}

		f.Tags = strings.Split(tagsRaw, ",")
		f.Settings = settings
		f.DueAt = timeFromNullableSQL(dueAt)
		f.StudiedAt = timeFromNullableSQL(studiedAt)
		f.CreatedAt = timeFromSQL(createdAt)
		f.UpdatedAt = timeFromSQL(updatedAt)
		f.LastCheckedAt = timeFromSQL(lastCheckedAt)
		flashcards = append(flashcards, &f)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return flashcards, err
}

/*
 * Study
 */

// NewStudy creates a new study.
func NewStudy(flashcardOID string) *Study {
	return &Study{
		OID:   NewOID(),
		stale: true,
	}
}

/* Object */

func (s *Study) Kind() string {
	return "study"
}

func (s *Study) UniqueOID() string {
	return s.OID
}

func (s *Study) ModificationTime() time.Time {
	return s.EndedAt
}

func (s *Study) Refresh() (bool, error) {
	// Study are immutable
	return false, nil
}

func (s *Study) Stale() bool {
	return s.stale
}

func (s *Study) State() State {
	// Mark study as new to try to update the corresponding flashcard
	// if the study is more recent that the last review.
	return Added
}

func (s *Study) ForceState(state State) {
	// Do nothing
}

func (s *Study) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(s)
	if err != nil {
		return err
	}
	return nil
}

func (s *Study) Write(w io.Writer) error {
	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (s *Study) Relations() []*Relation {
	return nil
}

func (s *Study) Blobs() []*BlobRef {
	return nil
}

func (s Study) String() string {
	return fmt.Sprintf("study %q started on %v", s.OID, s.StartedAt)
}

func (s *Study) Save() error {
	CurrentLogger().Debugf("Study %s has %d reviews", s.OID, len(s.Reviews))
	for _, review := range s.Reviews {
		CurrentLogger().Debugf("Saving review for flashcard %s...", review.FlashcardOID)
		// Read the flashcard to determine if the study is more recent that the last study
		flashcard, err := CurrentRepository().LoadFlashcardByOID(review.FlashcardOID)
		if err != nil {
			return err
		}
		if flashcard == nil {
			CurrentLogger().Debugf("Flashcard %s not found", review.FlashcardOID)
			continue
		}

		if flashcard.StudiedAt.After(review.CompletedAt) {
			// The last known study is more recent. Ignore this study.
			CurrentLogger().Debugf("Flashcard %s already studied since", review.FlashcardOID)
			continue
		}

		// Record the study
		CurrentLogger().Debugf("Updating flashcard %s following new study...", flashcard.ShortTitle)

		settingsRaw, err := SettingsJSON(review.Settings)
		if err != nil {
			return err
		}
		query := `
			UPDATE flashcard
			SET
				due_at = ?,
				studied_at = ?,
				settings = ?
			WHERE oid = ?;
			`
		_, err = CurrentDB().Client().Exec(query,
			timeToSQL(review.DueAt),
			timeToSQL(review.CompletedAt),
			settingsRaw,
			review.FlashcardOID)
		if err != nil {
			return err
		}
		CurrentLogger().Debugf("Updated flashcard %s following new study", flashcard.ShortTitle)
	}

	return nil
}

/* Anki SM-2 settings */
/*
-- 0=new, 1=learning, 2=review, 3=relearning
"type" INTEGER NOT NULL DEFAULT 0,

-- Queue types:
--   -1=suspend     => leeches as manual suspension is not supported
--    0=new         => new (never shown)
--    1=(re)lrn     => learning/relearning
--    2=rev         => review (as for type)
--    3=day (re)lrn => in learning, next review in at least a day after the previous review
queue INTEGER NOT NULL DEFAULT 0,

-- Due is used differently for different card types:
--    new: note oid or random int
--    due: integer day, relative to the repository's creation time
--    learning: integer timestamp in second
due INTEGER NOT NULL DEFAULT 0,

-- The interval. Negative = seconds, positive = days
ivl INTEGER NOT NULL DEFAULT 0,

-- The ease factor in permille (ex: 2500 = the interval will be multiplied by 2.5 the next time you press "Good").
ease_factor INTEGER NOT NULL DEFAULT 0,

-- The number of reviews.
repetitions INTEGER NOT NULL DEFAULT 0,

-- The number of times the card went from a "was answered correctly" to "was answered incorrectly" state.
lapses INTEGER NOT NULL DEFAULT 0,

-- Of the form a*1000+b, with:
--    a the number of reps left today
--    b the number of reps left till graduation
--    for example: '2004' means 2 reps left today and 4 reps till graduation
left INTEGER NOT NULL DEFAULT 0,
*/
