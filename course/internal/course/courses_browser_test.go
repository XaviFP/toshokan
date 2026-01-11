package course

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/XaviFP/toshokan/common/pagination"
)

func TestCoursesBrowser_BrowseEnrolled_Success(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	course1ID := uuid.New()
	course2ID := uuid.New()
	lesson1ID := uuid.New()
	lesson2ID := uuid.New()

	mockRepo := new(RepositoryMock)

	expectedConn := CoursesWithProgressConnection{
		Edges: []CourseWithProgressEdge{
			{
				Course: &CourseWithProgress{
					Course: Course{
						ID:          course1ID,
						Title:       "Go Fundamentals",
						Description: "Learn Go basics",
						CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					},
					CurrentLessonID: lesson1ID.String(),
				},
				Cursor: "cursor1",
			},
			{
				Course: &CourseWithProgress{
					Course: Course{
						ID:          course2ID,
						Title:       "Advanced Go",
						Description: "Master Go",
						CreatedAt:   time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
					},
					CurrentLessonID: lesson2ID.String(),
				},
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
	mockRepo.On("GetEnrolledCourses", ctx, userID, p).Return(expectedConn, nil)

	browser := NewCoursesBrowser(mockRepo)
	result, err := browser.BrowseEnrolled(ctx, userID, p)
	require.NoError(t, err)

	assert.Equal(t, 2, len(result.Edges))
	assert.Equal(t, course1ID, result.Edges[0].Course.Course.ID)
	assert.Equal(t, "Go Fundamentals", result.Edges[0].Course.Course.Title)
	assert.Equal(t, lesson1ID.String(), result.Edges[0].Course.CurrentLessonID)
	assert.Equal(t, course2ID, result.Edges[1].Course.Course.ID)
	assert.Equal(t, "Advanced Go", result.Edges[1].Course.Course.Title)
	assert.Equal(t, lesson2ID.String(), result.Edges[1].Course.CurrentLessonID)
	mockRepo.AssertExpectations(t)
}

func TestCoursesBrowser_BrowseEnrolled_EmptyResult(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	mockRepo := new(RepositoryMock)

	expectedConn := CoursesWithProgressConnection{
		Edges: []CourseWithProgressEdge{},
		PageInfo: pagination.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     "",
			EndCursor:       "",
		},
	}

	p := pagination.Pagination{First: 10}
	mockRepo.On("GetEnrolledCourses", ctx, userID, p).Return(expectedConn, nil)

	browser := NewCoursesBrowser(mockRepo)
	result, err := browser.BrowseEnrolled(ctx, userID, p)
	require.NoError(t, err)

	assert.Equal(t, 0, len(result.Edges))
	mockRepo.AssertExpectations(t)
}
