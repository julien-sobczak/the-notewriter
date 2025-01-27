package oid

import "testing"

// UseNext configures a predefined list of OID
func UseNext(t *testing.T, oids ...string) {
	generator = NewSuiteGenerator(oids...)
	t.Cleanup(Reset)
}

// UseFixed configures a fixed OID value
func UseFixed(t *testing.T, value OID) {
	generator = NewFixedGenerator(value)
	t.Cleanup(Reset)
}

// UseSequence configures a predefined sequence
func UseSequence(t *testing.T) {
	generator = NewSequenceGenerator()
	t.Cleanup(Reset)
}
