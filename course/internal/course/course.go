package course

import (
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/juju/errors"
)

var (
	ErrCourseNotFound       = errors.New("courses: course not found")
	ErrLessonNotFound       = errors.New("courses: lesson not found")
	ErrUserProgressNotFound = errors.New("courses: user progress not found")
	ErrInvalidCourse        = errors.New("courses: invalid course")
	ErrInvalidLesson        = errors.New("courses: invalid lesson")
	ErrUserAlreadyEnrolled  = errors.New("courses: user already enrolled in course")
	ErrNoTitle              = errors.New("courses: title is missing")
	ErrNoDescription        = errors.New("courses: description is missing")
	ErrNoBody               = errors.New("courses: body is required")
	ErrNoDecksReferenced    = errors.New("courses: lesson must reference at least one deck")
)

var (
	deckReferencePattern = `!\[deck\]\(([a-f0-9\-]{36})\)` // matches ![deck](uuid-format)
	deckReferenceRE      = regexp.MustCompile(deckReferencePattern)
)

type Course struct {
	ID          uuid.UUID  `json:"id"`
	Order       int64      `json:"order"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type Lesson struct {
	ID          uuid.UUID  `json:"id"`
	CourseID    uuid.UUID  `json:"course_id"`
	Order       int        `json:"order"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Body        string     `json:"body"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// UserCourseProgress represents a user's progress in a course
type UserCourseProgress struct {
	ID              uuid.UUID      `json:"id"`
	UserID          uuid.UUID      `json:"user_id"`
	CourseID        uuid.UUID      `json:"course_id"`
	CurrentLessonID uuid.UUID      `json:"current_lesson_id,omitempty"`
	State           *ProgressState `json:"state"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// ParseDeckReferences extracts deck UUIDs from markdown lesson body
// Looks for pattern: ![deck](uuid)
func ParseDeckReferences(body string) []uuid.UUID {
	matches := deckReferenceRE.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return []uuid.UUID{}
	}

	var deckIDs []uuid.UUID
	seen := make(map[uuid.UUID]bool)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		id, err := uuid.Parse(match[1])
		if err != nil {
			// Skip invalid UUIDs
			continue
		}

		// Avoid duplicates
		if !seen[id] {
			deckIDs = append(deckIDs, id)
			seen[id] = true
		}
	}

	return deckIDs
}

func ParseDeckReferencesAsStrings(body string) []string {
	deckIDs := ParseDeckReferences(body)
	strIDs := make([]string, len(deckIDs))
	for i, id := range deckIDs {
		strIDs[i] = id.String()
	}
	return strIDs
}
