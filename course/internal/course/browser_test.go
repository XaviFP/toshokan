package course

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/XaviFP/toshokan/common/pagination"
)

func TestBrowser_Browse_WithoutUserContext_Success(t *testing.T) {
	ctx := context.Background()
	courseID := uuid.New()
	lesson1ID := uuid.New()
	lesson2ID := uuid.New()

	mockRepo := new(RepositoryMock)

	expectedConn := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{ID: lesson1ID, Title: "Lesson 1"},
				Cursor: "cursor1",
			},
			{
				Lesson: Lesson{ID: lesson2ID, Title: "Lesson 2"},
				Cursor: "cursor2",
			},
		},
		PageInfo: pagination.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     "cursor1",
			EndCursor:       "cursor2",
		},
	}

	p := pagination.Pagination{First: 10}
	mockRepo.On("GetLessonsByCourseID", ctx, courseID, p).Return(expectedConn, nil)

	mockSyncer := new(StateSyncerMock)

	browser := NewLessonsBrowser(mockRepo, mockSyncer)
	result, err := browser.Browse(ctx, courseID, p, BrowseOptions{UserID: nil})
	require.NoError(t, err)

	assert.NotNil(t, result.PublicLessons)
	assert.Nil(t, result.ProgressLessons)
	assert.Equal(t, 2, len(result.PublicLessons.Edges))
	assert.Equal(t, lesson1ID, result.PublicLessons.Edges[0].Lesson.ID)
	assert.Equal(t, lesson2ID, result.PublicLessons.Edges[1].Lesson.ID)
	mockRepo.AssertExpectations(t)
	mockSyncer.AssertNotCalled(t, "Sync", mock.Anything, mock.Anything, mock.Anything)
}

func TestBrowser_Browse_WithUserContext_Success(t *testing.T) {
	ctx := context.Background()
	courseID := uuid.New()
	userID := uuid.New()
	lesson1ID := uuid.New()
	lesson2ID := uuid.New()
	lesson3ID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockSyncer := new(StateSyncerMock)

	lessonsConn := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{ID: lesson1ID, Title: "Lesson 1"},
				Cursor: "cursor1",
			},
			{
				Lesson: Lesson{ID: lesson2ID, Title: "Lesson 2"},
				Cursor: "cursor2",
			},
			{
				Lesson: Lesson{ID: lesson3ID, Title: "Lesson 3"},
				Cursor: "cursor3",
			},
		},
		PageInfo: pagination.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     "cursor1",
			EndCursor:       "cursor3",
		},
	}

	state := NewProgressState()
	state.Lessons[lesson1ID.String()] = &LessonProgress{
		IsCompleted: true,
		Decks:       make(map[string]*DeckProgress),
	}
	deckID := uuid.New()
	cardID := uuid.New()
	state.Lessons[lesson2ID.String()] = &LessonProgress{
		IsCompleted: false,
		Decks: map[string]*DeckProgress{
			deckID.String(): {
				IsCompleted: false,
				Cards: map[string]*CardProgress{
					cardID.String(): {
						IsCompleted: false,
					},
				},
			},
		},
	}

	userProgress := UserCourseProgress{
		CurrentLessonID: lesson2ID,
		State:           state,
	}

	p := pagination.Pagination{First: 10}
	mockRepo.On("GetLessonsByCourseID", ctx, courseID, p).Return(lessonsConn, nil)
	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)

	syncerCallDone := make(chan struct{})
	mockSyncer.On("Sync", context.WithoutCancel(ctx), userID, courseID).
		Run(func(args mock.Arguments) {
			close(syncerCallDone)
		}).
		Return(nil)

	browser := NewLessonsBrowser(mockRepo, mockSyncer)
	result, err := browser.Browse(ctx, courseID, p, BrowseOptions{UserID: &userID})

	require.NoError(t, err)

	assert.Nil(t, result.PublicLessons)
	assert.NotNil(t, result.ProgressLessons)
	assert.Equal(t, 3, len(result.ProgressLessons.Edges))

	// Check lesson 1 - completed but not current
	assert.Equal(t, lesson1ID, result.ProgressLessons.Edges[0].Lesson.Lesson.ID)
	assert.True(t, result.ProgressLessons.Edges[0].Lesson.IsCompleted)
	assert.False(t, result.ProgressLessons.Edges[0].Lesson.IsCurrent)

	// Check lesson 2 - current but not completed
	assert.Equal(t, lesson2ID, result.ProgressLessons.Edges[1].Lesson.Lesson.ID)
	assert.False(t, result.ProgressLessons.Edges[1].Lesson.IsCompleted)
	assert.True(t, result.ProgressLessons.Edges[1].Lesson.IsCurrent)

	// Check lesson 3 - no progress entry means not completed, not current
	assert.Equal(t, lesson3ID, result.ProgressLessons.Edges[2].Lesson.Lesson.ID)
	assert.False(t, result.ProgressLessons.Edges[2].Lesson.IsCompleted)
	assert.False(t, result.ProgressLessons.Edges[2].Lesson.IsCurrent)

	mockRepo.AssertExpectations(t)

	waitForGoroutine(t, syncerCallDone)
	mockSyncer.AssertExpectations(t)
}

func TestBrowser_Browse_WithoutUserContext_ErrorGettingLessons(t *testing.T) {
	ctx := context.Background()
	courseID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockSyncer := new(StateSyncerMock)

	expectedErr := errors.New("database error")
	p := pagination.Pagination{First: 10}
	mockRepo.On("GetLessonsByCourseID", ctx, courseID, p).Return(LessonsConnection{}, expectedErr)

	browser := NewLessonsBrowser(mockRepo, mockSyncer)
	result, err := browser.Browse(ctx, courseID, p, BrowseOptions{UserID: nil})

	require.Error(t, err)

	assert.Nil(t, result.PublicLessons)
	assert.Nil(t, result.ProgressLessons)
	mockRepo.AssertExpectations(t)
}

func TestBrowser_Browse_WithUserContext_ErrorGettingLessons(t *testing.T) {
	ctx := context.Background()
	courseID := uuid.New()
	userID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockSyncer := new(StateSyncerMock)

	expectedErr := errors.New("database error")
	p := pagination.Pagination{First: 10}
	mockRepo.On("GetLessonsByCourseID", ctx, courseID, p).Return(LessonsConnection{}, expectedErr)

	browser := NewLessonsBrowser(mockRepo, mockSyncer)
	result, err := browser.Browse(ctx, courseID, p, BrowseOptions{UserID: &userID})

	require.Error(t, err)

	assert.Nil(t, result.PublicLessons)
	assert.Nil(t, result.ProgressLessons)
	mockRepo.AssertExpectations(t)
}

func TestBrowser_Browse_WithUserContext_ErrorGettingProgress(t *testing.T) {
	ctx := context.Background()
	courseID := uuid.New()
	userID := uuid.New()
	lessonID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockSyncer := new(StateSyncerMock)

	lessonsConn := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{ID: lessonID, Title: "Lesson 1"},
				Cursor: "cursor1",
			},
		},
		PageInfo: pagination.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     "cursor1",
			EndCursor:       "cursor1",
		},
	}

	expectedErr := errors.New("progress not found")
	p := pagination.Pagination{First: 10}
	mockRepo.On("GetLessonsByCourseID", ctx, courseID, p).Return(lessonsConn, nil)
	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(UserCourseProgress{}, expectedErr)

	browser := NewLessonsBrowser(mockRepo, mockSyncer)
	result, err := browser.Browse(ctx, courseID, p, BrowseOptions{UserID: &userID})

	require.Error(t, err)

	assert.Nil(t, result.PublicLessons)
	assert.Nil(t, result.ProgressLessons)
	mockRepo.AssertExpectations(t)
}

func TestBrowser_Browse_WithUserContext_EmptyLessons(t *testing.T) {
	ctx := context.Background()
	courseID := uuid.New()
	userID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockSyncer := new(StateSyncerMock)

	lessonsConn := LessonsConnection{
		Edges: []LessonEdge{},
		PageInfo: pagination.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
		},
	}

	state := NewProgressState()
	userProgress := UserCourseProgress{
		CurrentLessonID: uuid.Nil,
		State:           state,
	}

	p := pagination.Pagination{First: 10}
	mockRepo.On("GetLessonsByCourseID", ctx, courseID, p).Return(lessonsConn, nil)
	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)

	syncerCallDone := make(chan struct{})
	mockSyncer.On("Sync", context.WithoutCancel(ctx), userID, courseID).
		Run(func(args mock.Arguments) {
			close(syncerCallDone)
		}).
		Return(nil)

	browser := NewLessonsBrowser(mockRepo, mockSyncer)
	result, err := browser.Browse(ctx, courseID, p, BrowseOptions{UserID: &userID})

	require.NoError(t, err)

	assert.Nil(t, result.PublicLessons)
	assert.NotNil(t, result.ProgressLessons)
	assert.Equal(t, 0, len(result.ProgressLessons.Edges))
	mockRepo.AssertExpectations(t)

	waitForGoroutine(t, syncerCallDone)
	mockSyncer.AssertExpectations(t)
}

func TestBrowser_Browse_WithUserContext_AllLessonsCompleted(t *testing.T) {
	ctx := context.Background()
	courseID := uuid.New()
	userID := uuid.New()
	lesson1ID := uuid.New()
	lesson2ID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockSyncer := new(StateSyncerMock)

	lessonsConn := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{ID: lesson1ID, Title: "Lesson 1"},
				Cursor: "cursor1",
			},
			{
				Lesson: Lesson{ID: lesson2ID, Title: "Lesson 2"},
				Cursor: "cursor2",
			},
		},
		PageInfo: pagination.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     "cursor1",
			EndCursor:       "cursor2",
		},
	}

	state := NewProgressState()
	state.Lessons[lesson1ID.String()] = &LessonProgress{
		IsCompleted: true,
		Decks:       make(map[string]*DeckProgress),
	}
	state.Lessons[lesson2ID.String()] = &LessonProgress{
		IsCompleted: true,
		Decks:       make(map[string]*DeckProgress),
	}

	userProgress := UserCourseProgress{
		CurrentLessonID: uuid.Nil,
		State:           state,
	}

	p := pagination.Pagination{First: 10}
	mockRepo.On("GetLessonsByCourseID", ctx, courseID, p).Return(lessonsConn, nil)
	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)

	syncerCallDone := make(chan struct{})
	mockSyncer.On("Sync", context.WithoutCancel(ctx), userID, courseID).
		Run(func(args mock.Arguments) {
			close(syncerCallDone)
		}).
		Return(nil)

	browser := NewLessonsBrowser(mockRepo, mockSyncer)
	result, err := browser.Browse(ctx, courseID, p, BrowseOptions{UserID: &userID})

	require.NoError(t, err)

	assert.NotNil(t, result.ProgressLessons)
	assert.Equal(t, 2, len(result.ProgressLessons.Edges))

	// All lessons should be completed and none current
	for _, edge := range result.ProgressLessons.Edges {
		assert.True(t, edge.Lesson.IsCompleted)
		assert.False(t, edge.Lesson.IsCurrent)
	}

	mockRepo.AssertExpectations(t)

	waitForGoroutine(t, syncerCallDone)
	mockSyncer.AssertExpectations(t)
}

func TestBrowser_Browse_WithUserContext_PaginationPreserved(t *testing.T) {
	ctx := context.Background()
	courseID := uuid.New()
	userID := uuid.New()
	lesson1ID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockSyncer := new(StateSyncerMock)

	lessonsConn := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{ID: lesson1ID, Title: "Lesson 1"},
				Cursor: "cursor1",
			},
		},
		PageInfo: pagination.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     "cursor1",
			EndCursor:       "cursor1",
		},
	}

	state := NewProgressState()
	userProgress := UserCourseProgress{
		CurrentLessonID: uuid.Nil,
		State:           state,
	}

	p := pagination.Pagination{Kind: pagination.PaginationKindOldestFirst, First: 10}
	mockRepo.On("GetLessonsByCourseID", ctx, courseID, p).Return(lessonsConn, nil)
	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)

	syncerCallDone := make(chan struct{})
	mockSyncer.On("Sync", context.WithoutCancel(ctx), userID, courseID).
		Run(func(args mock.Arguments) {
			close(syncerCallDone)
		}).
		Return(nil)

	browser := NewLessonsBrowser(mockRepo, mockSyncer)
	result, err := browser.Browse(ctx, courseID, p, BrowseOptions{UserID: &userID})
	require.NoError(t, err)

	assert.True(t, result.ProgressLessons.PageInfo.HasNextPage)
	assert.True(t, result.ProgressLessons.PageInfo.HasPreviousPage)
	assert.Equal(t, pagination.Cursor("cursor1"), result.ProgressLessons.PageInfo.StartCursor)
	assert.Equal(t, pagination.Cursor("cursor1"), result.ProgressLessons.PageInfo.EndCursor)

	mockRepo.AssertExpectations(t)

	waitForGoroutine(t, syncerCallDone)
	mockSyncer.AssertExpectations(t)
}

func TestBrowser_Browse_WithUserContext_CursorsPreserved(t *testing.T) {
	ctx := context.Background()
	courseID := uuid.New()
	userID := uuid.New()
	lesson1ID := uuid.New()
	lesson2ID := uuid.New()

	mockRepo := new(RepositoryMock)
	mockSyncer := new(StateSyncerMock)

	lessonsConn := LessonsConnection{
		Edges: []LessonEdge{
			{
				Lesson: Lesson{ID: lesson1ID, Title: "Lesson 1"},
				Cursor: "custom_cursor_1",
			},
			{
				Lesson: Lesson{ID: lesson2ID, Title: "Lesson 2"},
				Cursor: "custom_cursor_2",
			},
		},
		PageInfo: pagination.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     "custom_cursor_1",
			EndCursor:       "custom_cursor_2",
		},
	}

	state := NewProgressState()
	userProgress := UserCourseProgress{
		CurrentLessonID: uuid.Nil,
		State:           state,
	}

	p := pagination.Pagination{First: 10}
	mockRepo.On("GetLessonsByCourseID", ctx, courseID, p).Return(lessonsConn, nil)
	mockRepo.On("GetUserCourseProgress", ctx, userID, courseID).Return(userProgress, nil)

	syncerCallDone := make(chan struct{})
	mockSyncer.On("Sync", context.WithoutCancel(ctx), userID, courseID).
		Run(func(args mock.Arguments) {
			close(syncerCallDone)
		}).
		Return(nil)

	browser := NewLessonsBrowser(mockRepo, mockSyncer)
	result, err := browser.Browse(ctx, courseID, p, BrowseOptions{UserID: &userID})

	require.NoError(t, err)

	assert.NotNil(t, result.ProgressLessons)
	assert.Equal(t, 2, len(result.ProgressLessons.Edges))

	// Cursors should be preserved from the original edges
	assert.Equal(t, pagination.Cursor("custom_cursor_1"), result.ProgressLessons.Edges[0].Cursor)
	assert.Equal(t, pagination.Cursor("custom_cursor_2"), result.ProgressLessons.Edges[1].Cursor)

	mockRepo.AssertExpectations(t)

	waitForGoroutine(t, syncerCallDone)
	mockSyncer.AssertExpectations(t)
}

// waitForGoroutine waits for a goroutine to signal completion via the provided channel.
// Same as just <-c but with a timeout to avoid hanging tests.
func waitForGoroutine(t *testing.T, c <-chan struct{}) {
	select {
	case <-c:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for goroutine")
	}
}

type StateSyncerMock struct {
	mock.Mock
}

func (m *StateSyncerMock) Sync(ctx context.Context, userID uuid.UUID, courseID uuid.UUID) error {
	args := m.Called(ctx, userID, courseID)
	return args.Error(0)
}
