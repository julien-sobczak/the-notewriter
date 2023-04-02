package core

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHooks(t *testing.T) {

	t.Run("Valid", func(t *testing.T) {
		SetUpCollectionFromGoldenDirNamed(t, "TestHooks")

		err := CurrentCollection().Add(".")
		require.NoError(t, err)

		notes, err := CurrentCollection().SearchNotes(`@title:dup`)
		require.NoError(t, err)
		require.Len(t, notes, 1)
		note := notes[0]
		err = note.RunHooks()
		require.NoError(t, err)
	})

	t.Run("Missing", func(t *testing.T) {
		SetUpCollectionFromGoldenDirNamed(t, "TestHooks")

		err := CurrentCollection().Add(".")
		require.NoError(t, err)

		notes, err := CurrentCollection().SearchNotes(`@title:missing`)
		require.NoError(t, err)
		require.Len(t, notes, 1)
		note := notes[0]
		err = note.RunHooks()
		require.ErrorContains(t, err, "no executable")
	})

	t.Run("Not executable", func(t *testing.T) {
		SetUpCollectionFromGoldenDirNamed(t, "TestHooks")

		err := CurrentCollection().Add(".")
		require.NoError(t, err)

		notes, err := CurrentCollection().SearchNotes(`@title:program`)
		require.NoError(t, err)
		require.Len(t, notes, 1)
		note := notes[0]
		err = note.RunHooks()
		require.ErrorContains(t, err, "no executable")
	})

	t.Run("Multiple executables", func(t *testing.T) {
		SetUpCollectionFromGoldenDirNamed(t, "TestHooks")

		err := CurrentCollection().Add(".")
		require.NoError(t, err)

		notes, err := CurrentCollection().SearchNotes(`@title:multiple`)
		require.NoError(t, err)
		require.Len(t, notes, 1)
		note := notes[0]
		err = note.RunHooks()
		require.ErrorContains(t, err, "multiple possible executable")
	})

	t.Run("Error", func(t *testing.T) {
		SetUpCollectionFromGoldenDirNamed(t, "TestHooks")

		err := CurrentCollection().Add(".")
		require.NoError(t, err)

		notes, err := CurrentCollection().SearchNotes(`@title:error`)
		require.NoError(t, err)
		require.Len(t, notes, 1)
		note := notes[0]
		err = note.RunHooks()
		require.ErrorContains(t, err, "exit status 1")
	})

}
