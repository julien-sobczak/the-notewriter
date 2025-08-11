package core

import (
	"testing"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNewOperations(t *testing.T) {

	t.Run("MarkNote", func(t *testing.T) {
		oid := oid.OID("42d74d967d9b4e989502647ac510777ca1e22f4a")
		op := NewOperationMarkNote(oid)

		assert.Equal(t, oid, op.ObjectOID)
		assert.Equal(t, "mark-note", op.Name)
		assert.WithinDuration(t, clock.Now(), op.Timestamp, time.Second)
	})

	t.Run("UnmarkNote", func(t *testing.T) {
		oid := oid.OID("42d74d967d9b4e989502647ac510777ca1e22f4a")
		op := NewOperationUnmarkNote(oid)

		assert.Equal(t, oid, op.ObjectOID)
		assert.Equal(t, "unmark-note", op.Name)
		assert.WithinDuration(t, clock.Now(), op.Timestamp, time.Second)
	})

	t.Run("AddAnnotation", func(t *testing.T) {
		oid := oid.OID("42d74d967d9b4e989502647ac510777ca1e22f4a")
		annotation := Annotation{
			OID:  "f97ba6134cb447f88ae831ff745cf259ebe7d9ad",
			Text: "Use Markdown emphasis",
		}
		op := NewOperationAddAnnotation(oid, annotation)

		assert.Equal(t, oid, op.ObjectOID)
		assert.Equal(t, "add-annotation", op.Name)
		assert.WithinDuration(t, clock.Now(), op.Timestamp, time.Second)
		assert.Contains(t, op.Extras, "annotation")
		assert.IsType(t, Annotation{}, op.Extras["annotation"])
	})

	t.Run("RemoveAnnotation", func(t *testing.T) {
		oid := oid.OID("42d74d967d9b4e989502647ac510777ca1e22f4a")
		annotation := Annotation{
			OID: "f97ba6134cb447f88ae831ff745cf259ebe7d9ad",
		}
		op := NewOperationRemoveAnnotation(oid, annotation)

		assert.Equal(t, oid, op.ObjectOID)
		assert.Equal(t, "remove-annotation", op.Name)
		assert.WithinDuration(t, clock.Now(), op.Timestamp, time.Second)
		assert.Contains(t, op.Extras, "annotation")
		assert.IsType(t, Annotation{}, op.Extras["annotation"])
	})

	t.Run("CompleteReminder", func(t *testing.T) {
		oid := oid.OID("42d74d967d9b4e989502647ac510777ca1e22f4a")
		op := NewOperationCompleteReminder(oid)

		assert.Equal(t, oid, op.ObjectOID)
		assert.Equal(t, "complete-reminder", op.Name)
		assert.WithinDuration(t, clock.Now(), op.Timestamp, time.Second)
	})

	t.Run("ReviewFlashcard", func(t *testing.T) {
		oid := oid.OID("42d74d967d9b4e989502647ac510777ca1e22f4a")
		review := FlashcardReview{
			Feedback: "Good",
			Duration: 500 * time.Millisecond,
			DueAt:    clock.Now().Add(24 * time.Hour),
			Settings: map[string]any{
				"easeFactor": 2500,
			},
		}
		op := NewOperationReviewFlashcard(oid, review)

		assert.Equal(t, oid, op.ObjectOID)
		assert.Equal(t, "review-flashcard", op.Name)
		assert.WithinDuration(t, clock.Now(), op.Timestamp, time.Second)
		assert.Contains(t, op.Extras, "review")
		assert.IsType(t, FlashcardReview{}, op.Extras["review"])
	})
}

func TestNewPackFileFromOperations(t *testing.T) {
	NewTestRepository(t, WithFreezeOn("2024-01-01 00:00:00"))

	// Create a pack file with the different available operations
	packFile, err := NewPackFileFromOperations([]*Operation{
		NewOperationMarkNote("42d74d967d9b4e989502647ac510777ca1e22f4a"),
		NewOperationUnmarkNote("59235bca0a024d1ca46e0b5788e311c71d3e027e"),
		NewOperationAddAnnotation("40f5c217c3ca4a4f856638901b26a6025bb0b22b", Annotation{
			OID:  "f97ba6134cb447f88ae831ff745cf259ebe7d9ad",
			Text: "Use Markdown emphasis",
		}),
		NewOperationRemoveAnnotation("40f5c217c3ca4a4f856638901b26a6025bb0b22b", Annotation{
			OID: "f97ba6134cb447f88ae831ff745cf259ebe7d9ad",
		}),
		NewOperationReviewFlashcard("a0e9d4bc5cf64cb2b96a489b5a1ade36d0bd8d44", FlashcardReview{
			Feedback: "Good",
			Duration: 500 * time.Millisecond,
			DueAt:    clock.Now().Add(24 * time.Hour),
			Settings: map[string]any{
				"easeFactor": 2500,
			},
		}),
	})
	require.NoError(t, err)
	assert.FileExists(t, packFile.ObjectPath())

	// Add the pack file to the current index
	CurrentIndex().Add(packFile)

	// Reread the pack file to ensure it was written correctly
	packFile, err = CurrentIndex().ReadPackFile(packFile.OID)
	require.NoError(t, err)
	require.NotNil(t, packFile)
	require.Len(t, packFile.PackObjects, 5)

	// Check the first operation
	packOperation := packFile.PackObjects[0]
	assert.Equal(t, "operation", packOperation.Kind)
	assert.Equal(t, "2024-01-01T00:00:00Z", packOperation.CTime.Format(time.RFC3339))
	operation := packOperation.Read().(*Operation)
	assert.Equal(t, "mark-note", operation.Name)
	assert.Equal(t, oid.OID("42d74d967d9b4e989502647ac510777ca1e22f4a"), operation.ObjectOID)
	assert.WithinDuration(t, clock.Now(), operation.Timestamp, time.Second)
}

func TestOperationApply(t *testing.T) {
	tr := NewTestRepository(t, WithFreezeOn("2024-01-01 00:00:00"))

	// Insert the note
	tr.WriteFile("python.md", `# Python

## Flashcard: Python's creator

Who invented Python?

---

Guido van Rossum
`)

	_, err := CurrentRepository().Add(PathSpecs{"python.md"})
	require.NoError(t, err)
	err = CurrentRepository().Commit()
	require.NoError(t, err)

	// Check the note is present
	note := MustFindNoteByTitle(t, "Flashcard: Python's creator")
	assert.False(t, note.Marked)

	// Create a pack file with operations on this note
	packFile, err := NewPackFileFromOperations([]*Operation{
		NewOperationMarkNote(note.OID),
		NewOperationAddAnnotation(note.OID, Annotation{
			OID:  "42d74d967d9b4e989502647ac510777ca1e22f4a",
			Text: "Add reverse card",
		}),
	})
	require.NoError(t, err)
	assert.FileExists(t, packFile.ObjectPath())

	// Add the pack file in DB
	CurrentDB().UpsertPackFiles(packFile)

	// Reread the note
	note = MustFindNoteByTitle(t, "Flashcard: Python's creator")
	assert.True(t, note.Marked)
	assert.Equal(t, oid.MustParse("42d74d967d9b4e989502647ac510777ca1e22f4a"), note.Annotations[0].OID)

	// Edit the flashcard without changing the slug so that the existing note is updated
	tr.WriteFile("python.md", `# Python

## Flashcard: Python's creator

Who invented **Python**?

---

Guido van Rossum
`)
	note = MustFindNoteByTitle(t, "Flashcard: Python's creator")
	// Operation-relation fields must be preserved
	assert.True(t, note.Marked)
	assert.Equal(t, oid.MustParse("42d74d967d9b4e989502647ac510777ca1e22f4a"), note.Annotations[0].OID)

}

// Learning tests experimenting with package yaml.v3
func TestYAML(t *testing.T) {

	t.Run("Extra fields", func(t *testing.T) {

		var data = []byte(`
oid: 94a3f7eedf3d4963926d267c1e114bd14f65123a
created_at: 2024-01-01T00:00:00Z
extra_field: hello
another_field: 42
`)

		type MyStruct struct {
			OID   oid.OID        `yaml:"oid"`
			Extra map[string]any `yaml:",inline"`
		}

		var s MyStruct
		err := yaml.Unmarshal(data, &s)
		require.NoError(t, err)

		assert.Equal(t, oid.OID("94a3f7eedf3d4963926d267c1e114bd14f65123a"), s.OID)
		// Fields not present in the struct are stored in the Extra map
		assert.Equal(t, "hello", s.Extra["extra_field"])
		assert.Equal(t, 42, s.Extra["another_field"])

		// Write back to YAML to ensure extra fields are preserved
		data, err = yaml.Marshal(s)
		require.NoError(t, err)
		assert.Equal(t, "oid: 94a3f7eedf3d4963926d267c1e114bd14f65123a\nanother_field: 42\ncreated_at: 2024-01-01T00:00:00Z\nextra_field: hello\n", string(data))
	})
}

func TestOperationGraph(t *testing.T) {
	t.Run("NewOperationGraph", func(t *testing.T) {
		og := NewOperationGraph()
		assert.NotNil(t, og)
		assert.Empty(t, og.Entries)
	})

	t.Run("ReadWriteOperationGraph", func(t *testing.T) {
		_ = NewTestRepository(t, WithFreezeOn("2024-01-01 00:00:00"))
		
		// Create an operation graph with some entries
		og := &OperationGraph{
			Entries: []*OperationGraphEntry{
				{
					PackFileOID: oid.OID("42d74d967d9b4e989502647ac510777ca1e22f4a"),
					CTime:       clock.Now(),
				},
				{
					PackFileOID: oid.OID("f97ba6134cb447f88ae831ff745cf259ebe7d9ad"),
					CTime:       clock.Now().Add(-time.Hour),
				},
			},
		}

		// Test Save and Read
		err := og.Save()
		require.NoError(t, err)

		// Read it back
		readOG, err := ReadOperationGraph()
		require.NoError(t, err)
		require.Len(t, readOG.Entries, 2)
		assert.Equal(t, oid.OID("42d74d967d9b4e989502647ac510777ca1e22f4a"), readOG.Entries[0].PackFileOID)
		assert.Equal(t, oid.OID("f97ba6134cb447f88ae831ff745cf259ebe7d9ad"), readOG.Entries[1].PackFileOID)
	})

	t.Run("OperationGraphDiff", func(t *testing.T) {
		// Create local operation graph
		local := &OperationGraph{
			Entries: []*OperationGraphEntry{
				{
					PackFileOID: oid.OID("42d74d967d9b4e989502647ac510777ca1e22f4a"),
					CTime:       clock.Now(),
				},
			},
		}

		// Create remote operation graph with additional entry
		remote := &OperationGraph{
			Entries: []*OperationGraphEntry{
				{
					PackFileOID: oid.OID("42d74d967d9b4e989502647ac510777ca1e22f4a"),
					CTime:       clock.Now(),
				},
				{
					PackFileOID: oid.OID("f97ba6134cb447f88ae831ff745cf259ebe7d9ad"),
					CTime:       clock.Now().Add(-time.Hour),
				},
			},
		}

		// Test diff: what's missing in local from remote
		diff := local.Diff(remote)
		require.Len(t, diff.MissingPackFiles, 1)
		assert.Equal(t, oid.OID("f97ba6134cb447f88ae831ff745cf259ebe7d9ad"), diff.MissingPackFiles[0].OID)

		// Test reverse diff: what's missing in remote from local (should be empty)
		diffReverse := remote.Diff(local)
		assert.Empty(t, diffReverse.MissingPackFiles)
		assert.Empty(t, diffReverse.MissingBlobs)
	})
}
