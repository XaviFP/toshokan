package course

import (
	"context"
	"testing"

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

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}).Return(lessons, nil)
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

	state := NewProgressState()
	state.Lessons[lessonID.String()] = &LessonProgress{
		IsCompleted: false,
		Decks: map[string]*DeckProgress{
			oldDeckID.String(): {
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
					Body:     "Content with ![deck](" + oldDeckID.String() + ") and ![deck](" + newDeckID.String() + ")",
				},
				Cursor: lessonCursor,
			},
		},
	}

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}).Return(lessons, nil)
	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)
	mockRepo.On("UpdateUserProgress", ctx, mock.MatchedBy(func(ucp UserCourseProgress) bool {
		expectedState := &ProgressState{
			CurrentLessonID: lessonID,
			Lessons: map[string]*LessonProgress{
				lessonID.String(): {
					IsCompleted: false,
					Decks: map[string]*DeckProgress{
						oldDeckID.String(): {
							IsCompleted: false,
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
}

func TestSync_RemovesRemovedDeck(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lessonID := uuid.New()
	remainingDeckID := uuid.New()
	removedDeckID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	state := NewProgressState()
	state.Lessons[lessonID.String()] = &LessonProgress{
		IsCompleted: false,
		Decks: map[string]*DeckProgress{
			remainingDeckID.String(): {
				IsCompleted: false,
				Cards:       make(map[string]*CardProgress),
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

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}).Return(lessons, nil)
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
							Cards:       map[string]*CardProgress{},
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
		Deck: &pbDeck.Deck{Id: remainingDeckID.String(), Title: "Remaining Deck", Cards: []*pbDeck.Card{}},
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

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}).Return(lessons, nil)
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

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	state := NewProgressState()
	state.Lessons[remainingLessonID.String()] = &LessonProgress{
		IsCompleted: false,
		Decks: map[string]*DeckProgress{
			deckID.String(): {
				IsCompleted: false,
				Cards:       make(map[string]*CardProgress),
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

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}).Return(lessons, nil)
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
							Cards:       map[string]*CardProgress{},
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
		Deck: &pbDeck.Deck{Id: deckID.String(), Title: "Test Deck", Cards: []*pbDeck.Card{}},
	}, nil)

	syncer := NewStateSyncer(mockRepo, mockDecksClient)

	err := syncer.Sync(ctx, userID, courseID)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)

	assert.NotNil(t, userProgress.State.Lessons[remainingLessonID.String()])
	assert.Nil(t, userProgress.State.Lessons[removedLessonID.String()])
}

func TestSync_SetsCurrrentLessonToFirstIncomplete(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	lesson1ID := uuid.New()
	lesson2ID := uuid.New()
	deckID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	state := NewProgressState()
	state.Lessons[lesson1ID.String()] = &LessonProgress{
		IsCompleted: true,
		Decks: map[string]*DeckProgress{
			deckID.String(): {
				IsCompleted: true,
				Cards:       make(map[string]*CardProgress),
			},
		},
	}
	state.Lessons[lesson2ID.String()] = &LessonProgress{
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

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}).Return(lessons, nil)
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
							Cards:       map[string]*CardProgress{},
						},
					},
				},
				lesson2ID.String(): {
					IsCompleted: false,
					Decks: map[string]*DeckProgress{
						deckID.String(): {
							IsCompleted: false,
							Cards:       map[string]*CardProgress{},
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
		Deck: &pbDeck.Deck{Id: deckID.String(), Title: "Test Deck", Cards: []*pbDeck.Card{}},
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

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}).Return(LessonsConnection{}, nil)
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

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}).Return(lessons, nil)
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

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}).Return(lessons, nil)
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
