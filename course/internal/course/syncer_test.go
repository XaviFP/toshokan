package course

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/XaviFP/toshokan/common/pagination"
	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
)

func TestSync_InitializesMissingLesson(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lessonID := uuid.New()
	deckID := uuid.New()
	cardID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	state := NewProgressState()
	userProgress := UserCourseProgress{
		State: state,
	}

	lessonCursor, _ := pagination.ToCursor(LessonCursor{Order: 1})
	lessons := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{
					ID:       lessonID,
					CourseID: courseID,
					Order:    1,
					Title:    "Lesson 1",
					Body:     "Content with ![deck](" + deckID.String() + ")",
				},
				Cursor: lessonCursor,
			},
		},
	}

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}, false).Return(lessons, nil)
	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)
	mockRepo.On("UpdateUserProgress", ctx, mock.MatchedBy(func(ucp UserCourseProgress) bool {
		if ucp.State == nil || ucp.CurrentLessonID != lessonID {
			return false
		}
		if ucp.State.Lessons[lessonID.String()] == nil {
			return false
		}
		if ucp.State.Lessons[lessonID.String()].Decks[deckID.String()] == nil {
			return false
		}
		if ucp.State.Lessons[lessonID.String()].Decks[deckID.String()].Cards[cardID.String()] == nil {
			return false
		}
		return true
	})).Return(nil)

	mockDecksClient.On("GetDeck", ctx, mock.MatchedBy(func(req *pbDeck.GetDeckRequest) bool {
		return req.DeckId == deckID.String()
	})).Return(&pbDeck.GetDeckResponse{
		Deck: &pbDeck.Deck{
			Id:    deckID.String(),
			Title: "Test Deck",
			Cards: []*pbDeck.Card{{Id: cardID.String(), Title: "Q"}},
		},
	}, nil)

	syncer := NewStateSyncer(mockRepo, mockDecksClient)

	err := syncer.Sync(ctx, userID, courseID)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)

	assert.Equal(t, lessonID, userProgress.State.CurrentLessonID)
	assert.NotNil(t, userProgress.State.Lessons[lessonID.String()])
	assert.NotNil(t, userProgress.State.Lessons[lessonID.String()].Decks[deckID.String()])
	assert.NotNil(t, userProgress.State.Lessons[lessonID.String()].Decks[deckID.String()].Cards[cardID.String()])
}

func TestSync_AddsNewDecksToExistingLesson(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lessonID := uuid.New()
	oldDeckID := uuid.New()
	newDeckID := uuid.New()
	newCardID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	// The lesson was completed before the new deck was added to it.
	state := NewProgressState()
	state.Lessons[lessonID.String()] = &LessonProgress{
		IsCompleted: true,
		Decks: map[string]*DeckProgress{
			oldDeckID.String(): {
				IsCompleted: true,
				Cards:       make(map[string]*CardProgress),
			},
		},
	}

	userProgress := UserCourseProgress{
		State: state,
	}

	lessonCursor, _ := pagination.ToCursor(LessonCursor{Order: 1})
	lessons := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{
					ID:       lessonID,
					CourseID: courseID,
					Order:    1,
					Title:    "Lesson 1",
					Body:     "Content with ![deck](" + oldDeckID.String() + ") and ![deck](" + newDeckID.String() + ")",
				},
				Cursor: lessonCursor,
			},
		},
	}

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}, false).Return(lessons, nil)
	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)
	mockRepo.On("UpdateUserProgress", ctx, mock.MatchedBy(func(ucp UserCourseProgress) bool {
		expectedState := &ProgressState{
			CurrentLessonID: lessonID,
			Lessons: map[string]*LessonProgress{
				lessonID.String(): {
					IsCompleted: false,
					Decks: map[string]*DeckProgress{
						oldDeckID.String(): {
							IsCompleted: true,
							Cards:       map[string]*CardProgress{},
						},
						newDeckID.String(): {
							IsCompleted: false,
							Cards: map[string]*CardProgress{
								newCardID.String(): {IsCompleted: false},
							},
						},
					},
				},
			},
		}
		return assert.EqualValues(t, expectedState, ucp.State) && ucp.CurrentLessonID == lessonID
	})).Return(nil)

	mockDecksClient.On("GetDeck", ctx, mock.MatchedBy(func(req *pbDeck.GetDeckRequest) bool {
		return req.DeckId == oldDeckID.String()
	})).Return(&pbDeck.GetDeckResponse{
		Deck: &pbDeck.Deck{Id: oldDeckID.String(), Title: "Old Deck", Cards: []*pbDeck.Card{}},
	}, nil)
	mockDecksClient.On("GetDeck", ctx, mock.MatchedBy(func(req *pbDeck.GetDeckRequest) bool {
		return req.DeckId == newDeckID.String()
	})).Return(&pbDeck.GetDeckResponse{
		Deck: &pbDeck.Deck{
			Id:    newDeckID.String(),
			Title: "New Deck",
			Cards: []*pbDeck.Card{{Id: newCardID.String(), Title: "Q"}},
		},
	}, nil)

	syncer := NewStateSyncer(mockRepo, mockDecksClient)

	err := syncer.Sync(ctx, userID, courseID)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)

	assert.NotNil(t, userProgress.State.Lessons[lessonID.String()].Decks[oldDeckID.String()])
	assert.NotNil(t, userProgress.State.Lessons[lessonID.String()].Decks[newDeckID.String()])
	assert.NotNil(t, userProgress.State.Lessons[lessonID.String()].Decks[newDeckID.String()].Cards[newCardID.String()])
	assert.False(t, userProgress.State.Lessons[lessonID.String()].IsCompleted)
}

func TestSync_NewCardReopensCompletedLesson(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lesson1ID := uuid.New()
	lesson2ID := uuid.New()
	deck1ID := uuid.New()
	deck2ID := uuid.New()
	answeredCardID := uuid.New()
	otherCardID := uuid.New()
	newCardID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	// Both lessons are completed, so the current lesson is the last one.
	state := NewProgressState()
	state.CurrentLessonID = lesson2ID
	state.Lessons[lesson1ID.String()] = &LessonProgress{
		IsCompleted: true,
		Decks: map[string]*DeckProgress{
			deck1ID.String(): {
				IsCompleted: true,
				Cards: map[string]*CardProgress{
					answeredCardID.String(): {IsCompleted: true, CorrectAnswers: 1},
				},
			},
		},
	}
	state.Lessons[lesson2ID.String()] = &LessonProgress{
		IsCompleted: true,
		Decks: map[string]*DeckProgress{
			deck2ID.String(): {
				IsCompleted: true,
				Cards: map[string]*CardProgress{
					otherCardID.String(): {IsCompleted: true, CorrectAnswers: 1},
				},
			},
		},
	}

	userProgress := UserCourseProgress{
		CurrentLessonID: lesson2ID,
		State:           state,
	}

	lesson1Cursor, _ := pagination.ToCursor(LessonCursor{Order: 1})
	lesson2Cursor, _ := pagination.ToCursor(LessonCursor{Order: 2})
	lessons := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{
					ID:       lesson1ID,
					CourseID: courseID,
					Order:    1,
					Title:    "Lesson 1",
					Body:     "Content with ![deck](" + deck1ID.String() + ")",
				},
				Cursor: lesson1Cursor,
			},
			{
				Lesson: Lesson{
					ID:       lesson2ID,
					CourseID: courseID,
					Order:    2,
					Title:    "Lesson 2",
					Body:     "Content with ![deck](" + deck2ID.String() + ")",
				},
				Cursor: lesson2Cursor,
			},
		},
	}

	expected := &ProgressState{
		CurrentLessonID: lesson1ID,
		Lessons: map[string]*LessonProgress{
			lesson1ID.String(): {
				IsCompleted: false,
				Decks: map[string]*DeckProgress{
					deck1ID.String(): {
						IsCompleted: false,
						Cards: map[string]*CardProgress{
							answeredCardID.String(): {IsCompleted: true, CorrectAnswers: 1},
							newCardID.String():      {IsCompleted: false},
						},
					},
				},
			},
			lesson2ID.String(): {
				IsCompleted: true,
				Decks: map[string]*DeckProgress{
					deck2ID.String(): {
						IsCompleted: true,
						Cards: map[string]*CardProgress{
							otherCardID.String(): {IsCompleted: true, CorrectAnswers: 1},
						},
					},
				},
			},
		},
	}

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}, false).Return(lessons, nil)
	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)
	mockRepo.On("UpdateUserProgress", ctx, mock.Anything).Return(nil)

	mockDecksClient.On("GetDeck", ctx, mock.MatchedBy(func(req *pbDeck.GetDeckRequest) bool {
		return req.DeckId == deck1ID.String()
	})).Return(&pbDeck.GetDeckResponse{
		Deck: &pbDeck.Deck{
			Id:    deck1ID.String(),
			Title: "Deck 1",
			Cards: []*pbDeck.Card{
				{Id: answeredCardID.String(), Title: "Q1"},
				{Id: newCardID.String(), Title: "New Q"},
			},
		},
	}, nil)
	mockDecksClient.On("GetDeck", ctx, mock.MatchedBy(func(req *pbDeck.GetDeckRequest) bool {
		return req.DeckId == deck2ID.String()
	})).Return(&pbDeck.GetDeckResponse{
		Deck: &pbDeck.Deck{
			Id:    deck2ID.String(),
			Title: "Deck 2",
			Cards: []*pbDeck.Card{{Id: otherCardID.String(), Title: "Q2"}},
		},
	}, nil)

	syncer := NewStateSyncer(mockRepo, mockDecksClient)

	err := syncer.Sync(ctx, userID, courseID)
	require.NoError(t, err)

	assert.Equal(t, expected, userProgress.State)
	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)
}

func TestSync_RemovesRemovedDeck(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lessonID := uuid.New()
	remainingDeckID := uuid.New()
	removedDeckID := uuid.New()
	cardID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	state := NewProgressState()
	state.Lessons[lessonID.String()] = &LessonProgress{
		IsCompleted: false,
		Decks: map[string]*DeckProgress{
			remainingDeckID.String(): {
				IsCompleted: false,
				Cards: map[string]*CardProgress{
					cardID.String(): {IsCompleted: false},
				},
			},
			removedDeckID.String(): {
				IsCompleted: false,
				Cards:       make(map[string]*CardProgress),
			},
		},
	}

	userProgress := UserCourseProgress{
		State: state,
	}

	lessonCursor, _ := pagination.ToCursor(LessonCursor{Order: 1})
	lessons := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{
					ID:       lessonID,
					CourseID: courseID,
					Order:    1,
					Title:    "Lesson 1",
					Body:     "Content with ![deck](" + remainingDeckID.String() + ")",
				},
				Cursor: lessonCursor,
			},
		},
	}

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}, false).Return(lessons, nil)
	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)
	mockRepo.On("UpdateUserProgress", ctx, mock.MatchedBy(func(ucp UserCourseProgress) bool {
		expectedState := &ProgressState{
			CurrentLessonID: lessonID,
			Lessons: map[string]*LessonProgress{
				lessonID.String(): {
					IsCompleted: false,
					Decks: map[string]*DeckProgress{
						remainingDeckID.String(): {
							IsCompleted: false,
							Cards: map[string]*CardProgress{
								cardID.String(): {IsCompleted: false},
							},
						},
					},
				},
			},
		}
		return assert.EqualValues(t, expectedState, ucp.State) && ucp.CurrentLessonID == lessonID
	})).Return(nil)

	mockDecksClient.On("GetDeck", ctx, mock.MatchedBy(func(req *pbDeck.GetDeckRequest) bool {
		return req.DeckId == remainingDeckID.String()
	})).Return(&pbDeck.GetDeckResponse{
		Deck: &pbDeck.Deck{Id: remainingDeckID.String(), Title: "Remaining Deck", Cards: []*pbDeck.Card{{Id: cardID.String(), Title: "Q"}}},
	}, nil)

	syncer := NewStateSyncer(mockRepo, mockDecksClient)

	err := syncer.Sync(ctx, userID, courseID)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)

	assert.NotNil(t, userProgress.State.Lessons[lessonID.String()].Decks[remainingDeckID.String()])
	assert.Nil(t, userProgress.State.Lessons[lessonID.String()].Decks[removedDeckID.String()])
}

func TestSync_RemovesRemovedCard(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lessonID := uuid.New()
	deckID := uuid.New()
	remainingCardID := uuid.New()
	removedCardID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	state := NewProgressState()
	state.Lessons[lessonID.String()] = &LessonProgress{
		IsCompleted: false,
		Decks: map[string]*DeckProgress{
			deckID.String(): {
				IsCompleted: false,
				Cards: map[string]*CardProgress{
					remainingCardID.String(): {IsCompleted: false},
					removedCardID.String():   {IsCompleted: false},
				},
			},
		},
	}

	userProgress := UserCourseProgress{
		State: state,
	}

	lessonCursor, _ := pagination.ToCursor(LessonCursor{Order: 1})
	lessons := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{
					ID:       lessonID,
					CourseID: courseID,
					Order:    1,
					Title:    "Lesson 1",
					Body:     "Content with ![deck](" + deckID.String() + ")",
				},
				Cursor: lessonCursor,
			},
		},
	}

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}, false).Return(lessons, nil)
	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)
	mockRepo.On("UpdateUserProgress", ctx, mock.MatchedBy(func(ucp UserCourseProgress) bool {
		expectedState := &ProgressState{
			CurrentLessonID: lessonID,
			Lessons: map[string]*LessonProgress{
				lessonID.String(): {
					IsCompleted: false,
					Decks: map[string]*DeckProgress{
						deckID.String(): {
							IsCompleted: false,
							Cards: map[string]*CardProgress{
								remainingCardID.String(): {IsCompleted: false},
							},
						},
					},
				},
			},
		}
		return assert.EqualValues(t, expectedState, ucp.State) && ucp.CurrentLessonID == lessonID
	})).Return(nil)

	mockDecksClient.On("GetDeck", ctx, mock.MatchedBy(func(req *pbDeck.GetDeckRequest) bool {
		return req.DeckId == deckID.String()
	})).Return(&pbDeck.GetDeckResponse{
		Deck: &pbDeck.Deck{
			Id:    deckID.String(),
			Title: "Test Deck",
			Cards: []*pbDeck.Card{{Id: remainingCardID.String(), Title: "Q1"}},
		},
	}, nil)

	syncer := NewStateSyncer(mockRepo, mockDecksClient)

	err := syncer.Sync(ctx, userID, courseID)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)

	assert.NotNil(t, userProgress.State.Lessons[lessonID.String()].Decks[deckID.String()].Cards[remainingCardID.String()])
	assert.Nil(t, userProgress.State.Lessons[lessonID.String()].Decks[deckID.String()].Cards[removedCardID.String()])
}

func TestSync_RemovesRemovedLesson(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	remainingLessonID := uuid.New()
	removedLessonID := uuid.New()
	deckID := uuid.New()
	cardID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	state := NewProgressState()
	state.Lessons[remainingLessonID.String()] = &LessonProgress{
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
	state.Lessons[removedLessonID.String()] = &LessonProgress{
		IsCompleted: false,
		Decks:       make(map[string]*DeckProgress),
	}

	userProgress := UserCourseProgress{
		State: state,
	}

	lessonCursor, _ := pagination.ToCursor(LessonCursor{Order: 1})
	lessons := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{
					ID:       remainingLessonID,
					CourseID: courseID,
					Order:    1,
					Title:    "Lesson 1",
					Body:     "Content with ![deck](" + deckID.String() + ")",
				},
				Cursor: lessonCursor,
			},
		},
	}

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}, false).Return(lessons, nil)
	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)
	mockRepo.On("UpdateUserProgress", ctx, mock.MatchedBy(func(ucp UserCourseProgress) bool {
		expectedState := &ProgressState{
			CurrentLessonID: remainingLessonID,
			Lessons: map[string]*LessonProgress{
				remainingLessonID.String(): {
					IsCompleted: false,
					Decks: map[string]*DeckProgress{
						deckID.String(): {
							IsCompleted: false,
							Cards: map[string]*CardProgress{
								cardID.String(): {IsCompleted: false},
							},
						},
					},
				},
			},
		}
		return assert.EqualValues(t, expectedState, ucp.State) && ucp.CurrentLessonID == remainingLessonID
	})).Return(nil)

	mockDecksClient.On("GetDeck", ctx, mock.MatchedBy(func(req *pbDeck.GetDeckRequest) bool {
		return req.DeckId == deckID.String()
	})).Return(&pbDeck.GetDeckResponse{
		Deck: &pbDeck.Deck{Id: deckID.String(), Title: "Test Deck", Cards: []*pbDeck.Card{{Id: cardID.String(), Title: "Q"}}},
	}, nil)

	syncer := NewStateSyncer(mockRepo, mockDecksClient)

	err := syncer.Sync(ctx, userID, courseID)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)

	assert.NotNil(t, userProgress.State.Lessons[remainingLessonID.String()])
	assert.Nil(t, userProgress.State.Lessons[removedLessonID.String()])
}

func TestSync_RemovedIncompleteCardCompletesDeckAndLesson(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lesson1ID := uuid.New()
	lesson2ID := uuid.New()
	deck1ID := uuid.New()
	deck2ID := uuid.New()
	answeredCardID := uuid.New()
	removedCardID := uuid.New()
	lesson2CardID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	// The learner answered every card of lesson 1 except the one that gets removed.
	state := NewProgressState()
	state.CurrentLessonID = lesson1ID
	state.Lessons[lesson1ID.String()] = &LessonProgress{
		Decks: map[string]*DeckProgress{
			deck1ID.String(): {
				Cards: map[string]*CardProgress{
					answeredCardID.String(): {IsCompleted: true, CorrectAnswers: 1},
					removedCardID.String():  {IsCompleted: false},
				},
			},
		},
	}
	state.Lessons[lesson2ID.String()] = &LessonProgress{
		Decks: map[string]*DeckProgress{
			deck2ID.String(): {
				Cards: map[string]*CardProgress{
					lesson2CardID.String(): {IsCompleted: false},
				},
			},
		},
	}

	userProgress := UserCourseProgress{
		CurrentLessonID: lesson1ID,
		State:           state,
	}

	lesson1Cursor, _ := pagination.ToCursor(LessonCursor{Order: 1})
	lesson2Cursor, _ := pagination.ToCursor(LessonCursor{Order: 2})
	lessons := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{
					ID:       lesson1ID,
					CourseID: courseID,
					Order:    1,
					Title:    "Lesson 1",
					Body:     "Content with ![deck](" + deck1ID.String() + ")",
				},
				Cursor: lesson1Cursor,
			},
			{
				Lesson: Lesson{
					ID:       lesson2ID,
					CourseID: courseID,
					Order:    2,
					Title:    "Lesson 2",
					Body:     "Content with ![deck](" + deck2ID.String() + ")",
				},
				Cursor: lesson2Cursor,
			},
		},
	}

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}, false).Return(lessons, nil)
	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)
	mockRepo.On("UpdateUserProgress", ctx, mock.MatchedBy(func(ucp UserCourseProgress) bool {
		return ucp.CurrentLessonID == lesson2ID
	})).Return(nil)

	mockDecksClient.On("GetDeck", ctx, mock.MatchedBy(func(req *pbDeck.GetDeckRequest) bool {
		return req.DeckId == deck1ID.String()
	})).Return(&pbDeck.GetDeckResponse{
		Deck: &pbDeck.Deck{
			Id:    deck1ID.String(),
			Title: "Deck 1",
			Cards: []*pbDeck.Card{{Id: answeredCardID.String(), Title: "Q1"}},
		},
	}, nil)
	mockDecksClient.On("GetDeck", ctx, mock.MatchedBy(func(req *pbDeck.GetDeckRequest) bool {
		return req.DeckId == deck2ID.String()
	})).Return(&pbDeck.GetDeckResponse{
		Deck: &pbDeck.Deck{
			Id:    deck2ID.String(),
			Title: "Deck 2",
			Cards: []*pbDeck.Card{{Id: lesson2CardID.String(), Title: "Q2"}},
		},
	}, nil)

	syncer := NewStateSyncer(mockRepo, mockDecksClient)

	err := syncer.Sync(ctx, userID, courseID)
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)

	actual := userProgress.State
	require.NotNil(t, actual.Lessons[lesson1ID.String()].CompletedAt)
	require.NotNil(t, actual.Lessons[lesson1ID.String()].Decks[deck1ID.String()].CompletedAt)
	expected := &ProgressState{
		CurrentLessonID: lesson2ID,
		Lessons: map[string]*LessonProgress{
			lesson1ID.String(): {
				IsCompleted: true,
				CompletedAt: actual.Lessons[lesson1ID.String()].CompletedAt,
				Decks: map[string]*DeckProgress{
					deck1ID.String(): {
						IsCompleted: true,
						CompletedAt: actual.Lessons[lesson1ID.String()].Decks[deck1ID.String()].CompletedAt,
						Cards: map[string]*CardProgress{
							answeredCardID.String(): {IsCompleted: true, CorrectAnswers: 1},
						},
					},
				},
			},
			lesson2ID.String(): {
				Decks: map[string]*DeckProgress{
					deck2ID.String(): {
						Cards: map[string]*CardProgress{
							lesson2CardID.String(): {IsCompleted: false},
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expected, actual)
}

func TestSync_RemovedCompletedCardKeepsDeckIncomplete(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lessonID := uuid.New()
	deckID := uuid.New()
	removedCardID := uuid.New()
	remainingCardID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	state := NewProgressState()
	state.CurrentLessonID = lessonID
	state.Lessons[lessonID.String()] = &LessonProgress{
		Decks: map[string]*DeckProgress{
			deckID.String(): {
				Cards: map[string]*CardProgress{
					removedCardID.String():   {IsCompleted: true, CorrectAnswers: 1},
					remainingCardID.String(): {IsCompleted: false},
				},
			},
		},
	}

	userProgress := UserCourseProgress{
		CurrentLessonID: lessonID,
		State:           state,
	}

	lessonCursor, _ := pagination.ToCursor(LessonCursor{Order: 1})
	lessons := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{
					ID:       lessonID,
					CourseID: courseID,
					Order:    1,
					Title:    "Lesson 1",
					Body:     "Content with ![deck](" + deckID.String() + ")",
				},
				Cursor: lessonCursor,
			},
		},
	}

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}, false).Return(lessons, nil)
	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)
	mockRepo.On("UpdateUserProgress", ctx, mock.MatchedBy(func(ucp UserCourseProgress) bool {
		return ucp.CurrentLessonID == lessonID
	})).Return(nil)

	mockDecksClient.On("GetDeck", ctx, mock.MatchedBy(func(req *pbDeck.GetDeckRequest) bool {
		return req.DeckId == deckID.String()
	})).Return(&pbDeck.GetDeckResponse{
		Deck: &pbDeck.Deck{
			Id:    deckID.String(),
			Title: "Deck",
			Cards: []*pbDeck.Card{{Id: remainingCardID.String(), Title: "Q"}},
		},
	}, nil)

	syncer := NewStateSyncer(mockRepo, mockDecksClient)

	err := syncer.Sync(ctx, userID, courseID)
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)

	expected := &ProgressState{
		CurrentLessonID: lessonID,
		Lessons: map[string]*LessonProgress{
			lessonID.String(): {
				Decks: map[string]*DeckProgress{
					deckID.String(): {
						Cards: map[string]*CardProgress{
							remainingCardID.String(): {IsCompleted: false},
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expected, userProgress.State)
}

func TestSync_RemovedDeckEmbedCompletesLesson(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lessonID := uuid.New()
	finishedDeckID := uuid.New()
	removedDeckID := uuid.New()
	finishedCardID := uuid.New()
	unansweredCardID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	// The learner finished one deck; the lesson body then stops embedding the other one.
	finishedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	state := NewProgressState()
	state.CurrentLessonID = lessonID
	state.Lessons[lessonID.String()] = &LessonProgress{
		Decks: map[string]*DeckProgress{
			finishedDeckID.String(): {
				IsCompleted: true,
				CompletedAt: &finishedAt,
				Cards: map[string]*CardProgress{
					finishedCardID.String(): {IsCompleted: true, CorrectAnswers: 1},
				},
			},
			removedDeckID.String(): {
				Cards: map[string]*CardProgress{
					unansweredCardID.String(): {IsCompleted: false},
				},
			},
		},
	}

	userProgress := UserCourseProgress{
		CurrentLessonID: lessonID,
		State:           state,
	}

	lessonCursor, _ := pagination.ToCursor(LessonCursor{Order: 1})
	lessons := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{
					ID:       lessonID,
					CourseID: courseID,
					Order:    1,
					Title:    "Lesson 1",
					Body:     "Content with ![deck](" + finishedDeckID.String() + ")",
				},
				Cursor: lessonCursor,
			},
		},
	}

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}, false).Return(lessons, nil)
	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)
	mockRepo.On("UpdateUserProgress", ctx, mock.MatchedBy(func(ucp UserCourseProgress) bool {
		return ucp.CurrentLessonID == lessonID
	})).Return(nil)

	mockDecksClient.On("GetDeck", ctx, mock.MatchedBy(func(req *pbDeck.GetDeckRequest) bool {
		return req.DeckId == finishedDeckID.String()
	})).Return(&pbDeck.GetDeckResponse{
		Deck: &pbDeck.Deck{
			Id:    finishedDeckID.String(),
			Title: "Finished Deck",
			Cards: []*pbDeck.Card{{Id: finishedCardID.String(), Title: "Q"}},
		},
	}, nil)

	syncer := NewStateSyncer(mockRepo, mockDecksClient)

	err := syncer.Sync(ctx, userID, courseID)
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)

	actual := userProgress.State
	require.NotNil(t, actual.Lessons[lessonID.String()].CompletedAt)
	expected := &ProgressState{
		CurrentLessonID: lessonID,
		Lessons: map[string]*LessonProgress{
			lessonID.String(): {
				IsCompleted: true,
				CompletedAt: actual.Lessons[lessonID.String()].CompletedAt,
				Decks: map[string]*DeckProgress{
					finishedDeckID.String(): {
						IsCompleted: true,
						CompletedAt: &finishedAt,
						Cards: map[string]*CardProgress{
							finishedCardID.String(): {IsCompleted: true, CorrectAnswers: 1},
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expected, actual)
}

func TestSync_SetsCurrrentLessonToFirstIncomplete(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lesson1ID := uuid.New()
	lesson2ID := uuid.New()
	deckID := uuid.New()
	cardID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	state := NewProgressState()
	state.Lessons[lesson1ID.String()] = &LessonProgress{
		IsCompleted: true,
		Decks: map[string]*DeckProgress{
			deckID.String(): {
				IsCompleted: true,
				Cards: map[string]*CardProgress{
					cardID.String(): {IsCompleted: true, CorrectAnswers: 1},
				},
			},
		},
	}
	state.Lessons[lesson2ID.String()] = &LessonProgress{
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

	lessonCursor1, _ := pagination.ToCursor(LessonCursor{Order: 1})
	lessonCursor2, _ := pagination.ToCursor(LessonCursor{Order: 2})
	lessons := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{
					ID:       lesson1ID,
					CourseID: courseID,
					Order:    1,
					Title:    "Lesson 1",
					Body:     "Content with ![deck](" + deckID.String() + ")",
				},
				Cursor: lessonCursor1,
			},
			{
				Lesson: Lesson{
					ID:       lesson2ID,
					CourseID: courseID,
					Order:    2,
					Title:    "Lesson 2",
					Body:     "Content with ![deck](" + deckID.String() + ")",
				},
				Cursor: lessonCursor2,
			},
		},
	}

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}, false).Return(lessons, nil)
	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)
	mockRepo.On("UpdateUserProgress", ctx, mock.MatchedBy(func(ucp UserCourseProgress) bool {
		expectedState := &ProgressState{
			CurrentLessonID: lesson2ID,
			Lessons: map[string]*LessonProgress{
				lesson1ID.String(): {
					IsCompleted: true,
					Decks: map[string]*DeckProgress{
						deckID.String(): {
							IsCompleted: true,
							Cards: map[string]*CardProgress{
								cardID.String(): {IsCompleted: true, CorrectAnswers: 1},
							},
						},
					},
				},
				lesson2ID.String(): {
					IsCompleted: false,
					Decks: map[string]*DeckProgress{
						deckID.String(): {
							IsCompleted: false,
							Cards: map[string]*CardProgress{
								cardID.String(): {IsCompleted: false},
							},
						},
					},
				},
			},
		}
		return assert.EqualValues(t, expectedState, ucp.State) && ucp.CurrentLessonID == lesson2ID
	})).Return(nil)

	mockDecksClient.On("GetDeck", ctx, mock.MatchedBy(func(req *pbDeck.GetDeckRequest) bool {
		return req.DeckId == deckID.String()
	})).Return(&pbDeck.GetDeckResponse{
		Deck: &pbDeck.Deck{Id: deckID.String(), Title: "Test Deck", Cards: []*pbDeck.Card{{Id: cardID.String(), Title: "Q"}}},
	}, nil)

	syncer := NewStateSyncer(mockRepo, mockDecksClient)

	err := syncer.Sync(ctx, userID, courseID)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)

	assert.Equal(t, lesson2ID, userProgress.State.CurrentLessonID)
}

func TestSync_NilProgressStateReturnsError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()

	mockRepo := new(RepositoryMock)

	userProgress := UserCourseProgress{
		State: nil,
	}

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}, false).Return(LessonsConnection{}, nil)
	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)

	syncer := &stateSyncer{
		repo: mockRepo,
	}

	err := syncer.Sync(ctx, userID, courseID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "user progress state is nil")
}

func TestSync_Complex_MultiLessonMultiDeckScenario(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()

	lesson1ID := uuid.New()
	lesson2ID := uuid.New()
	lesson3ID := uuid.New()

	lesson1Deck1ID := uuid.New()
	lesson1Deck2ID := uuid.New()
	lesson2Deck1ID := uuid.New()

	card1 := uuid.New()
	card2 := uuid.New()
	card3 := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	// Initial state: lesson1 and lesson2 exist with some decks/cards, lesson3 is being added
	state := NewProgressState()
	state.Lessons[lesson1ID.String()] = &LessonProgress{
		IsCompleted: false,
		Decks: map[string]*DeckProgress{
			lesson1Deck1ID.String(): {
				IsCompleted: false,
				Cards: map[string]*CardProgress{
					card1.String(): {IsCompleted: false},
				},
			},
		},
	}
	state.Lessons[lesson2ID.String()] = &LessonProgress{
		IsCompleted: true,
		Decks: map[string]*DeckProgress{
			lesson2Deck1ID.String(): {
				IsCompleted: true,
				Cards: map[string]*CardProgress{
					card2.String(): {IsCompleted: true},
				},
			},
		},
	}

	userProgress := UserCourseProgress{
		CurrentLessonID: lesson1ID,
		State:           state,
	}

	lesson1Cursor, _ := pagination.ToCursor(LessonCursor{Order: 1})
	lesson2Cursor, _ := pagination.ToCursor(LessonCursor{Order: 2})
	lesson3Cursor, _ := pagination.ToCursor(LessonCursor{Order: 3})

	lessons := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{
					ID:       lesson1ID,
					CourseID: courseID,
					Order:    1,
					Title:    "Lesson 1",
					Body:     "Content with ![deck](" + lesson1Deck1ID.String() + ") and ![deck](" + lesson1Deck2ID.String() + ")",
				},
				Cursor: lesson1Cursor,
			},
			{
				Lesson: Lesson{
					ID:       lesson2ID,
					CourseID: courseID,
					Order:    2,
					Title:    "Lesson 2",
					Body:     "Content with ![deck](" + lesson2Deck1ID.String() + ")",
				},
				Cursor: lesson2Cursor,
			},
			{
				Lesson: Lesson{
					ID:       lesson3ID,
					CourseID: courseID,
					Order:    3,
					Title:    "Lesson 3",
					Body:     "Content with ![deck](" + lesson1Deck1ID.String() + ")",
				},
				Cursor: lesson3Cursor,
			},
		},
	}

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}, false).Return(lessons, nil)
	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)
	mockRepo.On("UpdateUserProgress", ctx, mock.MatchedBy(func(ucp UserCourseProgress) bool {
		expectedState := &ProgressState{
			CurrentLessonID: lesson1ID,
			Lessons: map[string]*LessonProgress{
				lesson1ID.String(): {
					IsCompleted: false,
					Decks: map[string]*DeckProgress{
						lesson1Deck1ID.String(): {
							IsCompleted: false,
							Cards: map[string]*CardProgress{
								card1.String(): {IsCompleted: false},
							},
						},
						lesson1Deck2ID.String(): {
							IsCompleted: false,
							Cards: map[string]*CardProgress{
								card3.String(): {IsCompleted: false},
							},
						},
					},
				},
				lesson2ID.String(): {
					IsCompleted: true,
					Decks: map[string]*DeckProgress{
						lesson2Deck1ID.String(): {
							IsCompleted: true,
							Cards: map[string]*CardProgress{
								card2.String(): {IsCompleted: true},
							},
						},
					},
				},
				lesson3ID.String(): {
					IsCompleted: false,
					Decks: map[string]*DeckProgress{
						lesson1Deck1ID.String(): {
							IsCompleted: false,
							Cards: map[string]*CardProgress{
								card1.String(): {IsCompleted: false},
							},
						},
					},
				},
			},
		}
		return assert.EqualValues(t, expectedState, ucp.State) && ucp.CurrentLessonID == lesson1ID
	})).Return(nil)

	// Expects calls for lesson1 and lesson2, lesson3 will be called once
	mockDecksClient.On("GetDeck", ctx, mock.MatchedBy(func(req *pbDeck.GetDeckRequest) bool {
		return req.DeckId == lesson1Deck1ID.String()
	})).Return(&pbDeck.GetDeckResponse{
		Deck: &pbDeck.Deck{
			Id:    lesson1Deck1ID.String(),
			Title: "Lesson1 Deck1",
			Cards: []*pbDeck.Card{{Id: card1.String(), Title: "Q1"}},
		},
	}, nil)

	mockDecksClient.On("GetDeck", ctx, mock.MatchedBy(func(req *pbDeck.GetDeckRequest) bool {
		return req.DeckId == lesson1Deck2ID.String()
	})).Return(&pbDeck.GetDeckResponse{
		Deck: &pbDeck.Deck{
			Id:    lesson1Deck2ID.String(),
			Title: "Lesson1 Deck2",
			Cards: []*pbDeck.Card{{Id: card3.String(), Title: "Q3"}},
		},
	}, nil)

	mockDecksClient.On("GetDeck", ctx, mock.MatchedBy(func(req *pbDeck.GetDeckRequest) bool {
		return req.DeckId == lesson2Deck1ID.String()
	})).Return(&pbDeck.GetDeckResponse{
		Deck: &pbDeck.Deck{
			Id:    lesson2Deck1ID.String(),
			Title: "Lesson2 Deck1",
			Cards: []*pbDeck.Card{{Id: card2.String(), Title: "Q2"}},
		},
	}, nil)

	syncer := NewStateSyncer(mockRepo, mockDecksClient)

	err := syncer.Sync(ctx, userID, courseID)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)

	// Check final state
	assert.Equal(t, lesson1ID, userProgress.State.CurrentLessonID)
	assert.NotNil(t, userProgress.State.Lessons[lesson1ID.String()])
	assert.NotNil(t, userProgress.State.Lessons[lesson2ID.String()])
	assert.NotNil(t, userProgress.State.Lessons[lesson3ID.String()])
	assert.NotNil(t, userProgress.State.Lessons[lesson1ID.String()].Decks[lesson1Deck2ID.String()])
}

func TestSync_NoChangesWhenAlreadyInSync(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lessonID := uuid.New()
	deckID := uuid.New()
	cardID := uuid.New()

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

	userProgress := UserCourseProgress{State: state}

	lessonCursor, _ := pagination.ToCursor(LessonCursor{Order: 1})
	lessons := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{
					ID:       lessonID,
					CourseID: courseID,
					Order:    1,
					Title:    "Lesson 1",
					Body:     "Content with ![deck](" + deckID.String() + ")",
				},
				Cursor: lessonCursor,
			},
		},
	}

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}, false).Return(lessons, nil)
	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)
	mockRepo.On("UpdateUserProgress", ctx, mock.MatchedBy(func(ucp UserCourseProgress) bool {
		expectedState := &ProgressState{
			CurrentLessonID: lessonID,
			Lessons: map[string]*LessonProgress{
				lessonID.String(): {
					IsCompleted: false,
					Decks: map[string]*DeckProgress{
						deckID.String(): {
							IsCompleted: false,
							Cards: map[string]*CardProgress{
								cardID.String(): {IsCompleted: false},
							},
						},
					},
				},
			},
		}
		return assert.EqualValues(t, expectedState, ucp.State) && ucp.CurrentLessonID == lessonID
	})).Return(nil)

	mockDecksClient.On("GetDeck", ctx, mock.MatchedBy(func(req *pbDeck.GetDeckRequest) bool {
		return req.DeckId == deckID.String()
	})).Return(&pbDeck.GetDeckResponse{
		Deck: &pbDeck.Deck{
			Id:    deckID.String(),
			Title: "Test Deck",
			Cards: []*pbDeck.Card{{Id: cardID.String(), Title: "Q"}},
		},
	}, nil)

	syncer := NewStateSyncer(mockRepo, mockDecksClient)

	err := syncer.Sync(ctx, userID, courseID)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)

	// State should be unchanged
	assert.Equal(t, lessonID, userProgress.State.CurrentLessonID)
	assert.Contains(t, userProgress.State.Lessons, lessonID.String())
	assert.Contains(t, userProgress.State.Lessons[lessonID.String()].Decks, deckID.String())
	assert.Contains(t, userProgress.State.Lessons[lessonID.String()].Decks[deckID.String()].Cards, cardID.String())
}

// Helper function for test
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
