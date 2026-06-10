package course

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestComputeStudentSummary(t *testing.T) {
	t.Run("skips_invalid_lesson_ids", func(t *testing.T) {
		validLessonID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
		courseID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

		progress := UserCourseProgress{
			ID:              uuid.New(),
			UserID:          userID,
			CourseID:        courseID,
			CurrentLessonID: validLessonID,
			CreatedAt:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:       time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		}

		state := &ProgressState{
			Lessons: map[string]*LessonProgress{
				// Valid UUID key
				validLessonID.String(): {
					IsCompleted: true,
					Decks: map[string]*DeckProgress{
						"deck1": {
							Cards: map[string]*CardProgress{
								"card1": {CorrectAnswers: 5, IncorrectAnswers: 1},
							},
						},
					},
				},
				// Invalid UUID key - should be skipped
				"not-a-uuid": {
					IsCompleted: true,
					Decks: map[string]*DeckProgress{
						"deck2": {
							Cards: map[string]*CardProgress{
								"card2": {CorrectAnswers: 10, IncorrectAnswers: 2},
							},
						},
					},
				},
				// Another invalid key
				"corrupted": {
					IsCompleted: true,
					Decks:       map[string]*DeckProgress{},
				},
			},
		}

		summary := computeStudentSummary(progress, state, 5)

		// Invalid lesson IDs are completely skipped (LessonsCompleted only counts valid)
		assert.Equal(t, 1, summary.LessonsCompleted)

		// LessonDetails should only contain valid UUIDs
		assert.Len(t, summary.LessonDetails, 1)
		assert.Equal(t, validLessonID, summary.LessonDetails[0].LessonID)

		// Answers from invalid lessons are NOT counted (continue skips everything)
		assert.Equal(t, 5, summary.TotalCorrectAnswers)
		assert.Equal(t, 1, summary.TotalIncorrectAnswers)
	})

	t.Run("handles_nil_state", func(t *testing.T) {
		progress := UserCourseProgress{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			CourseID:  uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		summary := computeStudentSummary(progress, nil, 3)

		assert.Equal(t, 0, summary.LessonsCompleted)
		assert.Equal(t, 3, summary.TotalLessons)
		assert.Empty(t, summary.LessonDetails)
	})

	t.Run("handles_nil_lessons_map", func(t *testing.T) {
		progress := UserCourseProgress{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			CourseID:  uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		state := &ProgressState{Lessons: nil}
		summary := computeStudentSummary(progress, state, 3)

		assert.Equal(t, 0, summary.LessonsCompleted)
		assert.Empty(t, summary.LessonDetails)
	})
}
