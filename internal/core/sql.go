package core

import "time"

// timeToSQL converts a time struct to a string representation compatible with SQLite.
func timeToSQL(date time.Time) string {
	if date.IsZero() {
		return ""
	}
	dateStr := date.Format(time.RFC3339Nano)
	return dateStr
}

// timeToSQL parses a string representation of a time to a time struct.
func timeFromSQL(dateStr string) time.Time {
	date, err := time.Parse(time.RFC3339Nano, dateStr)
	if err != nil {
		return time.Time{}
	}
	return date
}
