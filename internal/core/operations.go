package core

import (
	"fmt"
	"io"
	"time"

	"github.com/julien-sobczak/the-notewriter/pkg/clock"
	"github.com/julien-sobczak/the-notewriter/pkg/oid"
	"gopkg.in/yaml.v3"
)

// Operation represents a change to a StatefulObject.
//
// Operations use the CRDT (Conflict-free Replicated Data Type) model to ensure that
// changes can be applied in any order and still result in the same final state.
// Each operation has a timestamp that is used to resolve conflicts (LWW = last write wins).
//
// See https://en.wikipedia.org/wiki/Conflict-free_replicated_data_type
type Operation struct {
	OID       oid.OID        `json:"oid" yaml:"oid"`
	ObjectOID oid.OID        `json:"object_oid" yaml:"object_oid"` // OID of the object this operation applies to
	Name      string         `json:"name" yaml:"name"`
	Timestamp time.Time      `json:"timestamp" yaml:"timestamp"`
	Extras    map[string]any `yaml:",inline"` // Capture extra fields
}

// NewOperationMarkNote creates a new operation to mark a note.
func NewOperationMarkNote(noteOID oid.OID) *Operation {
	return &Operation{
		OID:       oid.New(),
		ObjectOID: noteOID,
		Name:      "mark-note",
		Timestamp: clock.Now(),
	}
}

// NewOperationUnmarkNote creates a new operation to mark a note.
func NewOperationUnmarkNote(noteOID oid.OID) *Operation {
	return &Operation{
		OID:       oid.New(),
		ObjectOID: noteOID,
		Name:      "unmark-note",
		Timestamp: clock.Now(),
	}
}

// NewOperationAddAnnotation creates a new operation to add an annotation to a note.
func NewOperationAddAnnotation(noteOID oid.OID, annotation Annotation) *Operation {
	return &Operation{
		OID:       oid.New(),
		ObjectOID: noteOID,
		Name:      "add-annotation",
		Timestamp: clock.Now(),
		Extras:    map[string]any{"annotation": annotation},
	}
}

// NewOperationRemoveAnnotation creates a new operation to remove an annotation from a note.
func NewOperationRemoveAnnotation(noteOID oid.OID, annotation Annotation) *Operation {
	return &Operation{
		OID:       oid.New(),
		ObjectOID: noteOID,
		Name:      "remove-annotation",
		Timestamp: clock.Now(),
		Extras:    map[string]any{"annotation": annotation},
	}
}

// NewOperationCompleteReminder creates a new operation to complete a reminder.
func NewOperationCompleteReminder(reminderOID oid.OID) *Operation {
	return &Operation{
		OID:       oid.New(),
		ObjectOID: reminderOID,
		Name:      "complete-reminder",
		Timestamp: clock.Now(),
	}
}

// NewOperationReviewFlashcard creates a new operation to review a flashcard.
func NewOperationReviewFlashcard(flashcardOID oid.OID, review FlashcardReview) *Operation {
	return &Operation{
		OID:       oid.New(),
		ObjectOID: flashcardOID,
		Name:      "review-flashcard",
		Timestamp: clock.Now(),
		Extras:    map[string]any{"review": review},
	}
}

/* Operations Implementation */

type OperationFn func(obj StatefulObject, operation *Operation) (StatefulObject, error)

var operationFns = map[string]OperationFn{
	"mark-note":         MarkNote,
	"unmark-note":       UnmarkNote,
	"add-annotation":    AddAnnotation,
	"remove-annotation": RemoveAnnotation,
	"complete-reminder": CompleteReminder,
	"review-flashcard":  ReviewFlashcard,
	// Add more operations here
}

/* Packable interface */

func (o *Operation) Kind() string {
	return "operation"
}
func (o *Operation) UniqueOID() oid.OID {
	return o.OID
}
func (o *Operation) Read(r io.Reader) error {
	return yaml.NewDecoder(r).Decode(o)
}
func (o *Operation) Write(w io.Writer) error {
	data, err := yaml.Marshal(o)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (o *Operation) ModificationTime() time.Time {
	return o.Timestamp
}

func (o *Operation) String() string {
	return fmt.Sprintf("Operation %s on object %q", o.Name, o.OID)
}

/* Operation functions */

// parseExtras parses the extras field of the operation into the given expected type.
func (o *Operation) parseExtras(fieldName string, v any) error {
	value, ok := o.Extras[fieldName]
	if !ok {
		return fmt.Errorf("missing field %q in extras", fieldName)
	}
	b, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(b, v); err != nil {
		return err
	}
	return nil
}

// Apply applies the operation to the given object.
func (o *Operation) Apply(obj StatefulObject) (StatefulObject, error) {
	fn, ok := operationFns[o.Name]
	if !ok {
		return nil, fmt.Errorf("operation %q not found", o.Name)
	}
	return fn(obj, o)
}

/*
 * Mark/Unmark a note
 */

func MarkNote(obj StatefulObject, operation *Operation) (StatefulObject, error) {
	note, ok := obj.(*Note)
	if !ok {
		return nil, fmt.Errorf("object %q is not a note", obj.UniqueOID())
	}
	note.Mark(operation.Timestamp)
	return note, nil
}

func UnmarkNote(obj StatefulObject, operation *Operation) (StatefulObject, error) {
	note, ok := obj.(*Note)
	if !ok {
		return nil, fmt.Errorf("object %q is not a note", obj.UniqueOID())
	}
	note.Unmark(operation.Timestamp)
	return note, nil
}

/*
 * Review a flashcard
 */

func ReviewFlashcard(obj StatefulObject, operation *Operation) (StatefulObject, error) {
	flashcard, ok := obj.(*Flashcard)
	if !ok {
		return nil, fmt.Errorf("object %q is not a flashcard", obj.UniqueOID())
	}
	var review FlashcardReview
	if err := operation.parseExtras("review", &review); err != nil {
		return nil, fmt.Errorf("unable to parse extras for operation %v: %w", operation.OID, err)
	}
	flashcard.Review(operation.Timestamp, &review)
	return flashcard, nil
}

/*
 * Add/Delete an annotation on a note
 */

func AddAnnotation(obj StatefulObject, operation *Operation) (StatefulObject, error) {
	note, ok := obj.(*Note)
	if !ok {
		return nil, fmt.Errorf("object %q is not a note", obj.UniqueOID())
	}
	var annotation Annotation
	if err := operation.parseExtras("annotation", &annotation); err != nil {
		return nil, fmt.Errorf("unable to parse extras for operation %v: %w", operation.OID, err)
	}
	note.AddAnnotation(operation.Timestamp, annotation)
	return note, nil

}

func RemoveAnnotation(obj StatefulObject, operation *Operation) (StatefulObject, error) {
	note, ok := obj.(*Note)
	if !ok {
		return nil, fmt.Errorf("object %q is not a note", obj.UniqueOID())
	}
	var annotation Annotation
	if err := operation.parseExtras("annotation", &annotation); err != nil {
		return nil, fmt.Errorf("unable to parse extras for operation %v: %w", operation.OID, err)
	}
	note.RemoveAnnotation(operation.Timestamp, annotation)
	return note, nil
}

/*
 * Completed a reminder
 */

func CompleteReminder(obj StatefulObject, operation *Operation) (StatefulObject, error) {
	reminder, ok := obj.(*Reminder)
	if !ok {
		return nil, fmt.Errorf("object %q is not a reminder", obj.UniqueOID())
	}
	reminder.Complete(operation.Timestamp)
	return reminder, nil
}

/* Pack File Management */

// NewPackFileFromOperations creates a new pack file from the given operations.
func NewPackFileFromOperations(operations []*Operation) (*PackFile, error) {
	packFile := &PackFile{
		OID: oid.New(),
		// Init pack file properties
		CTime: clock.Now(),
		Kind:  "operations",
	}

	// Create objects
	for _, operation := range operations {
		if err := packFile.AppendPackable(operation); err != nil {
			return nil, err
		}
	}

	// Save the pack file on disk
	if err := packFile.Save(); err != nil {
		return nil, err
	}

	return packFile, nil
}
