package course

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
)

func TestAnswerer_Answer_Success_AllCorrect(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lessonID := uuid.New()
	deckID := uuid.New()
	card1ID := uuid.New()
	card2ID := uuid.New()
	answer1ID := uuid.New()
	answer2ID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	state := NewProgressState()
	state.Lessons[lessonID.String()] = &LessonProgress{
		IsCompleted: false,
		Decks: map[string]*DeckProgress{
			deckID.String(): {
				IsCompleted: false,
				Cards: map[string]*CardProgress{
					card1ID.String(): {IsCompleted: false},
					card2ID.String(): {IsCompleted: false},
				},
			},
		},
	}

	userProgress := UserCourseProgress{
		State: state,
	}

	cardAnswers := []CardAnswer{
		{CardID: card1ID, AnswerID: answer1ID},
		{CardID: card2ID, AnswerID: answer2ID},
	}

	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)
	mockRepo.On("UpdateUserProgress", ctx, mock.MatchedBy(func(ucp UserCourseProgress) bool {
		return ucp.State != nil
	})).Return(nil)

	mockDecksClient.On("GetCards", ctx, mock.MatchedBy(func(req *pbDeck.GetCardsRequest) bool {
		return len(req.CardIds) == 2
	})).Return(&pbDeck.GetCardsResponse{
		Cards: map[string]*pbDeck.Card{
			card1ID.String(): {
				Id:    card1ID.String(),
				Title: "Question 1",
				PossibleAnswers: []*pbDeck.Answer{
					{Id: answer1ID.String(), IsCorrect: true},
					{Id: uuid.New().String(), IsCorrect: false},
				},
			},
			card2ID.String(): {
				Id:    card2ID.String(),
				Title: "Question 2",
				PossibleAnswers: []*pbDeck.Answer{
					{Id: answer2ID.String(), IsCorrect: true},
					{Id: uuid.New().String(), IsCorrect: false},
				},
			},
		},
	}, nil)

	answerer := NewAnswerer(mockRepo, mockDecksClient)

	err := answerer.Answer(ctx, userID, courseID, lessonID, deckID, cardAnswers)
	require.NoError(t, err)

	// Verify cards were marked as answered correctly
	assert.True(t, userProgress.State.Lessons[lessonID.String()].Decks[deckID.String()].Cards[card1ID.String()].IsCompleted)
	assert.True(t, userProgress.State.Lessons[lessonID.String()].Decks[deckID.String()].Cards[card2ID.String()].IsCompleted)
	assert.Equal(t, 1, userProgress.State.Lessons[lessonID.String()].Decks[deckID.String()].Cards[card1ID.String()].CorrectAnswers)
	assert.Equal(t, 1, userProgress.State.Lessons[lessonID.String()].Decks[deckID.String()].Cards[card2ID.String()].CorrectAnswers)

	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)

}

func TestAnswerer_Answer_Success_SomeIncorrect(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lessonID := uuid.New()
	deckID := uuid.New()
	card1ID := uuid.New()
	card2ID := uuid.New()
	correctAnswerID := uuid.New()
	incorrectAnswerID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	state := NewProgressState()
	state.Lessons[lessonID.String()] = &LessonProgress{
		IsCompleted: false,
		Decks: map[string]*DeckProgress{
			deckID.String(): {
				IsCompleted: false,
				Cards: map[string]*CardProgress{
					card1ID.String(): {IsCompleted: false},
					card2ID.String(): {IsCompleted: false},
				},
			},
		},
	}

	userProgress := UserCourseProgress{
		State: state,
	}

	cardAnswers := []CardAnswer{
		{CardID: card1ID, AnswerID: correctAnswerID},
		{CardID: card2ID, AnswerID: incorrectAnswerID},
	}

	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)
	mockRepo.On("UpdateUserProgress", ctx, mock.Anything).Return(nil)

	mockDecksClient.On("GetCards", ctx, mock.MatchedBy(func(req *pbDeck.GetCardsRequest) bool {
		return len(req.CardIds) == 2
	})).Return(&pbDeck.GetCardsResponse{
		Cards: map[string]*pbDeck.Card{
			card1ID.String(): {
				Id:    card1ID.String(),
				Title: "Question 1",
				PossibleAnswers: []*pbDeck.Answer{
					{Id: correctAnswerID.String(), IsCorrect: true},
					{Id: uuid.New().String(), IsCorrect: false},
				},
			},
			card2ID.String(): {
				Id:    card2ID.String(),
				Title: "Question 2",
				PossibleAnswers: []*pbDeck.Answer{
					{Id: uuid.New().String(), IsCorrect: true},
					{Id: incorrectAnswerID.String(), IsCorrect: false},
				},
			},
		},
	}, nil)

	answerer := NewAnswerer(mockRepo, mockDecksClient)

	err := answerer.Answer(ctx, userID, courseID, lessonID, deckID, cardAnswers)
	require.NoError(t, err)

	// Verify card1 was correct and card2 was incorrect
	assert.True(t, userProgress.State.Lessons[lessonID.String()].Decks[deckID.String()].Cards[card1ID.String()].IsCompleted)
	assert.False(t, userProgress.State.Lessons[lessonID.String()].Decks[deckID.String()].Cards[card2ID.String()].IsCompleted)
	assert.Equal(t, 1, userProgress.State.Lessons[lessonID.String()].Decks[deckID.String()].Cards[card1ID.String()].CorrectAnswers)
	assert.Equal(t, 1, userProgress.State.Lessons[lessonID.String()].Decks[deckID.String()].Cards[card2ID.String()].IncorrectAnswers)

	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)

}

func TestAnswerer_Answer_ErrorGettingUserProgress(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lessonID := uuid.New()
	deckID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(UserCourseProgress{}, assert.AnError)

	answerer := NewAnswerer(mockRepo, mockDecksClient)

	err := answerer.Answer(ctx, userID, courseID, lessonID, deckID, []CardAnswer{})

	require.Error(t, err)
	assert.Equal(t, assert.AnError, err)
	mockRepo.AssertExpectations(t)
}

func TestAnswerer_Answer_NilProgressState(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lessonID := uuid.New()
	deckID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	userProgress := UserCourseProgress{
		State: nil,
	}

	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)

	answerer := NewAnswerer(mockRepo, mockDecksClient)

	err := answerer.Answer(ctx, userID, courseID, lessonID, deckID, []CardAnswer{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "progress state not initialized")
	mockRepo.AssertExpectations(t)
}

func TestAnswerer_Answer_ErrorGettingCards(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lessonID := uuid.New()
	deckID := uuid.New()
	cardID := uuid.New()
	answerID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	state := NewProgressState()
	state.Lessons[lessonID.String()] = &LessonProgress{
		IsCompleted: false,
		Decks: map[string]*DeckProgress{
			deckID.String(): {
				IsCompleted: false,
				Cards: map[string]*CardProgress{
					cardID.String(): {IsCompleted: false},
				},
			},
		},
	}

	userProgress := UserCourseProgress{
		State: state,
	}

	cardAnswers := []CardAnswer{
		{CardID: cardID, AnswerID: answerID},
	}

	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)

	mockDecksClient.On("GetCards", ctx, mock.Anything).Return(&pbDeck.GetCardsResponse{}, assert.AnError)

	answerer := NewAnswerer(mockRepo, mockDecksClient)

	err := answerer.Answer(ctx, userID, courseID, lessonID, deckID, cardAnswers)

	require.Error(t, err)
	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)
}

func TestAnswerer_Answer_ErrorAnsweringCard(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lessonID := uuid.New()
	deckID := uuid.New()
	cardID := uuid.New()
	answerID := uuid.New()
	nonExistentCardID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	state := NewProgressState()
	state.Lessons[lessonID.String()] = &LessonProgress{
		IsCompleted: false,
		Decks: map[string]*DeckProgress{
			deckID.String(): {
				IsCompleted: false,
				Cards: map[string]*CardProgress{
					cardID.String(): {IsCompleted: false},
				},
			},
		},
	}

	userProgress := UserCourseProgress{
		State: state,
	}

	// Try to answer a card that doesn't exist in the user's progress
	cardAnswers := []CardAnswer{
		{CardID: nonExistentCardID, AnswerID: answerID},
	}

	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)

	mockDecksClient.On("GetCards", ctx, mock.Anything).Return(&pbDeck.GetCardsResponse{
		Cards: map[string]*pbDeck.Card{
			nonExistentCardID.String(): {
				Id:    nonExistentCardID.String(),
				Title: "Question",
				PossibleAnswers: []*pbDeck.Answer{
					{Id: answerID.String(), IsCorrect: true},
				},
			},
		},
	}, nil)

	answerer := NewAnswerer(mockRepo, mockDecksClient)

	err := answerer.Answer(ctx, userID, courseID, lessonID, deckID, cardAnswers)

	require.Error(t, err)
	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)
}

func TestAnswerer_Answer_ErrorUpdatingUserProgress(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lessonID := uuid.New()
	deckID := uuid.New()
	cardID := uuid.New()
	answerID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	state := NewProgressState()
	state.Lessons[lessonID.String()] = &LessonProgress{
		IsCompleted: false,
		Decks: map[string]*DeckProgress{
			deckID.String(): {
				IsCompleted: false,
				Cards: map[string]*CardProgress{
					cardID.String(): {IsCompleted: false},
				},
			},
		},
	}

	userProgress := UserCourseProgress{
		State: state,
	}

	cardAnswers := []CardAnswer{
		{CardID: cardID, AnswerID: answerID},
	}

	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)

	mockRepo.On("UpdateUserProgress", ctx, mock.Anything).Return(assert.AnError)

	mockDecksClient.On("GetCards", ctx, mock.Anything).Return(&pbDeck.GetCardsResponse{
		Cards: map[string]*pbDeck.Card{
			cardID.String(): {
				Id:    cardID.String(),
				Title: "Question",
				PossibleAnswers: []*pbDeck.Answer{
					{Id: answerID.String(), IsCorrect: true},
				},
			},
		},
	}, nil)

	answerer := NewAnswerer(mockRepo, mockDecksClient)

	err := answerer.Answer(ctx, userID, courseID, lessonID, deckID, cardAnswers)

	require.Error(t, err)
	assert.Contains(t, err.Error(), assert.AnError.Error())
	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)
}

func TestAnswerer_Answer_EmptyCardAnswers(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lessonID := uuid.New()
	deckID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	state := NewProgressState()
	state.Lessons[lessonID.String()] = &LessonProgress{
		IsCompleted: false,
		Decks: map[string]*DeckProgress{
			deckID.String(): {
				IsCompleted: false,
				Cards:       make(map[string]*CardProgress),
			},
		},
	}

	userProgress := UserCourseProgress{
		State: state,
	}

	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)
	mockRepo.On("UpdateUserProgress", ctx, mock.Anything).Return(nil)

	mockDecksClient.On("GetCards", ctx, mock.MatchedBy(func(req *pbDeck.GetCardsRequest) bool {
		return len(req.CardIds) == 0
	})).Return(&pbDeck.GetCardsResponse{
		Cards: map[string]*pbDeck.Card{},
	}, nil)

	answerer := NewAnswerer(mockRepo, mockDecksClient)

	err := answerer.Answer(ctx, userID, courseID, lessonID, deckID, []CardAnswer{})
	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)

}

func TestAnswerer_Answer_MultipleAnswersForSameCard(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lessonID := uuid.New()
	deckID := uuid.New()
	cardID := uuid.New()
	correctAnswerID := uuid.New()
	incorrectAnswerID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	state := NewProgressState()
	state.Lessons[lessonID.String()] = &LessonProgress{
		IsCompleted: false,
		Decks: map[string]*DeckProgress{
			deckID.String(): {
				IsCompleted: false,
				Cards: map[string]*CardProgress{
					cardID.String(): {IsCompleted: false},
				},
			},
		},
	}

	userProgress := UserCourseProgress{
		State: state,
	}

	// Submit first an incorrect answer, then a correct answer for the same card
	cardAnswers := []CardAnswer{
		{CardID: cardID, AnswerID: incorrectAnswerID},
		{CardID: cardID, AnswerID: correctAnswerID},
	}

	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)
	mockRepo.On("UpdateUserProgress", ctx, mock.Anything).Return(nil)

	mockDecksClient.On("GetCards", ctx, mock.MatchedBy(func(req *pbDeck.GetCardsRequest) bool {
		// Should have both card IDs (even though they're the same)
		return len(req.CardIds) == 2
	})).Return(&pbDeck.GetCardsResponse{
		Cards: map[string]*pbDeck.Card{
			cardID.String(): {
				Id:    cardID.String(),
				Title: "Question",
				PossibleAnswers: []*pbDeck.Answer{
					{Id: correctAnswerID.String(), IsCorrect: true},
					{Id: incorrectAnswerID.String(), IsCorrect: false},
				},
			},
		},
	}, nil)

	answerer := NewAnswerer(mockRepo, mockDecksClient)

	err := answerer.Answer(ctx, userID, courseID, lessonID, deckID, cardAnswers)
	require.NoError(t, err)

	// Card should be completed after the correct answer
	assert.True(t, userProgress.State.Lessons[lessonID.String()].Decks[deckID.String()].Cards[cardID.String()].IsCompleted)
	assert.Equal(t, 1, userProgress.State.Lessons[lessonID.String()].Decks[deckID.String()].Cards[cardID.String()].CorrectAnswers)
	assert.Equal(t, 1, userProgress.State.Lessons[lessonID.String()].Decks[deckID.String()].Cards[cardID.String()].IncorrectAnswers)

	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)

}
