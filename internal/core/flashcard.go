package core

import "time"

type CardType int

const (
	CardNew        CardType = 0
	CardLearning   CardType = 1
	CardReview     CardType = 2
	CardRelearning CardType = 3
)

type QueueType int

const (
	QueueSuspend  QueueType = -1 // leeches as manual suspension is not supported
	QueueNew      QueueType = 0  // new (never shown)
	QueueLearn    QueueType = 1  // learning/relearning
	QueueReview   QueueType = 2  // review (as for type)
	QueueDayLearn QueueType = 3  // in learning, next review in at least a day after the previous review
)

type Flashcard struct {
	ID int64

	// Note representing the flashcard
	NoteID int64

	// List of tags
	Tags []string

	// 0=new, 1=learning, 2=review, 3=relearning
	Type CardType

	// Queue types
	Queue QueueType

	// Due is used differently for different card types:
	//   - new: note id or random int
	//   - due: integer day, relative to the collection's creation time
	//   - learning: integer timestamp in second
	Due int

	// The interval. Negative = seconds, positive = days
	Interval int

	// The ease factor in permille (ex: 2500 = the interval will be multiplied by 2.5 the next time you press "Good").
	EaseFactor int

	// Number of reviews
	Repetitions int

	// The number of times the card went from a "was answered correctly" to "was answered incorrectly" state.
	Lapses int

	// Of the form a*1000+b, with:
	//   - a the number of reps left today
	//   - b the number of reps left till graduation
	// For example: '2004' means 2 reps left today and 4 reps till graduation
	Left int

	// Fields in Markdown (best for editing)
	FrontMarkdown string
	BackMarkdown  string
	// Fields in HTML (best for rendering)
	FrontHTML string
	BackHTML  string
	// Fields in raw text (best for indexing)
	FrontText string
	BackText  string

	// Timestamps to track changes
	CreatedAt *time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time
}
