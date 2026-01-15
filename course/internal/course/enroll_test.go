package course

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tilinna/clock"
	"google.golang.org/grpc"

	"github.com/XaviFP/toshokan/common/pagination"
	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
)

func TestEnroller_Enroll_Success(t *testing.T) {
	ctx := context.Background()
	userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
	courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
	lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
	deckID := uuid.MustParse("60766223-ff9f-4871-a497-f765c05a0c5e")
	cardID := uuid.MustParse("72bdff92-5bc8-4e1d-9217-d0b23e22ff33")

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	lessonCursor, _ := pagination.ToCursor(LessonCursor{Order: 1})
	lessons := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{
					ID:       lessonID,
					CourseID: courseID,
					Order:    1,
					Title:    "Introduction to Goroutines",
					Body:     "Content with ![deck](" + deckID.String() + ")",
				},
				Cursor: lessonCursor,
			},
		},
	}

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}, false).Return(lessons, nil)
	mockDecksClient.On("GetDeck", ctx, mock.MatchedBy(func(req *pbDeck.GetDeckRequest) bool {
		return req.DeckId == deckID.String()
	})).Return(&pbDeck.GetDeckResponse{
		Deck: &pbDeck.Deck{
			Id: deckID.String(),
			Cards: []*pbDeck.Card{
				{
					Id: cardID.String(),
				},
			},
		},
	}, nil)
	mockRepo.On("EnrollUserInCourse", ctx, userID, courseID, mock.MatchedBy(func(state ProgressState) bool {
		return state.Lessons != nil
	})).Return(nil)

	now := time.Date(2025, 12, 28, 11, 27, 5, 0, time.UTC)
	mockClock := clock.NewMock(now)
	enroller := NewEnroller(mockClock, mockRepo, mockDecksClient)
	progress, err := enroller.Enroll(ctx, userID, courseID)

	assert.NoError(t, err)
	assert.Equal(t, userID, progress.UserID)
	assert.Equal(t, courseID, progress.CourseID)
	assert.Equal(t, lessonID, progress.CurrentLessonID)
	assert.NotNil(t, progress.State)

	stateJSON, err := json.Marshal(progress.State)
	assert.NoError(t, err)

	assert.JSONEq(t, `{
		"current_lesson_id": "334ddbf8-1acc-405b-86d8-49f0d1ca636c",
		"lessons": {
		"334ddbf8-1acc-405b-86d8-49f0d1ca636c": {
			"decks": {
			"60766223-ff9f-4871-a497-f765c05a0c5e": {
				"cards": {
				"72bdff92-5bc8-4e1d-9217-d0b23e22ff33": {
					"correct_answers": 0,
					"incorrect_answers": 0,
					"is_completed": false
				}
				},
				"is_completed": false
			}
			},
			"is_completed": false
		}
		}
	}
	`, string(stateJSON))

	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)
}

func TestEnroller_Enroll_NoLessons(t *testing.T) {
	ctx := context.Background()
	userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
	courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}, false).Return(LessonsConnection{Edges: []LessonEdge{}}, nil)

	enroller := NewEnroller(clock.Realtime(), mockRepo, mockDecksClient)
	_, err := enroller.Enroll(ctx, userID, courseID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot enroll in a course with no lessons")
}

func TestEnroller_Enroll_NoDecks(t *testing.T) {
	ctx := context.Background()
	userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
	courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
	lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	lessonCursor, _ := pagination.ToCursor(LessonCursor{Order: 1})
	lessons := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{
					ID:       lessonID,
					CourseID: courseID,
					Order:    1,
					Title:    "Introduction to Goroutines",
					Body:     "Content with no decks",
				},
				Cursor: lessonCursor,
			},
		},
	}

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}, false).Return(lessons, nil)

	enroller := NewEnroller(clock.Realtime(), mockRepo, mockDecksClient)
	_, err := enroller.Enroll(ctx, userID, courseID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot enroll in a course with lessons that have no decks")
}

func TestEnroller_Enroll_GetDecksError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
	courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
	lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
	deckID := uuid.MustParse("60766223-ff9f-4871-a497-f765c05a0c5e")

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	lessonCursor, _ := pagination.ToCursor(LessonCursor{Order: 1})
	lessons := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{
					ID:       lessonID,
					CourseID: courseID,
					Order:    1,
					Title:    "Introduction to Goroutines",
					Body:     "Content with ![deck](" + deckID.String() + ")",
				},
				Cursor: lessonCursor,
			},
		},
	}

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}, false).Return(lessons, nil)
	mockDecksClient.On("GetDeck", ctx, mock.MatchedBy(func(req *pbDeck.GetDeckRequest) bool {
		return req.DeckId == deckID.String()
	})).Return((*pbDeck.GetDeckResponse)(nil), assert.AnError)

	enroller := NewEnroller(clock.Realtime(), mockRepo, mockDecksClient)
	_, err := enroller.Enroll(ctx, userID, courseID)

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)
}

func TestEnroller_Enroll_EnrollmentError(t *testing.T) {
	ctx := context.Background()
	userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
	courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
	lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
	deckID := uuid.MustParse("60766223-ff9f-4871-a497-f765c05a0c5e")
	cardID := uuid.MustParse("72bdff92-5bc8-4e1d-9217-d0b23e22ff33")

	mockRepo := new(RepositoryMock)
	mockDecksClient := new(MockDecksAPIClient)

	lessonCursor, _ := pagination.ToCursor(LessonCursor{Order: 1})
	lessons := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{
					ID:       lessonID,
					CourseID: courseID,
					Order:    1,
					Title:    "Introduction to Goroutines",
					Body:     "Content with ![deck](" + deckID.String() + ")",
				},
				Cursor: lessonCursor,
			},
		},
	}

	mockRepo.On("GetLessonsByCourseID", ctx, courseID, pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 1000}, false).Return(lessons, nil)
	mockDecksClient.On("GetDeck", ctx, mock.MatchedBy(func(req *pbDeck.GetDeckRequest) bool {
		return req.DeckId == deckID.String()
	})).Return(&pbDeck.GetDeckResponse{
		Deck: &pbDeck.Deck{
			Id: deckID.String(),
			Cards: []*pbDeck.Card{
				{
					Id: cardID.String(),
				},
			},
		},
	}, nil)
	mockRepo.On("EnrollUserInCourse", ctx, userID, courseID, mock.Anything).Return(assert.AnError)

	enroller := NewEnroller(clock.Realtime(), mockRepo, mockDecksClient)
	_, err := enroller.Enroll(ctx, userID, courseID)

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
	mockDecksClient.AssertExpectations(t)
}

type MockDecksAPIClient struct {
	mock.Mock
}

func (m *MockDecksAPIClient) GetDeck(ctx context.Context, in *pbDeck.GetDeckRequest, opts ...grpc.CallOption) (*pbDeck.GetDeckResponse, error) {
	args := m.Called(ctx, in)
	return args[0].(*pbDeck.GetDeckResponse), args.Error(1)
}

func (m *MockDecksAPIClient) GetDecks(ctx context.Context, in *pbDeck.GetDecksRequest, opts ...grpc.CallOption) (*pbDeck.GetDecksResponse, error) {
	args := m.Called(ctx, in)
	return args[0].(*pbDeck.GetDecksResponse), args.Error(1)
}

func (m *MockDecksAPIClient) CreateDeck(ctx context.Context, in *pbDeck.CreateDeckRequest, opts ...grpc.CallOption) (*pbDeck.CreateDeckResponse, error) {
	args := m.Called(ctx, in)
	return args[0].(*pbDeck.CreateDeckResponse), args.Error(1)
}

func (m *MockDecksAPIClient) DeleteDeck(ctx context.Context, in *pbDeck.DeleteDeckRequest, opts ...grpc.CallOption) (*pbDeck.DeleteDeckResponse, error) {
	args := m.Called(ctx, in)
	return args[0].(*pbDeck.DeleteDeckResponse), args.Error(1)
}

func (m *MockDecksAPIClient) GetPopularDecks(ctx context.Context, in *pbDeck.GetPopularDecksRequest, opts ...grpc.CallOption) (*pbDeck.GetPopularDecksResponse, error) {
	args := m.Called(ctx, in)
	return args[0].(*pbDeck.GetPopularDecksResponse), args.Error(1)
}

func (m *MockDecksAPIClient) CreateCard(ctx context.Context, in *pbDeck.CreateCardRequest, opts ...grpc.CallOption) (*pbDeck.CreateCardResponse, error) {
	args := m.Called(ctx, in)
	return args[0].(*pbDeck.CreateCardResponse), args.Error(1)
}

func (m *MockDecksAPIClient) GetCards(ctx context.Context, in *pbDeck.GetCardsRequest, opts ...grpc.CallOption) (*pbDeck.GetCardsResponse, error) {
	args := m.Called(ctx, in)
	return args[0].(*pbDeck.GetCardsResponse), args.Error(1)
}
