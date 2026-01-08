package course

import (
	"context"

	"github.com/google/uuid"
	"github.com/juju/errors"
	"github.com/tilinna/clock"

	"github.com/XaviFP/toshokan/common/pagination"
	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
)

type Enroller interface {
	Enroll(ctx context.Context, userID, courseID uuid.UUID) (UserCourseProgress, error)
}

type enroller struct {
	clock       clock.Clock
	repo        Repository
	decksClient pbDeck.DecksAPIClient
}

func NewEnroller(clock clock.Clock, repo Repository, decksClient pbDeck.DecksAPIClient) Enroller {
	return &enroller{
		clock:       clock,
		repo:        repo,
		decksClient: decksClient,
	}
}

func (e *enroller) Enroll(ctx context.Context, userID, courseID uuid.UUID) (UserCourseProgress, error) {
	// Initialize state with all lessons and decks for the course
	state := NewProgressState()

	// Get all lessons for the course
	// TODO: Refactor into Iterator to handle large number of lessons
	lessons, err := e.repo.GetLessonsByCourseID(ctx, courseID, pagination.NewOlderFirstPagination(pagination.WithFirst(1000)))
	if err != nil {
		return UserCourseProgress{}, errors.Trace(err)
	}

	if len(lessons.Edges) == 0 {
		return UserCourseProgress{}, errors.New("courses: cannot enroll in a course with no lessons")
	}

	firstLessonID := lessons.Edges[0].Lesson.ID

	// Initialize progress for each lesson and its decks
	for _, edge := range lessons.Edges {
		lesson := edge.Lesson

		lessonKey := lesson.ID.String()
		state.Lessons[lessonKey] = &LessonProgress{
			Decks:       make(map[string]*DeckProgress),
			IsCompleted: false,
		}

		// Get decks for this lesson
		parsedDeckIDs := ParseDeckReferences(lesson.Body)

		if len(parsedDeckIDs) == 0 {
			return UserCourseProgress{}, errors.New("courses: cannot enroll in a course with lessons that have no decks")
		}

		for _, dID := range parsedDeckIDs {
			res, err := e.decksClient.GetDeck(ctx, &pbDeck.GetDeckRequest{
				DeckId: dID.String(),
			})
			if err != nil {
				return UserCourseProgress{}, errors.Trace(err)
			}

			state.Lessons[lessonKey].Decks[res.Deck.Id] = &DeckProgress{
				Cards:       make(map[string]*CardProgress),
				IsCompleted: false,
			}

			for _, card := range res.Deck.Cards {
				state.Lessons[lessonKey].Decks[res.Deck.Id].Cards[card.Id] = &CardProgress{
					IsCompleted: false,
				}
			}

		}
	}

	state.CurrentLessonID = firstLessonID

	err = e.repo.EnrollUserInCourse(ctx, userID, courseID, *state)
	if err != nil {
		return UserCourseProgress{}, errors.Trace(err)
	}

	return UserCourseProgress{
		CourseID:        courseID,
		UserID:          userID,
		State:           state,
		CurrentLessonID: firstLessonID,
		CreatedAt:       e.clock.Now().UTC(),
		UpdatedAt:       e.clock.Now().UTC(),
	}, nil
}
