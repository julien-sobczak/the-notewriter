package core

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHooks(t *testing.T) {

	t.Run("Valid", func(t *testing.T) {
		SetUpRepositoryFromGoldenDirNamed(t, "TestHooks")

		err := CurrentRepository().Add(AnyPath)
		require.NoError(t, err)

		notes, err := CurrentRepository().SearchNotes(`slug:hooks-todo-dup`)
		require.NoError(t, err)
		require.Len(t, notes, 1)
		note := notes[0]
		err = note.RunHooks(nil)
		require.NoError(t, err)
	})

	t.Run("Missing", func(t *testing.T) {
		SetUpRepositoryFromGoldenDirNamed(t, "TestHooks")

		err := CurrentRepository().Add(AnyPath)
		require.NoError(t, err)

		notes, err := CurrentRepository().SearchNotes(`slug:hooks-note-missing`)
		require.NoError(t, err)
		require.Len(t, notes, 1)
		note := notes[0]
		err = note.RunHooks(nil)
		require.ErrorContains(t, err, "no executable")
	})

	t.Run("Not executable", func(t *testing.T) {
		SetUpRepositoryFromGoldenDirNamed(t, "TestHooks")

		err := CurrentRepository().Add(AnyPath)
		require.NoError(t, err)

		notes, err := CurrentRepository().SearchNotes(`slug:hooks-note-program`)
		require.NoError(t, err)
		require.Len(t, notes, 1)
		note := notes[0]
		err = note.RunHooks(nil)
		require.ErrorContains(t, err, "no executable")
	})

	t.Run("Multiple executables", func(t *testing.T) {
		SetUpRepositoryFromGoldenDirNamed(t, "TestHooks")

		err := CurrentRepository().Add(AnyPath)
		require.NoError(t, err)

		notes, err := CurrentRepository().SearchNotes(`slug:hooks-note-multiple`)
		require.NoError(t, err)
		require.Len(t, notes, 1)
		note := notes[0]
		err = note.RunHooks(nil)
		require.ErrorContains(t, err, "multiple possible executable")
	})

	t.Run("Error", func(t *testing.T) {
		SetUpRepositoryFromGoldenDirNamed(t, "TestHooks")

		err := CurrentRepository().Add(AnyPath)
		require.NoError(t, err)

		notes, err := CurrentRepository().SearchNotes(`slug:hooks-note-error`)
		require.NoError(t, err)
		require.Len(t, notes, 1)
		note := notes[0]
		err = note.RunHooks(nil)
		require.ErrorContains(t, err, "exit status 1")
	})

}
