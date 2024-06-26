package main

import (
	"crypto/sha1"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("Missing command")
	}
	command := os.Args[1]
	switch command {
	case "add":
		CurrentRepository().Add()
	case "commit":
		CurrentDB().Commit()
	default:
		log.Fatalf("Unsupported command %q", command)
	}
}

/* Helpers */

// OIDToPath converts an oid to a file path.
func OIDToPath(oid string) string {
	// We use the first two characters to spread objects into different directories
	// (same as .git/objects/) to avoid having a large unpractical directory.
	return oid[0:2] + "/" + oid
}

// Hash is an utility to determine a MD5 hash (acceptable as not used for security reasons).
func Hash(bytes []byte) string {
	h := sha1.New()
	h.Write(bytes)
	return fmt.Sprintf("%x", h.Sum(nil))
}

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
