package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"

	"github.com/XaviFP/toshokan/common/pagination"
	pb "github.com/XaviFP/toshokan/course/api/proto/v1"
	course "github.com/XaviFP/toshokan/course/internal/course"
	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
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

		expectedPagination := pagination.NewOldestFirstPagination(
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
		assert.Equal(t, lessons[0].Title, res.Lessons.Edges[0].Node.Title)
	})

	t.Run("error", func(t *testing.T) {
		courseID := uuid.MustParse("1f30a72f-5d7a-48da-a5c2-42efece6972a")

		expectedPagination := pagination.NewOldestFirstPagination(
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

func TestServer_SyncState(t *testing.T) {
	stateSyncerMock := &course.StateSyncerMock{}
	srv := &Server{StateSyncer: stateSyncerMock}

	t.Run("success", func(t *testing.T) {
		userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

		stateSyncerMock.On("Sync", mock.Anything, userID, courseID).Return(nil)

		req := pb.SyncStateRequest{
			UserId:   userID.String(),
			CourseId: courseID.String(),
		}

		res, err := srv.SyncState(context.Background(), &req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("failure", func(t *testing.T) {
		userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
		courseID := uuid.MustParse("1f30a72f-5d7a-48da-a5c2-42efece6972a")

		stateSyncerMock.On("Sync", mock.Anything, userID, courseID).Return(assert.AnError)

		req := pb.SyncStateRequest{
			UserId:   userID.String(),
			CourseId: courseID.String(),
		}

		res, err := srv.SyncState(context.Background(), &req)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("error_parse_user_id", func(t *testing.T) {
		req := pb.SyncStateRequest{}

		_, err := srv.SyncState(context.Background(), &req)
		assert.Error(t, err)
	})

	t.Run("error_parse_course_id", func(t *testing.T) {
		userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")

		req := pb.SyncStateRequest{
			UserId: userID.String(),
		}

		_, err := srv.SyncState(context.Background(), &req)
		assert.Error(t, err)
	})

	stateSyncerMock.AssertExpectations(t)
}

func TestServer_UpdateCourse(t *testing.T) {
	t.Run("success_update_title", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		newTitle := "Updated Title"
		updates := course.CourseUpdates{Title: &newTitle}

		updatedCourse := course.Course{
			ID:          courseID,
			Order:       1,
			Title:       newTitle,
			Description: "Original description",
			CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		repoMock.On("UpdateCourse", mock.Anything, courseID, updates).Return(updatedCourse, nil)

		req := &pb.UpdateCourseRequest{
			Id:    courseID.String(),
			Title: &newTitle,
		}

		res, err := srv.UpdateCourse(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, newTitle, res.Course.Title)
		assert.Equal(t, "Original description", res.Course.Description)
		repoMock.AssertExpectations(t)
	})

	t.Run("success_update_description", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		newDesc := "Updated Description"
		updates := course.CourseUpdates{Description: &newDesc}

		updatedCourse := course.Course{
			ID:          courseID,
			Order:       1,
			Title:       "Original Title",
			Description: newDesc,
			CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		repoMock.On("UpdateCourse", mock.Anything, courseID, updates).Return(updatedCourse, nil)

		req := &pb.UpdateCourseRequest{
			Id:          courseID.String(),
			Description: &newDesc,
		}

		res, err := srv.UpdateCourse(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, "Original Title", res.Course.Title)
		assert.Equal(t, newDesc, res.Course.Description)
		repoMock.AssertExpectations(t)
	})

	t.Run("success_update_order", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		newOrder := int64(42)
		updates := course.CourseUpdates{Order: &newOrder}

		updatedCourse := course.Course{
			ID:          courseID,
			Order:       42,
			Title:       "Original Title",
			Description: "Original description",
			CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		repoMock.On("UpdateCourse", mock.Anything, courseID, updates).Return(updatedCourse, nil)

		req := &pb.UpdateCourseRequest{
			Id:    courseID.String(),
			Order: &newOrder,
		}

		res, err := srv.UpdateCourse(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, int64(42), res.Course.Order)
		repoMock.AssertExpectations(t)
	})

	t.Run("success_update_multiple_fields", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		newTitle := "New Title"
		newDesc := "New Description"
		newOrder := int64(99)
		updates := course.CourseUpdates{Title: &newTitle, Description: &newDesc, Order: &newOrder}

		updatedCourse := course.Course{
			ID:          courseID,
			Order:       99,
			Title:       newTitle,
			Description: newDesc,
			CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		repoMock.On("UpdateCourse", mock.Anything, courseID, updates).Return(updatedCourse, nil)

		req := &pb.UpdateCourseRequest{
			Id:          courseID.String(),
			Title:       &newTitle,
			Description: &newDesc,
			Order:       &newOrder,
		}

		res, err := srv.UpdateCourse(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, newTitle, res.Course.Title)
		assert.Equal(t, newDesc, res.Course.Description)
		assert.Equal(t, int64(99), res.Course.Order)
		repoMock.AssertExpectations(t)
	})

	t.Run("error_no_fields_provided", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

		req := &pb.UpdateCourseRequest{
			Id: courseID.String(),
		}

		res, err := srv.UpdateCourse(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "no fields to update")
	})

	t.Run("error_course_not_found", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		newTitle := "Updated Title"
		updates := course.CourseUpdates{Title: &newTitle}

		repoMock.On("UpdateCourse", mock.Anything, courseID, updates).Return(course.Course{}, course.ErrNotFound)

		req := &pb.UpdateCourseRequest{
			Id:    courseID.String(),
			Title: &newTitle,
		}

		res, err := srv.UpdateCourse(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, res)
		repoMock.AssertExpectations(t)
	})

	t.Run("error_invalid_course_id", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		newTitle := "Updated Title"
		req := &pb.UpdateCourseRequest{
			Id:    "invalid-uuid",
			Title: &newTitle,
		}

		res, err := srv.UpdateCourse(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("error_repository_failure", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		newTitle := "Updated Title"
		updates := course.CourseUpdates{Title: &newTitle}

		repoMock.On("UpdateCourse", mock.Anything, courseID, updates).Return(course.Course{}, assert.AnError)

		req := &pb.UpdateCourseRequest{
			Id:    courseID.String(),
			Title: &newTitle,
		}

		res, err := srv.UpdateCourse(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, res)
		repoMock.AssertExpectations(t)
	})
}

func TestServer_UpdateLesson(t *testing.T) {
	t.Run("success_update_title", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		newTitle := "Updated Lesson Title"
		updates := course.LessonUpdates{Title: &newTitle}

		updatedLesson := course.Lesson{
			ID:          lessonID,
			CourseID:    courseID,
			Order:       1,
			Title:       newTitle,
			Description: "Original description",
			Body:        "![deck](11111111-1111-1111-1111-111111111111)",
			CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		repoMock.On("UpdateLesson", mock.Anything, lessonID, updates).Return(updatedLesson, nil)

		req := &pb.UpdateLessonRequest{
			Id:    lessonID.String(),
			Title: &newTitle,
		}

		res, err := srv.UpdateLesson(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, newTitle, res.Lesson.Title)
		assert.Equal(t, "Original description", res.Lesson.Description)
		repoMock.AssertExpectations(t)
	})

	t.Run("success_update_description", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		newDesc := "Updated Lesson Description"
		updates := course.LessonUpdates{Description: &newDesc}

		updatedLesson := course.Lesson{
			ID:          lessonID,
			CourseID:    courseID,
			Order:       1,
			Title:       "Original Title",
			Description: newDesc,
			Body:        "![deck](11111111-1111-1111-1111-111111111111)",
			CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		repoMock.On("UpdateLesson", mock.Anything, lessonID, updates).Return(updatedLesson, nil)

		req := &pb.UpdateLessonRequest{
			Id:          lessonID.String(),
			Description: &newDesc,
		}

		res, err := srv.UpdateLesson(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, "Original Title", res.Lesson.Title)
		assert.Equal(t, newDesc, res.Lesson.Description)
		repoMock.AssertExpectations(t)
	})

	t.Run("success_update_order", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		newOrder := int64(42)
		updates := course.LessonUpdates{Order: &newOrder}

		updatedLesson := course.Lesson{
			ID:          lessonID,
			CourseID:    courseID,
			Order:       42,
			Title:       "Original Title",
			Description: "Original description",
			Body:        "![deck](11111111-1111-1111-1111-111111111111)",
			CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		repoMock.On("UpdateLesson", mock.Anything, lessonID, updates).Return(updatedLesson, nil)

		req := &pb.UpdateLessonRequest{
			Id:    lessonID.String(),
			Order: &newOrder,
		}

		res, err := srv.UpdateLesson(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, int64(42), res.Lesson.Order)
		repoMock.AssertExpectations(t)
	})

	t.Run("success_update_body_with_valid_deck", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		deckClientMock := &DeckClientMock{}
		srv := &Server{Repository: repoMock, DeckClient: deckClientMock}

		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		deckID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
		newBody := "Updated content ![deck](" + deckID.String() + ")"
		updates := course.LessonUpdates{Body: &newBody}

		deckClientMock.On("GetDeck", mock.Anything, &pbDeck.GetDeckRequest{DeckId: deckID.String()}).
			Return(&pbDeck.GetDeckResponse{}, nil)

		updatedLesson := course.Lesson{
			ID:          lessonID,
			CourseID:    courseID,
			Order:       1,
			Title:       "Original Title",
			Description: "Original description",
			Body:        newBody,
			CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		repoMock.On("UpdateLesson", mock.Anything, lessonID, updates).Return(updatedLesson, nil)

		req := &pb.UpdateLessonRequest{
			Id:   lessonID.String(),
			Body: &newBody,
		}

		res, err := srv.UpdateLesson(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, newBody, res.Lesson.Body)
		repoMock.AssertExpectations(t)
		deckClientMock.AssertExpectations(t)
	})

	t.Run("error_update_body_without_deck_reference", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		newBody := "Content without any deck reference"

		req := &pb.UpdateLessonRequest{
			Id:   lessonID.String(),
			Body: &newBody,
		}

		res, err := srv.UpdateLesson(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "lesson must reference at least one deck")
	})

	t.Run("error_update_body_with_invalid_deck", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		deckClientMock := &DeckClientMock{}
		srv := &Server{Repository: repoMock, DeckClient: deckClientMock}

		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		deckID := uuid.MustParse("99999999-9999-9999-9999-999999999999")
		newBody := "Content ![deck](" + deckID.String() + ")"

		deckClientMock.On("GetDeck", mock.Anything, &pbDeck.GetDeckRequest{DeckId: deckID.String()}).
			Return(nil, assert.AnError)

		req := &pb.UpdateLessonRequest{
			Id:   lessonID.String(),
			Body: &newBody,
		}

		res, err := srv.UpdateLesson(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "does not exist")
		deckClientMock.AssertExpectations(t)
	})

	t.Run("success_update_multiple_fields", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		deckClientMock := &DeckClientMock{}
		srv := &Server{Repository: repoMock, DeckClient: deckClientMock}

		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		deckID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
		newTitle := "New Title"
		newDesc := "New Description"
		newOrder := int64(99)
		newBody := "New body ![deck](" + deckID.String() + ")"
		updates := course.LessonUpdates{Title: &newTitle, Description: &newDesc, Order: &newOrder, Body: &newBody}

		deckClientMock.On("GetDeck", mock.Anything, &pbDeck.GetDeckRequest{DeckId: deckID.String()}).
			Return(&pbDeck.GetDeckResponse{}, nil)

		updatedLesson := course.Lesson{
			ID:          lessonID,
			CourseID:    courseID,
			Order:       99,
			Title:       newTitle,
			Description: newDesc,
			Body:        newBody,
			CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		repoMock.On("UpdateLesson", mock.Anything, lessonID, updates).Return(updatedLesson, nil)

		req := &pb.UpdateLessonRequest{
			Id:          lessonID.String(),
			Title:       &newTitle,
			Description: &newDesc,
			Order:       &newOrder,
			Body:        &newBody,
		}

		res, err := srv.UpdateLesson(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, newTitle, res.Lesson.Title)
		assert.Equal(t, newDesc, res.Lesson.Description)
		assert.Equal(t, int64(99), res.Lesson.Order)
		assert.Equal(t, newBody, res.Lesson.Body)
		repoMock.AssertExpectations(t)
		deckClientMock.AssertExpectations(t)
	})

	t.Run("error_no_fields_provided", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")

		req := &pb.UpdateLessonRequest{
			Id: lessonID.String(),
		}

		res, err := srv.UpdateLesson(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "no fields to update")
	})

	t.Run("error_lesson_not_found", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		newTitle := "Updated Title"
		updates := course.LessonUpdates{Title: &newTitle}

		repoMock.On("UpdateLesson", mock.Anything, lessonID, updates).Return(course.Lesson{}, course.ErrNotFound)

		req := &pb.UpdateLessonRequest{
			Id:    lessonID.String(),
			Title: &newTitle,
		}

		res, err := srv.UpdateLesson(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, res)
		repoMock.AssertExpectations(t)
	})

	t.Run("error_invalid_lesson_id", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		newTitle := "Updated Title"
		req := &pb.UpdateLessonRequest{
			Id:    "invalid-uuid",
			Title: &newTitle,
		}

		res, err := srv.UpdateLesson(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("error_repository_failure", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		newTitle := "Updated Title"
		updates := course.LessonUpdates{Title: &newTitle}

		repoMock.On("UpdateLesson", mock.Anything, lessonID, updates).Return(course.Lesson{}, assert.AnError)

		req := &pb.UpdateLessonRequest{
			Id:    lessonID.String(),
			Title: &newTitle,
		}

		res, err := srv.UpdateLesson(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, res)
		repoMock.AssertExpectations(t)
	})
}

// DeckClientMock implements pbDeck.DecksAPIClient for testing
type DeckClientMock struct {
	mock.Mock
}

func (m *DeckClientMock) GetDeck(ctx context.Context, in *pbDeck.GetDeckRequest, opts ...grpc.CallOption) (*pbDeck.GetDeckResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDeck.GetDeckResponse), args.Error(1)
}

func (m *DeckClientMock) GetDecks(ctx context.Context, in *pbDeck.GetDecksRequest, opts ...grpc.CallOption) (*pbDeck.GetDecksResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*pbDeck.GetDecksResponse), args.Error(1)
}

func (m *DeckClientMock) CreateDeck(ctx context.Context, in *pbDeck.CreateDeckRequest, opts ...grpc.CallOption) (*pbDeck.CreateDeckResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*pbDeck.CreateDeckResponse), args.Error(1)
}

func (m *DeckClientMock) DeleteDeck(ctx context.Context, in *pbDeck.DeleteDeckRequest, opts ...grpc.CallOption) (*pbDeck.DeleteDeckResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*pbDeck.DeleteDeckResponse), args.Error(1)
}

func (m *DeckClientMock) GetPopularDecks(ctx context.Context, in *pbDeck.GetPopularDecksRequest, opts ...grpc.CallOption) (*pbDeck.GetPopularDecksResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*pbDeck.GetPopularDecksResponse), args.Error(1)
}

func (m *DeckClientMock) CreateCard(ctx context.Context, in *pbDeck.CreateCardRequest, opts ...grpc.CallOption) (*pbDeck.CreateCardResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*pbDeck.CreateCardResponse), args.Error(1)
}

func (m *DeckClientMock) GetCards(ctx context.Context, in *pbDeck.GetCardsRequest, opts ...grpc.CallOption) (*pbDeck.GetCardsResponse, error) {
	args := m.Called(ctx, in)
	return args.Get(0).(*pbDeck.GetCardsResponse), args.Error(1)
}
