package course

import (
	"context"

	"github.com/google/uuid"
	"github.com/juju/errors"

	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
)

type CardAnswer struct {
	CardID   uuid.UUID
	AnswerID uuid.UUID
}

type Answerer interface {
	Answer(ctx context.Context, userID, courseID, lessonID, deckID uuid.UUID, cardAnswers []CardAnswer) error
}

type answerer struct {
	repo       Repository
	deckClient pbDeck.DecksAPIClient
}

func NewAnswerer(repo Repository, deckClient pbDeck.DecksAPIClient) Answerer {
	return &answerer{
		repo:       repo,
		deckClient: deckClient,
	}
}

// Answer processes the user's answers for a set of cards in a deck within a lesson.
func (a *answerer) Answer(ctx context.Context, userID, courseID, lessonID, deckID uuid.UUID, cardAnswers []CardAnswer) error {
	userProgress, err := a.repo.GetUserCourseProgress(ctx, userID, courseID)
	if err != nil {
		return err
	}

	if userProgress.State == nil {
		return errors.Trace(ErrUnitProgressStateNotInitialized)
	}

	cardIDs := make([]string, 0, len(cardAnswers))
	for _, ca := range cardAnswers {
		cardIDs = append(cardIDs, ca.CardID.String())
	}

	// TODO: Deck service should accept []CardAnswer and return which are correct
	getCardsRes, err := a.deckClient.GetCards(ctx, &pbDeck.GetCardsRequest{
		CardIds: cardIDs,
	})
	if err != nil {
		return errors.Trace(err)
	}

	correctAnswers := make(map[string]struct{}, len(getCardsRes.Cards))
	for _, card := range getCardsRes.Cards {
		for _, answer := range card.PossibleAnswers {
			if answer.IsCorrect {
				correctAnswers[answer.Id] = struct{}{}
			}
		}
	}

	for _, cardAnswer := range cardAnswers {
		_, correct := correctAnswers[cardAnswer.AnswerID.String()]
		if err := userProgress.State.AnswerCard(lessonID.String(), deckID.String(), cardAnswer.CardID.String(), correct); err != nil {
			return errors.Trace(err)
		}
	}

	// If this lesson is now completed, mark current lesson as this one so clients see is_current=true
	// This is handled by the StateSyncer as well, but doing it here makes the change visible 'immediately' (upon next request)
	if userProgress.State.IsLessonCompleted(lessonID.String()) {
		userProgress.CurrentLessonID = lessonID
	}

	if err := a.repo.UpdateUserProgress(ctx, userProgress); err != nil {
		return errors.Trace(err)
	}

	return nil
}
