package oid_test

import (
	"testing"

	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUniqueGenerator(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		gen := oid.NewUniqueGenerator()

		oid1 := gen.New()
		oid2 := gen.New()

		require.NotNil(t, oid1)
		require.NotNil(t, oid2)
		assert.NotEqual(t, oid1, oid2)
		assert.Len(t, oid1, 40)
		assert.Len(t, oid2, 40)
	})

	t.Run("NewFromBytes", func(t *testing.T) {
		gen := oid.NewUniqueGenerator()

		oid1 := gen.NewFromBytes([]byte("test data"))
		oid2 := gen.NewFromBytes([]byte("test data"))

		require.NotNil(t, oid1)
		require.NotNil(t, oid2)
		assert.Equal(t, oid1, oid2)
		assert.Len(t, oid1, 40)
	})

}

func TestSuiteGenerator(t *testing.T) {

	t.Run("New", func(t *testing.T) {
		gen := oid.NewSuiteGenerator(
			"1234567890abcdef1234567890abcdef12345678",
			"abcdef1234567890abcdef1234567890abcdef12",
		)

		oid1 := gen.New()
		oid2 := gen.New()

		require.NotNil(t, oid1)
		require.NotNil(t, oid2)
		assert.Equal(t, oid.OID("1234567890abcdef1234567890abcdef12345678"), oid1)
		assert.Equal(t, oid.OID("abcdef1234567890abcdef1234567890abcdef12"), oid2)
	})

	t.Run("NewFromBytes", func(t *testing.T) {
		gen := oid.NewSuiteGenerator(
			"1234567890abcdef1234567890abcdef12345678",
			"abcdef1234567890abcdef1234567890abcdef12",
		)

		oid1 := gen.NewFromBytes([]byte("test data"))
		oid2 := gen.NewFromBytes([]byte("test data"))

		require.NotNil(t, oid1)
		require.NotNil(t, oid2)
		assert.Equal(t, oid.OID("1234567890abcdef1234567890abcdef12345678"), oid1)
		assert.Equal(t, oid.OID("abcdef1234567890abcdef1234567890abcdef12"), oid2)
	})
}

func TestFixedGenerator(t *testing.T) {

	t.Run("New", func(t *testing.T) {
		gen := oid.NewFixedGenerator("1234567890abcdef1234567890abcdef12345678")

		oid1 := gen.New()
		oid2 := gen.New()

		require.NotNil(t, oid1)
		require.NotNil(t, oid2)
		assert.Equal(t, oid.OID("1234567890abcdef1234567890abcdef12345678"), oid1)
		assert.Equal(t, oid.OID("1234567890abcdef1234567890abcdef12345678"), oid2)
	})

	t.Run("NewFromBytes", func(t *testing.T) {
		gen := oid.NewFixedGenerator("1234567890abcdef1234567890abcdef12345678")

		oid1 := gen.NewFromBytes([]byte("test data"))
		oid2 := gen.NewFromBytes([]byte("test data"))

		require.NotNil(t, oid1)
		require.NotNil(t, oid2)
		assert.Equal(t, oid.OID("1234567890abcdef1234567890abcdef12345678"), oid1)
		assert.Equal(t, oid.OID("1234567890abcdef1234567890abcdef12345678"), oid2)
	})
}

func TestSequenceGenerator(t *testing.T) {

	t.Run("New", func(t *testing.T) {
		gen := oid.NewSequenceGenerator()

		oid1 := gen.New()
		oid2 := gen.New()

		require.NotNil(t, oid1)
		require.NotNil(t, oid2)
		assert.Equal(t, oid.OID("1000000000000000000000000000000000000000"), oid1)
		assert.Equal(t, oid.OID("2000000000000000000000000000000000000000"), oid2)
	})

	t.Run("NewFromBytes", func(t *testing.T) {
		gen := oid.NewSequenceGenerator()

		oid1 := gen.NewFromBytes([]byte("test data"))
		oid2 := gen.NewFromBytes([]byte("test data"))

		require.NotNil(t, oid1)
		require.NotNil(t, oid2)
		assert.Equal(t, oid.OID("1000000000000000000000000000000000000000"), oid1)
		assert.Equal(t, oid.OID("2000000000000000000000000000000000000000"), oid2)
	})
}
