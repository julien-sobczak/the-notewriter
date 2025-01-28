package core

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notewriter/internal/markdown"
	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"github.com/julien-sobczak/the-notewriter/pkg/text"
	"gopkg.in/yaml.v3"
)

type Reminder struct {
	OID oid.OID `yaml:"oid" json:"oid"`

	// Pack file where this object belongs
	PackFileOID oid.OID `yaml:"packfile_oid" json:"packfile_oid"`

	// File
	FileOID oid.OID `yaml:"file_oid" json:"file_oid"`

	// Note representing the flashcard
	NoteOID oid.OID `yaml:"note_oid" json:"note_oid"`
	Note    *Note   `yaml:"-" json:"-"` // Lazy-loaded

	// The filepath of the file containing the note (denormalized field)
	RelativePath string `yaml:"relative_path" json:"relative_path"`

	// Description
	Description markdown.Document `yaml:"description" json:"description"`

	// Tag value containig the formula to determine the next occurence
	Tag string `yaml:"tag" json:"tag"`

	// Timestamps to track progress
	LastPerformedAt time.Time `yaml:"last_performed_at" json:"last_performed_at"`
	NextPerformedAt time.Time `yaml:"next_performed_at" json:"next_performed_at"`

	// Timestamps to track changes
	CreatedAt     time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt     time.Time `yaml:"updated_at" json:"updated_at"`
	DeletedAt     time.Time `yaml:"deleted_at,omitempty" json:"deleted_at,omitempty"`
	LastIndexedAt time.Time `yaml:"last_indexed_at,omitempty" json:"last_indexed_at,omitempty"`
}

func NewOrExistingReminder(packFile *PackFile, note *Note, parsedReminder *ParsedReminder) (*Reminder, error) {
	// Try to find an existing note (instead of recreating it from scratch after every change)
	existingReminder, err := CurrentRepository().FindMatchingReminder(note, parsedReminder)
	if err != nil {
		return nil, err
	}
	if existingReminder != nil {
		existingReminder.update(packFile, note, parsedReminder)
		return existingReminder, nil
	}
	return NewReminder(packFile, note, parsedReminder)
}

// NewReminder instantiates a new reminder.
func NewReminder(packFile *PackFile, note *Note, parsedReminder *ParsedReminder) (*Reminder, error) {
	r := &Reminder{
		OID:           oid.New(),
		PackFileOID:   packFile.OID,
		FileOID:       note.FileOID,
		NoteOID:       note.OID,
		RelativePath:  note.RelativePath,
		Tag:           parsedReminder.Tag,
		Description:   parsedReminder.Description,
		CreatedAt:     packFile.CTime,
		UpdatedAt:     packFile.CTime,
		LastIndexedAt: packFile.CTime,
	}

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

func (r *Reminder) UniqueOID() oid.OID {
	return r.OID
}

func (r *Reminder) ModificationTime() time.Time {
	return r.UpdatedAt
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

func (r *Reminder) Relations() []*Relation {
	return nil
}

func (r Reminder) String() string {
	return fmt.Sprintf("reminder %s [%s]", r.Tag, r.OID)
}

/* Update */

func (r *Reminder) update(packFile *PackFile, note *Note, parsedReminder *ParsedReminder) error {
	stale := false

	if r.FileOID != note.FileOID {
		r.FileOID = note.FileOID
		stale = true
	}
	if r.NoteOID != note.OID {
		r.NoteOID = note.OID
		r.Note = note
		stale = true
	}
	if r.Description != parsedReminder.Description {
		r.Description = parsedReminder.Description
		stale = true
	}
	if r.Tag != parsedReminder.Tag {
		r.Tag = parsedReminder.Tag
		stale = true
		err := r.Next()
		if err != nil {
			return err
		}
	}

	r.PackFileOID = packFile.OID
	r.LastIndexedAt = packFile.CTime

	if stale {
		r.UpdatedAt = packFile.CTime
	}

	return nil
}

/* State Management */

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

/* Format */

func (r *Reminder) ToYAML() string {
	return ToBeautifulYAML(r)
}

func (r *Reminder) ToJSON() string {
	return ToBeautifulJSON(r)
}

func (r *Reminder) ToMarkdown() string {
	var sb strings.Builder
	sb.WriteString(string(r.Description))
	sb.WriteRune(' ')
	sb.WriteRune('`')
	sb.WriteString(r.Tag)
	sb.WriteRune('`')
	return sb.String()
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

	if len(possibleFutureDates) == 0 {
		// Must not happen
		return time.Time{}, fmt.Errorf("no date can be determined for reminder %q", originalExpr)
	}

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

func (r *Reminder) Save() error {
	CurrentLogger().Debugf("Saving reminder %s...", r.Description)
	query := `
		INSERT INTO reminder(
			oid,
			packfile_oid,
			file_oid,
			note_oid,
			relative_path,
			description,
			tag,
			last_performed_at,
			next_performed_at,
			created_at,
			updated_at,
			last_indexed_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(oid) DO UPDATE SET
			packfile_oid = ?,
			file_oid = ?,
			note_oid = ?,
			relative_path = ?,
			description = ?,
			tag = ?,
			last_performed_at = ?,
			next_performed_at = ?,
			updated_at = ?,
			last_indexed_at = ?
		;
	`
	_, err := CurrentDB().Client().Exec(query,
		// Insert
		r.OID,
		r.PackFileOID,
		r.FileOID,
		r.NoteOID,
		r.RelativePath,
		r.Description,
		r.Tag,
		timeToSQL(r.LastPerformedAt),
		timeToSQL(r.NextPerformedAt),
		timeToSQL(r.CreatedAt),
		timeToSQL(r.UpdatedAt),
		timeToSQL(r.LastIndexedAt),
		// Update
		r.PackFileOID,
		r.FileOID,
		r.NoteOID,
		r.RelativePath,
		r.Description,
		r.Tag,
		timeToSQL(r.LastPerformedAt),
		timeToSQL(r.NextPerformedAt),
		timeToSQL(r.UpdatedAt),
		timeToSQL(r.LastIndexedAt),
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *Reminder) Delete() error {
	CurrentLogger().Debugf("Deleting reminder %s...", r.Description)
	query := `DELETE FROM reminder WHERE oid = ? AND packfile_oid = ?;`
	_, err := CurrentDB().Client().Exec(query, r.OID, r.PackFileOID)
	return err
}

/* SQL Queries */

// CountReminders returns the total number of reminders.
func (r *Repository) CountReminders() (int, error) {
	var count int
	if err := CurrentDB().Client().QueryRow(`SELECT count(*) FROM reminder`).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repository) FindReminders() ([]*Reminder, error) {
	return QueryReminders(CurrentDB().Client(), "")
}

func (r *Repository) FindMatchingReminder(note *Note, parsedReminder *ParsedReminder) (*Reminder, error) {
	return QueryReminder(CurrentDB().Client(), `WHERE note_oid = ? and description = ?`, note.OID, parsedReminder.Description)
}

func (r *Repository) FindMatchingReminders(noteOID oid.OID, descriptionRaw string) ([]*Reminder, error) {
	return QueryReminders(CurrentDB().Client(), `WHERE note_oid = ? and description = ?`, noteOID, descriptionRaw)
}

func (r *Repository) LoadReminderByOID(oid oid.OID) (*Reminder, error) {
	return QueryReminder(CurrentDB().Client(), `WHERE oid = ?`, oid)
}

func (r *Repository) FindRemindersByUpcomingDate(deadline time.Time) ([]*Reminder, error) {
	return QueryReminders(CurrentDB().Client(), `WHERE next_performed_at > ?`, timeToSQL(deadline))
}

func (r *Repository) FindRemindersLastCheckedBefore(point time.Time, path string) ([]*Reminder, error) {
	if path == "." {
		path = ""
	}
	return QueryReminders(CurrentDB().Client(), `WHERE last_indexed_at < ? AND relative_path LIKE ?`, timeToSQL(point), path+"%")
}

/* SQL Helpers */

func QueryReminder(db SQLClient, whereClause string, args ...any) (*Reminder, error) {
	var r Reminder
	var lastPerformedAt string
	var nextPerformedAt string
	var createdAt string
	var updatedAt string
	var lastIndexedAt string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			oid,
			packfile_oid,
			file_oid,
			note_oid,
			relative_path,
			description,
			tag,
			last_performed_at,
			next_performed_at,
			created_at,
			updated_at,
			last_indexed_at
		FROM reminder
		%s;`, whereClause), args...).
		Scan(
			&r.OID,
			&r.PackFileOID,
			&r.FileOID,
			&r.NoteOID,
			&r.RelativePath,
			&r.Description,
			&r.Tag,
			&lastPerformedAt,
			&nextPerformedAt,
			&createdAt,
			&updatedAt,
			&lastIndexedAt,
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
	r.LastIndexedAt = timeFromSQL(lastIndexedAt)

	return &r, nil
}

func QueryReminders(db SQLClient, whereClause string, args ...any) ([]*Reminder, error) {
	var reminders []*Reminder

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			oid,
			packfile_oid,
			file_oid,
			note_oid,
			relative_path,
			description,
			tag,
			last_performed_at,
			next_performed_at,
			created_at,
			updated_at,
			last_indexed_at
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
		var lastIndexedAt string

		err = rows.Scan(
			&r.OID,
			&r.PackFileOID,
			&r.FileOID,
			&r.NoteOID,
			&r.RelativePath,
			&r.Description,
			&r.Tag,
			&lastPerformedAt,
			&nextPerformedAt,
			&createdAt,
			&updatedAt,
			&lastIndexedAt,
		)
		if err != nil {
			return nil, err
		}

		r.LastPerformedAt = timeFromSQL(lastPerformedAt)
		r.NextPerformedAt = timeFromSQL(nextPerformedAt)
		r.CreatedAt = timeFromSQL(createdAt)
		r.UpdatedAt = timeFromSQL(updatedAt)
		r.LastIndexedAt = timeFromSQL(lastIndexedAt)
		reminders = append(reminders, &r)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return reminders, err
}
