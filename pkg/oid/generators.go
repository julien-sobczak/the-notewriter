package oid

import (
	"crypto/sha1"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

var generator Generator = &UniqueGenerator{}

/* Generator */

type Generator interface {
	New() OID
	NewFromBytes(b []byte) OID
}

// Reset restores the original unique OID generator.
// Useful in tests with a defer after overriding the default generator.
func Reset() {
	generator = &UniqueGenerator{}
}

/*
 * UniqueGenerator
 */

// UniqueGenerator is a production-grade Generator returning unique, random OIDs.
type UniqueGenerator struct{}

func NewUniqueGenerator() *UniqueGenerator {
	return &UniqueGenerator{}
}

// New generates an OID.
// Every call generates a new unique OID.
func (g *UniqueGenerator) New() OID {
	// We use the same "format" as Git (=40-length string) but use a content hash only for blob objects.
	// We use a randomly generated ID for other objects that is fixed even if objects are updated.

	// Ex (Git): 5e3f1b351782c017590b4b70fee709bf9c83b050
	// Ex (UUIDv4): 123e4567-e89b-12d3-a456-426655440000

	// Algorithm:
	// Remove `-` + add 8 random characters
	oid := strings.ReplaceAll(uuid.New().String()+uuid.New().String(), "-", "")[0:40]
	return OID(oid)
}

// NewFromBytes generates an OID based on bytes.
// The same bytes will generate the same OID.
func (g *UniqueGenerator) NewFromBytes(b []byte) OID {
	h := sha1.New()
	h.Write(b)

	// This gets the finalized hash result as a byte
	// slice. The argument to `Sum` can be used to append
	// to an existing byte slice: it usually isn't needed.
	bs := h.Sum(nil)

	// SHA1 values are often printed in hex, for example
	// in git commits. Use the `%x` format verb to convert
	// a hash results to a hex string.
	return OID(fmt.Sprintf("%x", bs))
}

/*
 * SuiteGenerator
 */

// SuiteGenerator returns a predefined suite of OIDs.
// This generator is useful for tests when OIDs are relevant for the test case.
type SuiteGenerator struct {
	nextOIDs []string
}

func NewSuiteGenerator(nextOIDs ...string) *SuiteGenerator {
	return &SuiteGenerator{nextOIDs: nextOIDs}
}

func (g *SuiteGenerator) New() OID {
	return g.nextOID()
}

func (g *SuiteGenerator) NewFromBytes(b []byte) OID {
	return g.nextOID()
}

func (g *SuiteGenerator) nextOID() OID {
	if len(g.nextOIDs) > 0 {
		oid, nextOIDs := g.nextOIDs[0], g.nextOIDs[1:]
		g.nextOIDs = nextOIDs
		return OID(oid)
	}
	panic("No more OIDs")
}

/*
 * FixedGenerator
 */

// FixedGenerator returns always the same OID.
// This generator is useful for tests when OIDs are relevant for the test case.
type FixedGenerator struct {
	oid OID
}

func NewFixedGenerator(oid OID) *FixedGenerator {
	return &FixedGenerator{oid: oid}
}

func (g *FixedGenerator) New() OID {
	return g.oid
}

func (g *FixedGenerator) NewFromBytes(b []byte) OID {
	return g.oid
}

/*
 * SequenceGenerator
 */

// SequenceGenerator returns numbered OIDs in a predictable format.
// This generator is useful for tests when checking different objects.
type SequenceGenerator struct {
	count int
}

func NewSequenceGenerator() *SequenceGenerator {
	return &SequenceGenerator{count: 0}
}

func (g *SequenceGenerator) New() OID {
	g.count++
	return OID(fmt.Sprintf("%040d", g.count))
}

func (g *SequenceGenerator) NewFromBytes(b []byte) OID {
	return g.New()
}
