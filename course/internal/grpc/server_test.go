package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/XaviFP/toshokan/common/pagination"
	pb "github.com/XaviFP/toshokan/course/api/proto/v1"
	course "github.com/XaviFP/toshokan/course/internal/course"
)

func TestServer_GetCourse(t *testing.T) {
	repoMock := &course.RepositoryMock{}
	srv := &Server{Repository: repoMock}

	t.Run("success", func(t *testing.T) {
		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		course := course.Course{
			ID:          courseID,
			Title:       "Go Mastery",
			Description: "Master the Go programming language",
			CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		repoMock.On("GetCourse", mock.Anything, courseID).Return(course, nil)
		req := pb.GetCourseRequest{CourseId: courseID.String()}

		res, err := srv.GetCourse(context.Background(), &req)
		assert.NoError(t, err)
		assert.Equal(t, course.ID.String(), res.Course.Id)
		assert.Equal(t, course.Title, res.Course.Title)
		assert.Equal(t, course.Description, res.Course.Description)
	})

	t.Run("failure", func(t *testing.T) {
		courseID := uuid.MustParse("1f30a72f-5d7a-48da-a5c2-42efece6972a")
		repoMock.On("GetCourse", mock.Anything, courseID).Return(course.Course{}, assert.AnError)
		req := pb.GetCourseRequest{CourseId: courseID.String()}

		res, err := srv.GetCourse(context.Background(), &req)
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}

func TestServer_GetLesson(t *testing.T) {
	repoMock := &course.RepositoryMock{}
	srv := &Server{Repository: repoMock}

	t.Run("success", func(t *testing.T) {
		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		lesson := course.Lesson{
			ID:          lessonID,
			CourseID:    courseID,
			Order:       1,
			Title:       "Introduction to Goroutines",
			Description: "Learn about concurrent programming",
			Body:        "Content here",
			CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		repoMock.On("GetLesson", mock.Anything, lessonID).Return(lesson, nil)
		req := pb.GetLessonRequest{LessonId: lessonID.String()}

		res, err := srv.GetLesson(context.Background(), &req)
		assert.NoError(t, err)
		assert.Equal(t, lesson.ID.String(), res.Lesson.Id)
		assert.Equal(t, lesson.Title, res.Lesson.Title)
		assert.Equal(t, lesson.Description, res.Lesson.Description)
	})

	t.Run("failure", func(t *testing.T) {
		lessonID := uuid.MustParse("1f30a72f-5d7a-48da-a5c2-42efece6972a")
		repoMock.On("GetLesson", mock.Anything, lessonID).Return(course.Lesson{}, assert.AnError)
		req := pb.GetLessonRequest{LessonId: lessonID.String()}

		res, err := srv.GetLesson(context.Background(), &req)
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}

func TestServer_CreateCourse(t *testing.T) {
	successTitle := "Go Mastery"
	failureTitle := "Failed Course"

	t.Run("success", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		repoMock.On("StoreCourse", mock.Anything, mock.MatchedBy(func(c course.Course) bool {
			return c.Title == successTitle
		})).Return(nil)
		srv := &Server{Repository: repoMock}

		req := pb.CreateCourseRequest{
			Title:       successTitle,
			Description: "Master the Go programming language",
		}

		res, err := srv.CreateCourse(context.Background(), &req)
		assert.NoError(t, err)
		assert.Equal(t, successTitle, res.Course.Title)
		assert.NotEmpty(t, res.Course.Id)
	})

	t.Run("failure", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		repoMock.On("StoreCourse", mock.Anything, mock.MatchedBy(func(c course.Course) bool {
			return c.Title == failureTitle
		})).Return(assert.AnError)
		srv := &Server{Repository: repoMock}

		req := pb.CreateCourseRequest{
			Title:       failureTitle,
			Description: "This will fail",
		}

		res, err := srv.CreateCourse(context.Background(), &req)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("validation_error_no_title", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		req := pb.CreateCourseRequest{
			Title:       "",
			Description: "Missing title",
		}

		res, err := srv.CreateCourse(context.Background(), &req)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("validation_error_no_description", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		req := pb.CreateCourseRequest{
			Title:       "Valid Title",
			Description: "",
		}

		res, err := srv.CreateCourse(context.Background(), &req)
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}

func TestServer_EnrollUser(t *testing.T) {
	enroller := &course.EnrollerMock{}
	srv := &Server{Enroller: enroller}

	t.Run("success", func(t *testing.T) {
		userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		enroller.On("Enroll", mock.Anything, userID, courseID).Return(course.UserCourseProgress{}, nil)

		req := pb.EnrollUserRequest{
			UserId:   userID.String(),
			CourseId: courseID.String(),
		}

		res, err := srv.EnrollUser(context.Background(), &req)
		assert.NoError(t, err)
		assert.True(t, res.Success)
	})

	t.Run("failure", func(t *testing.T) {
		userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
		courseID := uuid.MustParse("1f30a72f-5d7a-48da-a5c2-42efece6972a")
		enroller.On("Enroll", mock.Anything, userID, courseID).Return(course.UserCourseProgress{}, assert.AnError)

		req := pb.EnrollUserRequest{
			UserId:   userID.String(),
			CourseId: courseID.String(),
		}

		res, err := srv.EnrollUser(context.Background(), &req)
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}

func TestServer_GetLessons(t *testing.T) {
	repoMock := &course.RepositoryMock{}
	browserMock := &course.LessonsBrowserMock{}
	srv := &Server{
		Repository:     repoMock,
		LessonsBrowser: browserMock,
	}

	t.Run("success", func(t *testing.T) {
		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		lessons := []course.Lesson{
			{
				ID:          uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c"),
				CourseID:    courseID,
				Order:       1,
				Title:       "Introduction",
				Description: "Getting started",
				Body:        "Content",
				CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		}

		conn := course.LessonsConnection{
			Edges: []course.LessonEdge{
				{
					Lesson: lessons[0],
					Cursor: "cursor1",
				},
			},
			PageInfo: pagination.PageInfo{
				HasNextPage:     false,
				HasPreviousPage: false,
				StartCursor:     "start",
				EndCursor:       "end",
			},
		}

		browseResult := course.BrowseResult{
			PublicLessons: &conn,
		}

		expectedPagination := pagination.NewOlderFirstPagination(
			pagination.WithFirst(10),
		)
		expectedOpts := course.BrowseOptions{}

		browserMock.On("Browse", mock.Anything, courseID, expectedPagination, expectedOpts).Return(browseResult, nil)

		req := pb.GetLessonsRequest{
			CourseId: courseID.String(),
			Pagination: &pb.Pagination{
				First: 10,
			},
		}

		res, err := srv.GetLessons(context.Background(), &req)
		assert.NoError(t, err)
		assert.Len(t, res.Lessons.Edges, 1)
		assert.Equal(t, lessons[0].Title, res.Lessons.Edges[0].Lesson.Title)
	})

	t.Run("error", func(t *testing.T) {
		courseID := uuid.MustParse("1f30a72f-5d7a-48da-a5c2-42efece6972a")

		expectedPagination := pagination.NewOlderFirstPagination(
			pagination.WithFirst(10),
		)
		expectedOpts := course.BrowseOptions{}

		browserMock.On("Browse", mock.Anything, courseID, expectedPagination, expectedOpts).Return(course.BrowseResult{}, assert.AnError)

		req := pb.GetLessonsRequest{
			CourseId: courseID.String(),
			Pagination: &pb.Pagination{
				First: 10,
			},
		}

		res, err := srv.GetLessons(context.Background(), &req)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	browserMock.AssertExpectations(t)
	repoMock.AssertExpectations(t)
}

func TestServer_GetUserProgress(t *testing.T) {
	repoMock := &course.RepositoryMock{}
	srv := &Server{Repository: repoMock}

	t.Run("success", func(t *testing.T) {
		userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")

		progress := course.UserCourseProgress{
			ID:              uuid.New(),
			UserID:          userID,
			CourseID:        courseID,
			CurrentLessonID: lessonID,
			State:           course.NewProgressState(),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		repoMock.On("GetUserCourseProgress", mock.Anything, userID, courseID).Return(progress, nil)

		req := pb.GetUserProgressRequest{
			UserId:   userID.String(),
			CourseId: courseID.String(),
		}

		res, err := srv.GetUserProgress(context.Background(), &req)
		assert.NoError(t, err)
		assert.Equal(t, userID.String(), res.Progress.UserId)
		assert.Equal(t, courseID.String(), res.Progress.CourseId)
		assert.Equal(t, lessonID.String(), res.Progress.CurrentLessonId)
	})

	t.Run("failure", func(t *testing.T) {
		userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
		courseID := uuid.MustParse("1f30a72f-5d7a-48da-a5c2-42efece6972a")

		repoMock.On("GetUserCourseProgress", mock.Anything, userID, courseID).Return(course.UserCourseProgress{}, assert.AnError)

		req := pb.GetUserProgressRequest{
			UserId:   userID.String(),
			CourseId: courseID.String(),
		}

		res, err := srv.GetUserProgress(context.Background(), &req)
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}
