package oid

import (
	"fmt"
)

type OID string

const Nil = OID("")
const Missing string = "4044044044044044044044044044044044044040"

func (o OID) IsNil() bool {
	return string(o) == ""
}

// Prefix returns the first two characters of the OID as used in .nt/objects.
func (o OID) Prefix() string {
	if o.IsNil() {
		return ""
	}
	return string(o)[0:2]
}

// RelativePath returns the relative path inside .nt/objects.
func (o OID) RelativePath() string {
	// We use the first two characters to spread objects into different directories
	// (same as .git/objects/) to avoid having a large unpractical directory.
	return fmt.Sprintf("%s/%s", o.Prefix(), o)
}

// String returns the OID as a string.
func (o OID) String() string {
	return string(o)
}

/* Constructors */

func New() OID {
	return generator.New()
}
func NewFromBytes(b []byte) OID {
	return generator.NewFromBytes(b)
}

/* Parser */

// MustParse parses an OID or panic if the OID format is not valid.
func MustParse(s string) OID {
	if len(s) != 40 {
		panic("Invalid OID")
	}
	return OID(s)
}

// ParseOrNil parses an OID or returns NilOID.
func ParseOrNil(s string) OID {
	if len(s) != 40 {
		return Nil
	}
	return OID(s)
}
