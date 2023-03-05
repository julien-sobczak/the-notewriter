package core

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/clock"
	"github.com/julien-sobczak/the-notetaker/pkg/markdown"
	"github.com/julien-sobczak/the-notetaker/pkg/text"
	"gopkg.in/yaml.v3"
)

type Reminder struct {
	OID string `yaml:"oid"`

	// File
	FileOID string `yaml:"file_oid"`
	File    *File  `yaml:"-"` // Lazy-loaded

	// Note representing the flashcard
	NoteOID string `yaml:"note_oid"`
	Note    *Note  `yaml:"-"` // Lazy-loaded

	// The filepath of the file containing the note (denormalized field)
	RelativePath string `yaml:"relative_path"`

	// Description
	DescriptionRaw      string `yaml:"description_raw"`
	DescriptionMarkdown string `yaml:"description_markdown"`
	DescriptionHTML     string `yaml:"description_html"`
	DescriptionText     string `yaml:"description_text"`

	// Tag value containig the formula to determine the next occurence
	Tag string `yaml:"tag"`

	// Timestamps to track progress
	LastPerformedAt time.Time `yaml:"last_performed_at"`
	NextPerformedAt time.Time `yaml:"next_performed_at"`

	// Timestamps to track changes
	CreatedAt     time.Time `yaml:"created_at"`
	UpdatedAt     time.Time `yaml:"updated_at"`
	DeletedAt     time.Time `yaml:"-"`
	LastCheckedAt time.Time `yaml:"-"`

	new   bool
	stale bool
}

func NewOrExistingReminder(note *Note, descriptionRaw, tag string) (*Reminder, error) {
	descriptionRaw = strings.TrimSpace(descriptionRaw)

	reminders, err := FindRemindersMatching(note.OID, descriptionRaw)
	if err != nil {
		log.Fatal(err)
	}
	if len(reminders) == 1 {
		reminder := reminders[0]
		err = reminder.Update(note, descriptionRaw, tag)
		return reminder, err
	}
	return NewReminder(note, descriptionRaw, tag)
}

// NewReminder instantiates a new reminder.
func NewReminder(note *Note, descriptionRaw, tag string) (*Reminder, error) {
	descriptionRaw = strings.TrimSpace(descriptionRaw)

	r := &Reminder{
		OID:          NewOID(),
		FileOID:      note.FileOID,
		NoteOID:      note.OID,
		RelativePath: note.RelativePath,
		Tag:          tag,
		CreatedAt:    clock.Now(),
		UpdatedAt:    clock.Now(),
		stale:        true,
		new:          true,
	}

	r.updateContent(descriptionRaw)

	err := r.Next()
	if err != nil {
		return nil, err
	}

	return r, nil
}

/* Object */

func (r *Reminder) Kind() string {
	return "reminder"
}

func (r *Reminder) UniqueOID() string {
	return r.OID
}

func (r *Reminder) ModificationTime() time.Time {
	return r.UpdatedAt
}

func (r *Reminder) State() State {
	if !r.DeletedAt.IsZero() {
		return Deleted
	}
	if r.new {
		return Added
	}
	if r.stale {
		return Modified
	}
	return None
}

func (r *Reminder) ForceState(state State) {
	switch state {
	case Added:
		r.new = true
	case Deleted:
		r.DeletedAt = clock.Now()
	}
	r.stale = true
}

func (n *Reminder) Read(r io.Reader) error {
	err := yaml.NewDecoder(r).Decode(n)
	if err != nil {
		return err
	}
	return nil
}

func (r *Reminder) Write(w io.Writer) error {
	data, err := yaml.Marshal(r)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (r *Reminder) SubObjects() []StatefulObject {
	return nil
}

func (r *Reminder) Blobs() []BlobRef {
	// Use Media.Blobs() instead
	return nil
}

func (r Reminder) String() string {
	return fmt.Sprintf("reminder %s [%s]", r.Tag, r.OID)
}

/* Update */

func (r *Reminder) updateContent(descriptionRaw string) {
	r.DescriptionRaw = descriptionRaw
	r.DescriptionMarkdown = markdown.ToMarkdown(r.DescriptionRaw)
	r.DescriptionHTML = markdown.ToHTML(r.DescriptionMarkdown)
	r.DescriptionText = markdown.ToText(r.DescriptionMarkdown)
}

func (r *Reminder) Update(note *Note, descriptionRaw, tag string) error {
	if r.FileOID != note.FileOID {
		r.FileOID = note.FileOID
		r.File = note.File
		r.stale = true
	}
	if r.NoteOID != note.OID {
		r.NoteOID = note.OID
		r.Note = note
		r.stale = true
	}
	if r.DescriptionRaw != descriptionRaw {
		r.updateContent(descriptionRaw)
		r.stale = true
	}
	if r.Tag != tag {
		r.Tag = tag
		r.stale = true
		err := r.Next()
		if err != nil {
			return err
		}
	}
	return nil
}

/* State Management */

func (r *Reminder) New() bool {
	return r.new
}

func (r *Reminder) Updated() bool {
	return r.stale
}

func (r *Reminder) Next() error {
	if clock.Now().Before(r.NextPerformedAt) {
		// already OK
		return nil
	}

	expression := strings.TrimPrefix(r.Tag, "#reminder-")

	lastPerformedAt := r.NextPerformedAt
	nextPerformedAt, err := EvaluateTimeExpression(expression)
	if err != nil {
		return err
	}
	r.LastPerformedAt = lastPerformedAt
	r.NextPerformedAt = nextPerformedAt
	return nil
}

/* Parsing */

// EvaluateTimeExpression determine the next matching reminder date
func EvaluateTimeExpression(expr string) (time.Time, error) {
	originalExpr := expr
	today := clock.Now()

	// Static dates are easier to address first
	var reStaticDate = regexp.MustCompile(`(\d{4})(?:-(\d{2})(?:-(\d{2})))`)
	if reStaticDate.MatchString(expr) {
		var year, month, day int
		match := reStaticDate.FindStringSubmatch(expr)
		year, _ = strconv.Atoi(match[1])
		monthStr := match[2]
		dayStr := match[3]
		if dayStr == "" {
			day = 1
		} else {
			day, _ = strconv.Atoi(dayStr)
		}
		if monthStr == "" {
			if day < today.Day() {
				month = int(today.Month()) + 1
			} else {
				month = int(today.Month())
			}
		} else {
			month, _ = strconv.Atoi(monthStr)
		}
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
	}

	// We have an expression where the year, month, day can be ommitted and where different syntaxes are supported (through variables).
	// The first step is to determine the different parts to know if we have a year, a month, or day.
	yearSpecified := false
	yearExpr := ""
	monthSpecified := false
	monthExpr := ""
	daySpecified := false
	dayExpr := ""

	expr = strings.TrimPrefix(expr, "every-") // syntaxic sugar (not useful for the algorithm)

	// Detect year expression
	match, _ := regexp.MatchString(`^\d{4}-?.*`, expr)
	if match {
		yearSpecified = true
		yearExpr = expr[0:4]
		expr = expr[4:]
	} else if strings.HasPrefix(expr, "${year}") {
		yearSpecified = true
		yearExpr = "year"
		expr = strings.TrimPrefix(expr, "${year}")
	} else if strings.HasPrefix(expr, "${odd-year}") {
		yearSpecified = true
		yearExpr = "odd-year"
		expr = strings.TrimPrefix(expr, "${odd-year}")
	} else if strings.HasPrefix(expr, "${even-year}") {
		yearSpecified = true
		yearExpr = "even-year"
		expr = strings.TrimPrefix(expr, "${even-year}")
	} else {
		yearSpecified = false
	}

	if expr != "" {
		expr = strings.TrimPrefix(expr, "-")

		// Detect month expression
		match, _ = regexp.MatchString(`^\d{2}-?.*`, expr)
		if match {
			monthSpecified = true
			monthExpr = expr[0:2]
			expr = expr[2:]
		} else if strings.HasPrefix(expr, "${month}") {
			monthSpecified = true
			monthExpr = "month"
			expr = strings.TrimPrefix(expr, "${month}")
		} else if strings.HasPrefix(expr, "${odd-month}") {
			monthSpecified = true
			monthExpr = "odd-month"
			expr = strings.TrimPrefix(expr, "${odd-month}")
		} else if strings.HasPrefix(expr, "${even-month}") {
			monthSpecified = true
			monthExpr = "even-month"
			expr = strings.TrimPrefix(expr, "${even-month}")
		} else {
			monthSpecified = false
		}

		if expr != "" {
			expr = strings.TrimPrefix(expr, "-")

			// Detect day expression
			match, _ := regexp.MatchString(`^\d{2}-?.*`, expr)
			if match {
				daySpecified = true
				dayExpr = expr[0:2]
				expr = expr[2:]
			} else if strings.HasPrefix(expr, "${day}") {
				daySpecified = true
				dayExpr = "day"
				expr = strings.TrimPrefix(expr, "${day}")
			} else if strings.HasPrefix(expr, "${monday}") {
				daySpecified = true
				dayExpr = "monday"
				expr = strings.TrimPrefix(expr, "${monday}")
			} else if strings.HasPrefix(expr, "${tuesday}") {
				daySpecified = true
				dayExpr = "tuesday"
				expr = strings.TrimPrefix(expr, "${tuesday}")
			} else if strings.HasPrefix(expr, "${wednesday}") {
				daySpecified = true
				dayExpr = "wednesday"
				expr = strings.TrimPrefix(expr, "${wednesday}")
			} else if strings.HasPrefix(expr, "${thursday}") {
				daySpecified = true
				dayExpr = "thursday"
				expr = strings.TrimPrefix(expr, "${thursday}")
			} else if strings.HasPrefix(expr, "${friday}") {
				daySpecified = true
				dayExpr = "friday"
				expr = strings.TrimPrefix(expr, "${friday}")
			} else if strings.HasPrefix(expr, "${saturday}") {
				daySpecified = true
				dayExpr = "saturday"
				expr = strings.TrimPrefix(expr, "${saturday}")
			} else if strings.HasPrefix(expr, "${sunday}") {
				daySpecified = true
				dayExpr = "sunday"
				expr = strings.TrimPrefix(expr, "${sunday}")
			} else {
				daySpecified = false
			}
		}

	}

	// The reminder must have been completely parsed now
	if expr != "" {
		return time.Time{}, fmt.Errorf("unexpected character after the end of reminder expression %q", originalExpr)
	}

	// We must at least have a year, a month, or a day
	if !yearSpecified && !monthSpecified && !daySpecified {
		return time.Time{}, fmt.Errorf("missing date in reminder expression %q", originalExpr)
	}

	// Generate all possible combinations
	possibleDates := generateDates(yearExpr, monthExpr, dayExpr)

	// Filter to keep only future dates
	var possibleFutureDates []time.Time
	for _, possibleDate := range possibleDates {
		if possibleDate.After(today) {
			possibleFutureDates = append(possibleFutureDates, possibleDate)
		}
	}

	// Sort to find the next future date
	sort.Slice(possibleFutureDates, func(i, j int) bool {
		return possibleFutureDates[i].Before(possibleFutureDates[j])
	})

	return possibleFutureDates[0], nil
}

func generateDates(yearExpr, monthExpr, dayExpr string) []time.Time {
	// Implementation: We generate all potential candidate dates as it's not easy to determine the target value.
	//
	// Ex: `reminder-${year}-07-02`
	// * If today is 2023-07-01, the expected year is 2023
	// * If today is 2023-08-01, the expected year is 2024
	// The code doesn't bother and simply return [2023-07-02, 2024-07-02].
	// The calling code will just have to sort the date and return the first future date.
	//
	// The function is recursive. We replace each variable by all possible values before evaluating the new expressions again
	// until they are no more variables to replace.

	// Base case
	if text.IsNumber(yearExpr) && text.IsNumber(monthExpr) && text.IsNumber(dayExpr) {
		year, _ := strconv.Atoi(yearExpr)
		month, _ := strconv.Atoi(monthExpr)
		day, _ := strconv.Atoi(dayExpr)
		return []time.Time{time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)}
	}

	today := clock.Now()
	var dates []time.Time
	if !text.IsNumber(yearExpr) {
		switch yearExpr {
		case "":
			fallthrough
		case "year":
			// this year or next year
			dates = append(dates, generateDates(fmt.Sprint(today.Year()), monthExpr, dayExpr)...)
			dates = append(dates, generateDates(fmt.Sprint(today.Year()+1), monthExpr, dayExpr)...)
			return dates
		case "odd-year":
			if today.Year()%2 == 0 {
				dates = append(dates, generateDates(fmt.Sprint(today.Year()), monthExpr, dayExpr)...)
				dates = append(dates, generateDates(fmt.Sprint(today.Year()+2), monthExpr, dayExpr)...)
			} else {
				dates = append(dates, generateDates(fmt.Sprint(today.Year()+1), monthExpr, dayExpr)...)
			}
			return dates
		case "even-year":
			if today.Year()%2 == 1 {
				dates = append(dates, generateDates(fmt.Sprint(today.Year()), monthExpr, dayExpr)...)
				dates = append(dates, generateDates(fmt.Sprint(today.Year()+2), monthExpr, dayExpr)...)
			} else {
				dates = append(dates, generateDates(fmt.Sprint(today.Year()+1), monthExpr, dayExpr)...)
			}
			return dates
		default:
			log.Fatalf("Unsupported year expression %q", yearExpr)
		}
	}

	year, _ := strconv.Atoi(yearExpr)

	if !text.IsNumber(monthExpr) {
		switch monthExpr {
		case "":
			fallthrough
		case "month":
			if today.Year() == year {
				// this month + next month
				dates = append(dates, generateDates(yearExpr, fmt.Sprintf("%02d", today.Month()), dayExpr)...)
				if today.Month() == time.December {
					dates = append(dates, generateDates(yearExpr, "01", dayExpr)...)
				} else {
					dates = append(dates, generateDates(yearExpr, fmt.Sprintf("%02d", today.Month()+1), dayExpr)...)
				}
			} else {
				// First month of a future year
				dates = append(dates, generateDates(yearExpr, "01", dayExpr)...)
			}
			return dates
		case "odd-month":
			if today.Year() == year {
				if today.Month()%2 == 0 {
					// this month + next odd month
					dates = append(dates, generateDates(yearExpr, fmt.Sprintf("%02d", today.Month()), dayExpr)...)
					if today.Month() == time.December {
						dates = append(dates, generateDates(yearExpr, "02", dayExpr)...)
					} else {
						dates = append(dates, generateDates(yearExpr, fmt.Sprintf("%02d", today.Month()+2), dayExpr)...)
					}
				} else {
					// next month (NB: +1 is safe as we know the current month is even)
					dates = append(dates, generateDates(yearExpr, fmt.Sprintf("%02d", today.Month()+1), dayExpr)...)
				}
			} else {
				// First odd month of a future year
				dates = append(dates, generateDates(yearExpr, "02", dayExpr)...)
			}
			return dates
		case "even-month":
			if today.Year() == year {
				if today.Month()%2 == 1 {
					// this month + next even month
					dates = append(dates, generateDates(yearExpr, fmt.Sprintf("%02d", today.Month()), dayExpr)...)
					if today.Month() == time.November {
						dates = append(dates, generateDates(yearExpr, "01", dayExpr)...)
					} else {
						dates = append(dates, generateDates(yearExpr, fmt.Sprintf("%02d", today.Month()+2), dayExpr)...)
					}
				} else {
					// next month
					if today.Month() == time.December {
						dates = append(dates, generateDates(yearExpr, "01", dayExpr)...)
					} else {
						dates = append(dates, generateDates(yearExpr, fmt.Sprintf("%02d", today.Month()+1), dayExpr)...)
					}
				}
			} else {
				// First even month of a future year
				dates = append(dates, generateDates(yearExpr, "01", dayExpr)...)
			}
			return dates
		default:
			log.Fatalf("Unsupported month expression %q", monthExpr)
		}
	}

	month, _ := strconv.Atoi(monthExpr)
	currentMonth := time.Month(month)
	start := time.Date(year, currentMonth, 1, 0, 0, 0, 0, time.UTC)

	// We know that dayExpr is not a number if we reach this block
	switch dayExpr {
	case "":
		fallthrough
	case "day":
		if today.Year() == year && today.Month() == time.Month(month) {
			dates = append(dates, generateDates(yearExpr, monthExpr, fmt.Sprintf("%02d", today.Day()+1))...)
			dates = append(dates, generateDates(yearExpr, monthExpr, "01")...) // end of month
		} else {
			dates = append(dates, generateDates(yearExpr, monthExpr, "01")...)
		}
		return dates
	case "monday":
		for start.Month() == currentMonth {
			start = start.AddDate(0, 0, 1)
			if start.Weekday() == time.Monday {
				dates = append(dates, start)
			}
		}
		return dates
	case "tuesday":
		for start.Month() == currentMonth {
			start = start.AddDate(0, 0, 1)
			if start.Weekday() == time.Tuesday {
				dates = append(dates, start)
			}
		}
		return dates
	case "wednesday":
		for start.Month() == currentMonth {
			start = start.AddDate(0, 0, 1)
			if start.Weekday() == time.Wednesday {
				dates = append(dates, start)
			}
		}
		return dates
	case "thursday":
		for start.Month() == currentMonth {
			start = start.AddDate(0, 0, 1)
			if start.Weekday() == time.Thursday {
				dates = append(dates, start)
			}
		}
		return dates
	case "friday":
		for start.Month() == currentMonth {
			start = start.AddDate(0, 0, 1)
			if start.Weekday() == time.Friday {
				dates = append(dates, start)
			}
		}
		return dates
	case "saturday":
		for start.Month() == currentMonth {
			start = start.AddDate(0, 0, 1)
			if start.Weekday() == time.Saturday {
				dates = append(dates, start)
			}
		}
		return dates
	case "sunday":
		for start.Month() == currentMonth {
			start = start.AddDate(0, 0, 1)
			if start.Weekday() == time.Sunday {
				dates = append(dates, start)
			}
		}
		return dates
	default:
		log.Fatalf("Unsupported day expression %q", dayExpr)
	}
	return dates
}

/* Database Management */

func (r *Reminder) Check() error {
	db := CurrentDB().Client()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = r.CheckWithTx(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}
	return nil

}

func (r *Reminder) CheckWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Checking reminder %s...", r.DescriptionRaw)
	r.LastCheckedAt = clock.Now()
	query := `
		UPDATE reminder
		SET last_checked_at = ?
		WHERE oid = ?;`
	_, err := tx.Exec(query,
		timeToSQL(r.LastCheckedAt),
		r.OID,
	)

	return err
}

func (r *Reminder) Save(tx *sql.Tx) error {
	var err error
	switch r.State() {
	case Added:
		err = r.InsertWithTx(tx)
	case Modified:
		err = r.UpdateWithTx(tx)
	case Deleted:
		err = r.DeleteWithTx(tx)
	default:
		err = r.CheckWithTx(tx)
	}
	r.new = false
	r.stale = false
	return err
}

func (r *Reminder) OldSave() error { // FIXME remove deprecated
	if !r.stale {
		return r.Check()
	}

	db := CurrentDB().Client()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = r.SaveWithTx(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	r.new = false
	r.stale = false

	return nil
}

func (r *Reminder) SaveWithTx(tx *sql.Tx) error { // FIXME remove deprecated
	if !r.stale {
		return r.CheckWithTx(tx)
	}

	now := clock.Now()
	r.UpdatedAt = now
	r.LastCheckedAt = now

	if !r.new {
		if err := r.UpdateWithTx(tx); err != nil {
			return err
		}
	} else {
		r.CreatedAt = now
		if err := r.InsertWithTx(tx); err != nil {
			return err
		}
	}

	r.new = false
	r.stale = false

	return nil
}

func (r *Reminder) InsertWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Inserting reminder %s...", r.DescriptionRaw)
	query := `
		INSERT INTO reminder(
			oid,
			file_oid,
			note_oid,
			relative_path,
			description_raw,
			description_markdown,
			description_html,
			description_text,
			tag,
			last_performed_at,
			next_performed_at,
			created_at,
			updated_at,
			last_checked_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	_, err := tx.Exec(query,
		r.OID,
		r.FileOID,
		r.NoteOID,
		r.RelativePath,
		r.DescriptionRaw,
		r.DescriptionMarkdown,
		r.DescriptionHTML,
		r.DescriptionText,
		r.Tag,
		timeToSQL(r.LastPerformedAt),
		timeToSQL(r.NextPerformedAt),
		timeToSQL(r.CreatedAt),
		timeToSQL(r.UpdatedAt),
		timeToSQL(r.LastCheckedAt),
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *Reminder) UpdateWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Updating reminder %s...", r.DescriptionRaw)
	query := `
		UPDATE reminder
		SET
			file_oid = ?,
			note_oid = ?,
			relative_path = ?,
			description_raw = ?,
			description_markdown = ?,
			description_html = ?,
			description_text = ?,
			tag = ?,
			last_performed_at = ?,
			next_performed_at = ?,
			updated_at = ?,
			last_checked_at = ?
		WHERE oid = ?;
	`
	_, err := tx.Exec(query,
		r.FileOID,
		r.NoteOID,
		r.RelativePath,
		r.DescriptionRaw,
		r.DescriptionMarkdown,
		r.DescriptionHTML,
		r.DescriptionText,
		r.Tag,
		timeToSQL(r.LastPerformedAt),
		timeToSQL(r.NextPerformedAt),
		timeToSQL(r.UpdatedAt),
		timeToSQL(r.LastCheckedAt),
		r.OID,
	)

	return err
}

func (r *Reminder) Delete() error {
	db := CurrentDB().Client()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = r.DeleteWithTx(tx)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (r *Reminder) DeleteWithTx(tx *sql.Tx) error {
	CurrentLogger().Debugf("Deleting reminder %s...", r.DescriptionRaw)
	query := `DELETE FROM reminder WHERE oid = ?;`
	_, err := tx.Exec(query, r.OID)
	return err
}

// CountReminders returns the total number of reminders.
func CountReminders() (int, error) {
	db := CurrentDB().Client()

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM reminder`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func FindReminders() ([]*Reminder, error) {
	return QueryReminders("")
}

func FindRemindersMatching(noteOID string, descriptionRaw string) ([]*Reminder, error) {
	return QueryReminders(`WHERE note_oid = ? and description_raw`, noteOID, descriptionRaw)
}

func LoadReminderByOID(oid string) (*Reminder, error) {
	return QueryReminder(`WHERE oid = ?`, oid)
}

func FindRemindersByUpcomingDate(deadline time.Time) ([]*Reminder, error) {
	return QueryReminders(`WHERE next_performed_at > ?`, timeToSQL(deadline))
}

func FindRemindersLastCheckedBefore(point time.Time, path string) ([]*Reminder, error) {
	return QueryReminders(`WHERE last_checked_at < ? AND relative_path LIKE ?`, timeToSQL(point), path+"%")
}

/* SQL Helpers */

func QueryReminder(whereClause string, args ...any) (*Reminder, error) {
	db := CurrentDB().Client()

	var r Reminder
	var lastPerformedAt string
	var nextPerformedAt string
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
			description_raw,
			description_markdown,
			description_html,
			description_text,
			tag,
			last_performed_at,
			next_performed_at,
			created_at,
			updated_at,
			last_checked_at
		FROM reminder
		%s;`, whereClause), args...).
		Scan(
			&r.OID,
			&r.FileOID,
			&r.NoteOID,
			&r.RelativePath,
			&r.DescriptionRaw,
			&r.DescriptionMarkdown,
			&r.DescriptionHTML,
			&r.DescriptionText,
			&r.Tag,
			&lastPerformedAt,
			&nextPerformedAt,
			&createdAt,
			&updatedAt,
			&lastCheckedAt,
		); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	r.LastPerformedAt = timeFromSQL(lastPerformedAt)
	r.NextPerformedAt = timeFromSQL(nextPerformedAt)
	r.CreatedAt = timeFromSQL(createdAt)
	r.UpdatedAt = timeFromSQL(updatedAt)
	r.LastCheckedAt = timeFromSQL(lastCheckedAt)

	return &r, nil
}

func QueryReminders(whereClause string, args ...any) ([]*Reminder, error) {
	db := CurrentDB().Client()

	var reminders []*Reminder

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			oid,
			file_oid,
			note_oid,
			relative_path,
			description_raw,
			description_markdown,
			description_html,
			description_text,
			tag,
			last_performed_at,
			next_performed_at,
			created_at,
			updated_at,
			last_checked_at
		FROM reminder
		%s;`, whereClause), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var r Reminder
		var lastPerformedAt string
		var nextPerformedAt string
		var createdAt string
		var updatedAt string
		var lastCheckedAt string

		err = rows.Scan(
			&r.OID,
			&r.FileOID,
			&r.NoteOID,
			&r.RelativePath,
			&r.DescriptionRaw,
			&r.DescriptionMarkdown,
			&r.DescriptionHTML,
			&r.DescriptionText,
			&r.Tag,
			&lastPerformedAt,
			&nextPerformedAt,
			&createdAt,
			&updatedAt,
			&lastCheckedAt,
		)
		if err != nil {
			return nil, err
		}

		r.LastPerformedAt = timeFromSQL(lastPerformedAt)
		r.NextPerformedAt = timeFromSQL(nextPerformedAt)
		r.CreatedAt = timeFromSQL(createdAt)
		r.UpdatedAt = timeFromSQL(updatedAt)
		r.LastCheckedAt = timeFromSQL(lastCheckedAt)
		reminders = append(reminders, &r)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return reminders, err
}
