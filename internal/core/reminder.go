package core

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/julien-sobczak/the-notetaker/pkg/clock"
	"github.com/julien-sobczak/the-notetaker/pkg/markdown"
	"github.com/julien-sobczak/the-notetaker/pkg/text"
)

type Reminder struct {
	ID int64

	// File
	FileID int64
	File   *File // Lazy-loaded

	// Note representing the flashcard
	NoteID int64
	Note   *Note // Lazy-loaded

	// Description
	DescriptionRaw      string
	DescriptionMarkdown string
	DescriptionHTML     string
	DescriptionText     string

	// Tag value containig the formula to determine the next occurence
	Tag string

	// Timestamps to track progress
	LastPerformedAt time.Time
	NextPerformedAt time.Time

	// Timestamps to track changes
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     time.Time
	LastCheckedAt time.Time
}

// NewReminder instantiates a new reminder.
func NewReminder(n *Note, descriptionRaw, tag string) (*Reminder, error) {
	r := &Reminder{
		ID:             0,
		FileID:         n.FileID,
		NoteID:         n.ID,
		DescriptionRaw: descriptionRaw,
		Tag:            tag,
	}

	r.DescriptionMarkdown = markdown.ToMarkdown(r.DescriptionRaw)
	r.DescriptionHTML = markdown.ToHTML(r.DescriptionMarkdown)
	r.DescriptionText = markdown.ToText(r.DescriptionMarkdown)

	err := r.Next()
	if err != nil {
		return nil, err
	}

	return r, nil
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

func (r *Reminder) Save() error {
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

	return nil
}

func (r *Reminder) SaveWithTx(tx *sql.Tx) error {
	now := clock.Now()
	r.UpdatedAt = now
	r.LastCheckedAt = now

	if r.ID != 0 {
		return r.UpdateWithTx(tx)
	} else {
		r.CreatedAt = now
		return r.InsertWithTx(tx)
	}
}

func (r *Reminder) InsertWithTx(tx *sql.Tx) error {
	query := `
		INSERT INTO reminder(
			id,
			file_id,
			note_id,
			description_raw,
			description_markdown,
			description_html,
			description_text,
			tag,
			last_performed_at,
			next_performed_at,
			created_at,
			updated_at,
			deleted_at,
			last_checked_at
		)
		VALUES (NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
	`
	res, err := tx.Exec(query,
		r.FileID,
		r.NoteID,
		r.DescriptionRaw,
		r.DescriptionMarkdown,
		r.DescriptionHTML,
		r.DescriptionText,
		r.Tag,
		timeToSQL(r.LastPerformedAt),
		timeToSQL(r.NextPerformedAt),
		timeToSQL(r.CreatedAt),
		timeToSQL(r.UpdatedAt),
		timeToSQL(r.DeletedAt),
		timeToSQL(r.LastCheckedAt),
	)
	if err != nil {
		return err
	}

	var id int64
	if id, err = res.LastInsertId(); err != nil {
		return err
	}
	r.ID = id

	return nil
}

func (r *Reminder) UpdateWithTx(tx *sql.Tx) error {
	query := `
		UPDATE reminder
		SET
			file_id = ?,
			note_id = ?,
			description_raw = ?,
			description_markdown = ?,
			description_html = ?,
			description_text = ?,
			tag = ?,
			last_performed_at = ?,
			next_performed_at = ?,
			updated_at = ?,
			deleted_at = ?,
			last_checked_at = ?
		)
		WHERE id = ?;
	`
	_, err := tx.Exec(query,
		r.FileID,
		r.NoteID,
		r.DescriptionRaw,
		r.DescriptionMarkdown,
		r.DescriptionHTML,
		r.DescriptionText,
		r.Tag,
		timeToSQL(r.LastPerformedAt),
		timeToSQL(r.NextPerformedAt),
		timeToSQL(r.UpdatedAt),
		timeToSQL(r.DeletedAt),
		timeToSQL(r.LastCheckedAt),
		r.ID,
	)

	return err
}

// CountReminders returns the total number of reminders.
func CountReminders() (int, error) {
	db := CurrentDB().Client()

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM reminder WHERE deleted_at = ''`).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func FindReminders() ([]*Reminder, error) {
	return QueryReminders("")
}

func LoadReminderByID(id int64) (*Reminder, error) {
	return QueryReminder(`WHERE id = ?`, id)
}

func FindRemindersByUpcomingDate(deadline time.Time) ([]*Reminder, error) {
	return QueryReminders(`WHERE next_performed_at > ?`, timeToSQL(deadline))
}

/* SQL Helpers */

func QueryReminder(whereClause string, args ...any) (*Reminder, error) {
	db := CurrentDB().Client()

	var r Reminder
	var lastPerformedAt string
	var nextPerformedAt string
	var createdAt string
	var updatedAt string
	var deletedAt string
	var lastCheckedAt string

	// Query for a value based on a single row.
	if err := db.QueryRow(fmt.Sprintf(`
		SELECT
			id,
			file_id,
			note_id,
			description_raw,
			description_markdown,
			description_html,
			description_text,
			tag,
			last_performed_at,
			next_performed_at,
			created_at,
			updated_at,
			deleted_at,
			last_checked_at
		FROM reminder
		%s;`, whereClause), args...).
		Scan(
			&r.ID,
			&r.FileID,
			&r.NoteID,
			&r.DescriptionRaw,
			&r.DescriptionMarkdown,
			&r.DescriptionHTML,
			&r.DescriptionText,
			&r.Tag,
			&lastPerformedAt,
			&nextPerformedAt,
			&createdAt,
			&updatedAt,
			&deletedAt,
			&lastCheckedAt,
		); err != nil {

		return nil, err
	}

	r.LastPerformedAt = timeFromSQL(lastPerformedAt)
	r.NextPerformedAt = timeFromSQL(nextPerformedAt)
	r.CreatedAt = timeFromSQL(createdAt)
	r.UpdatedAt = timeFromSQL(updatedAt)
	r.DeletedAt = timeFromSQL(deletedAt)
	r.LastCheckedAt = timeFromSQL(lastCheckedAt)

	return &r, nil
}

func QueryReminders(whereClause string, args ...any) ([]*Reminder, error) {
	db := CurrentDB().Client()

	var reminders []*Reminder

	rows, err := db.Query(fmt.Sprintf(`
		SELECT
			id,
			file_id,
			note_id,
			description_raw,
			description_markdown,
			description_html,
			description_text,
			tag,
			last_performed_at,
			next_performed_at,
			created_at,
			updated_at,
			deleted_at,
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
		var deletedAt string
		var lastCheckedAt string

		err = rows.Scan(
			&r.ID,
			&r.FileID,
			&r.NoteID,
			&r.DescriptionRaw,
			&r.DescriptionMarkdown,
			&r.DescriptionHTML,
			&r.DescriptionText,
			&r.Tag,
			&lastPerformedAt,
			&nextPerformedAt,
			&createdAt,
			&updatedAt,
			&deletedAt,
			&lastCheckedAt,
		)
		if err != nil {
			return nil, err
		}

		r.LastPerformedAt = timeFromSQL(lastPerformedAt)
		r.NextPerformedAt = timeFromSQL(nextPerformedAt)
		r.CreatedAt = timeFromSQL(createdAt)
		r.UpdatedAt = timeFromSQL(updatedAt)
		r.DeletedAt = timeFromSQL(deletedAt)
		r.LastCheckedAt = timeFromSQL(lastCheckedAt)
		reminders = append(reminders, &r)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return reminders, err
}
