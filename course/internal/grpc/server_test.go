package grpc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

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

func TestServer_GetCourseDeckIds(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		deck1ID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		deck2ID := uuid.MustParse("1f30a72f-5d7a-48da-a5c2-42efece6972a")
		repoMock.On("GetCourse", mock.Anything, courseID).Return(course.Course{ID: courseID}, nil)
		repoMock.On("GetCourseDeckIDs", mock.Anything, courseID).Return([]uuid.UUID{deck1ID, deck2ID}, nil)

		res, err := srv.GetCourseDeckIds(context.Background(), &pb.GetCourseDeckIdsRequest{CourseId: courseID.String()})
		assert.NoError(t, err)
		assert.Equal(t, []string{deck1ID.String(), deck2ID.String()}, res.DeckIds)
	})

	t.Run("success_no_decks", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		repoMock.On("GetCourse", mock.Anything, courseID).Return(course.Course{ID: courseID}, nil)
		repoMock.On("GetCourseDeckIDs", mock.Anything, courseID).Return([]uuid.UUID{}, nil)

		res, err := srv.GetCourseDeckIds(context.Background(), &pb.GetCourseDeckIdsRequest{CourseId: courseID.String()})
		assert.NoError(t, err)
		assert.Empty(t, res.DeckIds)
	})

	t.Run("failure_invalid_course_id", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		res, err := srv.GetCourseDeckIds(context.Background(), &pb.GetCourseDeckIdsRequest{CourseId: "invalid-uuid"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Nil(t, res)
	})

	t.Run("failure_course_not_found", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		repoMock.On("GetCourse", mock.Anything, courseID).Return(course.Course{}, errors.Trace(course.ErrCourseNotFound))

		res, err := srv.GetCourseDeckIds(context.Background(), &pb.GetCourseDeckIdsRequest{CourseId: courseID.String()})
		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Nil(t, res)
		repoMock.AssertNotCalled(t, "GetCourseDeckIDs", mock.Anything, mock.Anything)
	})

	t.Run("failure_repository_error", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		repoMock.On("GetCourse", mock.Anything, courseID).Return(course.Course{ID: courseID}, nil)
		repoMock.On("GetCourseDeckIDs", mock.Anything, courseID).Return([]uuid.UUID(nil), assert.AnError)

		res, err := srv.GetCourseDeckIds(context.Background(), &pb.GetCourseDeckIdsRequest{CourseId: courseID.String()})
		assert.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, res)
	})
}

func TestServer_GetLessonDeckIds(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		deck1ID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		deck2ID := uuid.MustParse("1f30a72f-5d7a-48da-a5c2-42efece6972a")
		repoMock.On("GetLesson", mock.Anything, lessonID).Return(course.Lesson{ID: lessonID}, nil)
		repoMock.On("GetLessonDeckIDs", mock.Anything, lessonID).Return([]uuid.UUID{deck1ID, deck2ID}, nil)

		res, err := srv.GetLessonDeckIds(context.Background(), &pb.GetLessonDeckIdsRequest{LessonId: lessonID.String()})
		assert.NoError(t, err)
		assert.Equal(t, []string{deck1ID.String(), deck2ID.String()}, res.DeckIds)
	})

	t.Run("success_no_decks", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		repoMock.On("GetLesson", mock.Anything, lessonID).Return(course.Lesson{ID: lessonID}, nil)
		repoMock.On("GetLessonDeckIDs", mock.Anything, lessonID).Return([]uuid.UUID{}, nil)

		res, err := srv.GetLessonDeckIds(context.Background(), &pb.GetLessonDeckIdsRequest{LessonId: lessonID.String()})
		assert.NoError(t, err)
		assert.Empty(t, res.DeckIds)
	})

	t.Run("failure_invalid_lesson_id", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		res, err := srv.GetLessonDeckIds(context.Background(), &pb.GetLessonDeckIdsRequest{LessonId: "invalid-uuid"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Nil(t, res)
	})

	t.Run("failure_lesson_not_found", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		repoMock.On("GetLesson", mock.Anything, lessonID).Return(course.Lesson{}, errors.Trace(course.ErrLessonNotFound))

		res, err := srv.GetLessonDeckIds(context.Background(), &pb.GetLessonDeckIdsRequest{LessonId: lessonID.String()})
		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Nil(t, res)
		repoMock.AssertNotCalled(t, "GetLessonDeckIDs", mock.Anything, mock.Anything)
	})

	t.Run("failure_repository_error", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		repoMock.On("GetLesson", mock.Anything, lessonID).Return(course.Lesson{ID: lessonID}, nil)
		repoMock.On("GetLessonDeckIDs", mock.Anything, lessonID).Return([]uuid.UUID(nil), assert.AnError)

		res, err := srv.GetLessonDeckIds(context.Background(), &pb.GetLessonDeckIdsRequest{LessonId: lessonID.String()})
		assert.ErrorIs(t, err, assert.AnError)
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
	courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
	userID := uuid.MustParse("4acc3aa4-7684-4a64-b61c-28a599881922")
	createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("success_returns_lessons_with_progress_without_body", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		browserMock := &course.LessonsBrowserMock{}
		srv := &Server{Repository: repoMock, LessonsBrowser: browserMock}
		repoMock.On("GetCourse", mock.Anything, courseID).Return(course.Course{ID: courseID}, nil)

		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		browserMock.On("Browse", mock.Anything, courseID, pagination.NewOldestFirstPagination(pagination.WithFirst(10)), course.BrowseOptions{
			UserID:   &userID,
			Bodyless: true,
		}).Return(course.BrowseResult{
			ProgressLessons: &course.LessonsWithProgressConnection{
				Edges: []course.LessonWithProgressEdge{
					{
						Lesson: &course.LessonWithProgress{
							Lesson: course.Lesson{
								ID:          lessonID,
								CourseID:    courseID,
								Order:       1,
								Title:       "Introduction",
								Description: "Getting started",
								CreatedAt:   createdAt,
							},
							IsCompleted: true,
							IsCurrent:   false,
						},
						Cursor: "cursor1",
					},
				},
				PageInfo: pagination.PageInfo{
					HasNextPage: true,
					StartCursor: "cursor1",
					EndCursor:   "cursor1",
				},
			},
		}, nil)

		actual, err := srv.GetLessons(context.Background(), &pb.GetLessonsRequest{
			CourseId:   courseID.String(),
			UserId:     userID.String(),
			Pagination: &pb.Pagination{First: 10},
		})

		require.NoError(t, err)

		expected := &pb.GetLessonsResponse{
			Lessons: &pb.LessonsWithProgressConnection{
				Edges: []*pb.LessonsWithProgressConnection_Edge{
					{
						Node: &pb.LessonWithProgress{
							Lesson: &pb.Lesson{
								Id:          lessonID.String(),
								CourseId:    courseID.String(),
								Order:       1,
								Title:       "Introduction",
								Description: "Getting started",
								CreatedAt:   timestamppb.New(createdAt),
							},
							IsCompleted: true,
							IsCurrent:   false,
						},
						Cursor: "cursor1",
					},
				},
				PageInfo: &pb.PageInfo{
					HasNextPage: true,
					StartCursor: "cursor1",
					EndCursor:   "cursor1",
				},
			},
		}
		assert.Equal(t, expected, actual)
		browserMock.AssertExpectations(t)
	})

	t.Run("success_without_pagination", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		browserMock := &course.LessonsBrowserMock{}
		srv := &Server{Repository: repoMock, LessonsBrowser: browserMock}
		repoMock.On("GetCourse", mock.Anything, courseID).Return(course.Course{ID: courseID}, nil)

		browserMock.On("Browse", mock.Anything, courseID, pagination.NewOldestFirstPagination(), course.BrowseOptions{
			UserID:   &userID,
			Bodyless: true,
		}).Return(course.BrowseResult{
			ProgressLessons: &course.LessonsWithProgressConnection{},
		}, nil)

		actual, err := srv.GetLessons(context.Background(), &pb.GetLessonsRequest{
			CourseId: courseID.String(),
			UserId:   userID.String(),
		})

		require.NoError(t, err)

		expected := &pb.GetLessonsResponse{
			Lessons: &pb.LessonsWithProgressConnection{
				PageInfo: &pb.PageInfo{},
			},
		}
		assert.Equal(t, expected, actual)
		browserMock.AssertExpectations(t)
	})

	t.Run("invalid_course_id", func(t *testing.T) {
		browserMock := &course.LessonsBrowserMock{}
		srv := &Server{LessonsBrowser: browserMock}

		actual, err := srv.GetLessons(context.Background(), &pb.GetLessonsRequest{
			CourseId:   "not-a-uuid",
			UserId:     userID.String(),
			Pagination: &pb.Pagination{First: 10},
		})

		require.Error(t, err)
		assert.Nil(t, actual)
		browserMock.AssertNotCalled(t, "Browse")
	})

	t.Run("invalid_user_id", func(t *testing.T) {
		browserMock := &course.LessonsBrowserMock{}
		srv := &Server{LessonsBrowser: browserMock}

		actual, err := srv.GetLessons(context.Background(), &pb.GetLessonsRequest{
			CourseId:   courseID.String(),
			UserId:     "not-a-uuid",
			Pagination: &pb.Pagination{First: 10},
		})

		require.Error(t, err)
		assert.Nil(t, actual)
		browserMock.AssertNotCalled(t, "Browse")
	})

	t.Run("failure_course_not_found", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		browserMock := &course.LessonsBrowserMock{}
		srv := &Server{Repository: repoMock, LessonsBrowser: browserMock}
		repoMock.On("GetCourse", mock.Anything, courseID).Return(course.Course{}, errors.Trace(course.ErrCourseNotFound))

		actual, err := srv.GetLessons(context.Background(), &pb.GetLessonsRequest{
			CourseId:   courseID.String(),
			UserId:     userID.String(),
			Pagination: &pb.Pagination{First: 10},
		})

		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Nil(t, actual)
		browserMock.AssertNotCalled(t, "Browse")
	})

	t.Run("failure_get_course_error", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		browserMock := &course.LessonsBrowserMock{}
		srv := &Server{Repository: repoMock, LessonsBrowser: browserMock}
		repoMock.On("GetCourse", mock.Anything, courseID).Return(course.Course{}, assert.AnError)

		actual, err := srv.GetLessons(context.Background(), &pb.GetLessonsRequest{
			CourseId:   courseID.String(),
			UserId:     userID.String(),
			Pagination: &pb.Pagination{First: 10},
		})

		assert.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, actual)
		browserMock.AssertNotCalled(t, "Browse")
	})

	t.Run("browse_error", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		browserMock := &course.LessonsBrowserMock{}
		srv := &Server{Repository: repoMock, LessonsBrowser: browserMock}
		repoMock.On("GetCourse", mock.Anything, courseID).Return(course.Course{ID: courseID}, nil)

		browserMock.On("Browse", mock.Anything, courseID, pagination.NewOldestFirstPagination(pagination.WithFirst(10)), course.BrowseOptions{
			UserID:   &userID,
			Bodyless: true,
		}).Return(course.BrowseResult{}, errors.Trace(course.ErrUserProgressNotFound))

		actual, err := srv.GetLessons(context.Background(), &pb.GetLessonsRequest{
			CourseId:   courseID.String(),
			UserId:     userID.String(),
			Pagination: &pb.Pagination{First: 10},
		})

		require.ErrorIs(t, err, course.ErrUserProgressNotFound)
		assert.Nil(t, actual)
	})
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

func TestServer_CreateLesson(t *testing.T) {
	courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
	deckID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	body := "Content ![deck](" + deckID.String() + ")"

	t.Run("success", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		deckClientMock := &DeckClientMock{}
		srv := &Server{Repository: repoMock, DeckClient: deckClientMock}
		repoMock.On("GetCourse", mock.Anything, courseID).Return(course.Course{ID: courseID}, nil)
		deckClientMock.On("GetDeck", mock.Anything, &pbDeck.GetDeckRequest{DeckId: deckID.String()}).Return(&pbDeck.GetDeckResponse{}, nil)
		repoMock.On("StoreLesson", mock.Anything, mock.Anything).Return(nil)

		actual, err := srv.CreateLesson(context.Background(), &pb.CreateLessonRequest{
			CourseId:    courseID.String(),
			Order:       1,
			Title:       "Introduction",
			Description: "Getting started",
			Body:        body,
		})

		require.NoError(t, err)

		expected := &pb.CreateLessonResponse{
			Lesson: &pb.Lesson{
				Id:          actual.Lesson.Id,
				CourseId:    courseID.String(),
				Order:       1,
				Title:       "Introduction",
				Description: "Getting started",
				Body:        body,
				CreatedAt:   actual.Lesson.CreatedAt,
			},
		}
		assert.Equal(t, expected, actual)
		repoMock.AssertExpectations(t)
	})

	t.Run("failure_course_not_found", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		deckClientMock := &DeckClientMock{}
		srv := &Server{Repository: repoMock, DeckClient: deckClientMock}
		repoMock.On("GetCourse", mock.Anything, courseID).Return(course.Course{}, errors.Trace(course.ErrCourseNotFound))

		actual, err := srv.CreateLesson(context.Background(), &pb.CreateLessonRequest{
			CourseId:    courseID.String(),
			Order:       1,
			Title:       "Introduction",
			Description: "Getting started",
			Body:        body,
		})

		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Nil(t, actual)
		repoMock.AssertNotCalled(t, "StoreLesson", mock.Anything, mock.Anything)
	})

	t.Run("failure_get_course_error", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		deckClientMock := &DeckClientMock{}
		srv := &Server{Repository: repoMock, DeckClient: deckClientMock}
		repoMock.On("GetCourse", mock.Anything, courseID).Return(course.Course{}, assert.AnError)

		actual, err := srv.CreateLesson(context.Background(), &pb.CreateLessonRequest{
			CourseId:    courseID.String(),
			Order:       1,
			Title:       "Introduction",
			Description: "Getting started",
			Body:        body,
		})

		assert.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, actual)
		repoMock.AssertNotCalled(t, "StoreLesson", mock.Anything, mock.Anything)
	})
}

func TestServer_DeleteCourse(t *testing.T) {
	courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

	t.Run("success", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}
		repoMock.On("DeleteCourse", mock.Anything, courseID).Return([]uuid.UUID{uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")}, nil)

		actual, err := srv.DeleteCourse(context.Background(), &pb.DeleteCourseRequest{CourseId: courseID.String()})

		require.NoError(t, err)
		assert.Equal(t, &pb.DeleteCourseResponse{}, actual)
		repoMock.AssertExpectations(t)
	})

	t.Run("failure_invalid_course_id", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		actual, err := srv.DeleteCourse(context.Background(), &pb.DeleteCourseRequest{CourseId: "invalid"})

		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Nil(t, actual)
		repoMock.AssertNotCalled(t, "DeleteCourse", mock.Anything, mock.Anything)
	})

	t.Run("failure_not_found", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}
		repoMock.On("DeleteCourse", mock.Anything, courseID).Return([]uuid.UUID(nil), errors.Trace(course.ErrCourseNotFound))

		actual, err := srv.DeleteCourse(context.Background(), &pb.DeleteCourseRequest{CourseId: courseID.String()})

		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Nil(t, actual)
	})

	t.Run("failure_repository_error", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}
		repoMock.On("DeleteCourse", mock.Anything, courseID).Return([]uuid.UUID(nil), assert.AnError)

		actual, err := srv.DeleteCourse(context.Background(), &pb.DeleteCourseRequest{CourseId: courseID.String()})

		assert.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, actual)
	})
}

func TestServer_DeleteLesson(t *testing.T) {
	courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
	lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")

	t.Run("success", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}
		repoMock.On("DeleteLesson", mock.Anything, courseID, lessonID).Return(nil)

		actual, err := srv.DeleteLesson(context.Background(), &pb.DeleteLessonRequest{CourseId: courseID.String(), LessonId: lessonID.String()})

		require.NoError(t, err)
		assert.Equal(t, &pb.DeleteLessonResponse{}, actual)
		repoMock.AssertExpectations(t)
	})

	t.Run("failure_invalid_course_id", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		actual, err := srv.DeleteLesson(context.Background(), &pb.DeleteLessonRequest{CourseId: "invalid", LessonId: lessonID.String()})

		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Nil(t, actual)
		repoMock.AssertNotCalled(t, "DeleteLesson", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure_invalid_lesson_id", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		actual, err := srv.DeleteLesson(context.Background(), &pb.DeleteLessonRequest{CourseId: courseID.String(), LessonId: "invalid"})

		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Nil(t, actual)
		repoMock.AssertNotCalled(t, "DeleteLesson", mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("failure_not_found", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}
		repoMock.On("DeleteLesson", mock.Anything, courseID, lessonID).Return(errors.Trace(course.ErrLessonNotFound))

		actual, err := srv.DeleteLesson(context.Background(), &pb.DeleteLessonRequest{CourseId: courseID.String(), LessonId: lessonID.String()})

		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Nil(t, actual)
	})

	t.Run("failure_repository_error", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}
		repoMock.On("DeleteLesson", mock.Anything, courseID, lessonID).Return(assert.AnError)

		actual, err := srv.DeleteLesson(context.Background(), &pb.DeleteLessonRequest{CourseId: courseID.String(), LessonId: lessonID.String()})

		assert.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, actual)
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

func (m *DeckClientMock) UpdateDeck(ctx context.Context, in *pbDeck.UpdateDeckRequest, opts ...grpc.CallOption) (*pbDeck.UpdateDeckResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDeck.UpdateDeckResponse), args.Error(1)
}

func (m *DeckClientMock) UpdateCard(ctx context.Context, in *pbDeck.UpdateCardRequest, opts ...grpc.CallOption) (*pbDeck.UpdateCardResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDeck.UpdateCardResponse), args.Error(1)
}

func (m *DeckClientMock) CreateAnswer(ctx context.Context, in *pbDeck.CreateAnswerRequest, opts ...grpc.CallOption) (*pbDeck.CreateAnswerResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDeck.CreateAnswerResponse), args.Error(1)
}

func (m *DeckClientMock) DeleteAnswer(ctx context.Context, in *pbDeck.DeleteAnswerRequest, opts ...grpc.CallOption) (*pbDeck.DeleteAnswerResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDeck.DeleteAnswerResponse), args.Error(1)
}

func (m *DeckClientMock) DeleteCard(ctx context.Context, in *pbDeck.DeleteCardRequest, opts ...grpc.CallOption) (*pbDeck.DeleteCardResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDeck.DeleteCardResponse), args.Error(1)
}

func (m *DeckClientMock) UpdateAnswer(ctx context.Context, in *pbDeck.UpdateAnswerRequest, opts ...grpc.CallOption) (*pbDeck.UpdateAnswerResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pbDeck.UpdateAnswerResponse), args.Error(1)
}

// ============================================
// Backoffice Endpoint Tests
// ============================================

func TestServer_GetStudentsProgress(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		userID1 := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
		userID2 := uuid.MustParse("5f48b711-d3ae-5e1a-ba55-77f3de9dd2da")
		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")

		studentsConn := course.StudentsProgressConnection{
			Edges: []course.StudentProgressEdge{
				{
					Student: course.StudentProgressSummary{
						UserID:                userID1,
						CurrentLessonID:       lessonID,
						LessonsCompleted:      3,
						TotalLessons:          5,
						TotalCorrectAnswers:   15,
						TotalIncorrectAnswers: 2,
						EnrolledAt:            time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
						LastActivityAt:        time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
					},
					Cursor: "cursor1",
				},
				{
					Student: course.StudentProgressSummary{
						UserID:                userID2,
						CurrentLessonID:       lessonID,
						LessonsCompleted:      1,
						TotalLessons:          5,
						TotalCorrectAnswers:   5,
						TotalIncorrectAnswers: 3,
						EnrolledAt:            time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
						LastActivityAt:        time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
					},
					Cursor: "cursor2",
				},
			},
			PageInfo: pagination.PageInfo{
				HasNextPage:     true,
				HasPreviousPage: false,
				StartCursor:     "cursor1",
				EndCursor:       "cursor2",
			},
		}

		repoMock.On("GetEnrolledUsersProgress", mock.Anything, courseID, mock.Anything).Return(studentsConn, nil)
		repoMock.On("CountEnrolledUsers", mock.Anything, courseID).Return(50, nil)

		req := &pb.GetStudentsProgressRequest{
			CourseId:   courseID.String(),
			Pagination: &pb.Pagination{First: 20},
			Detailed:   false,
		}

		res, err := srv.GetStudentsProgress(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Len(t, res.Students.Edges, 2)
		assert.Equal(t, int32(50), res.TotalEnrolled)
		assert.Equal(t, userID1.String(), res.Students.Edges[0].Node.UserId)
		assert.Equal(t, int32(3), res.Students.Edges[0].Node.LessonsCompleted)
		assert.Equal(t, int32(15), res.Students.Edges[0].Node.TotalCorrectAnswers)
		assert.True(t, res.Students.PageInfo.HasNextPage)
		assert.False(t, res.Students.PageInfo.HasPreviousPage)
	})

	t.Run("success_with_detailed", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		completedAt := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)

		studentsConn := course.StudentsProgressConnection{
			Edges: []course.StudentProgressEdge{
				{
					Student: course.StudentProgressSummary{
						UserID:           userID,
						CurrentLessonID:  lessonID,
						LessonsCompleted: 1,
						TotalLessons:     5,
						EnrolledAt:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
						LastActivityAt:   time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
						LessonDetails: []course.LessonProgressDetail{
							{
								LessonID:         lessonID,
								IsCompleted:      true,
								CompletedAt:      &completedAt,
								CorrectAnswers:   10,
								IncorrectAnswers: 2,
							},
						},
					},
					Cursor: "cursor1",
				},
			},
		}

		repoMock.On("GetEnrolledUsersProgress", mock.Anything, courseID, mock.Anything).Return(studentsConn, nil)
		repoMock.On("CountEnrolledUsers", mock.Anything, courseID).Return(1, nil)

		req := &pb.GetStudentsProgressRequest{
			CourseId:   courseID.String(),
			Pagination: &pb.Pagination{First: 20},
			Detailed:   true,
		}

		res, err := srv.GetStudentsProgress(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Len(t, res.Students.Edges, 1)
		assert.Len(t, res.Students.Edges[0].Node.LessonDetails, 1)
		assert.Equal(t, lessonID.String(), res.Students.Edges[0].Node.LessonDetails[0].LessonId)
		assert.True(t, res.Students.Edges[0].Node.LessonDetails[0].IsCompleted)
		assert.Equal(t, int32(10), res.Students.Edges[0].Node.LessonDetails[0].CorrectAnswers)
	})

	t.Run("failure_invalid_course_id", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		req := &pb.GetStudentsProgressRequest{
			CourseId:   "invalid-uuid",
			Pagination: &pb.Pagination{First: 20},
		}

		res, err := srv.GetStudentsProgress(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("failure_repo_error", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		repoMock.On("GetEnrolledUsersProgress", mock.Anything, courseID, mock.Anything).Return(course.StudentsProgressConnection{}, assert.AnError)

		req := &pb.GetStudentsProgressRequest{
			CourseId:   courseID.String(),
			Pagination: &pb.Pagination{First: 20},
		}

		res, err := srv.GetStudentsProgress(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("success_with_nil_pagination", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")

		studentsConn := course.StudentsProgressConnection{
			Edges: []course.StudentProgressEdge{
				{
					Student: course.StudentProgressSummary{
						UserID:           userID,
						CurrentLessonID:  lessonID,
						LessonsCompleted: 2,
						TotalLessons:     5,
						EnrolledAt:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
						LastActivityAt:   time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
					},
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

		repoMock.On("GetEnrolledUsersProgress", mock.Anything, courseID, mock.MatchedBy(func(p pagination.Pagination) bool {
			// Verify default pagination is applied: first=20
			return p.First == 20
		})).Return(studentsConn, nil)
		repoMock.On("CountEnrolledUsers", mock.Anything, courseID).Return(1, nil)

		req := &pb.GetStudentsProgressRequest{
			CourseId:   courseID.String(),
			Pagination: nil, // nil pagination should default to first=20
		}

		res, err := srv.GetStudentsProgress(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Len(t, res.Students.Edges, 1)
		assert.Equal(t, int32(1), res.TotalEnrolled)
	})
}

func TestServer_GetUserState(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		progressID := uuid.MustParse("8a5bc922-3def-4567-8901-2345abcdef67")

		progressState := course.NewProgressState()
		progressState.Lessons["lesson1"] = &course.LessonProgress{
			IsCompleted: true,
			Decks:       make(map[string]*course.DeckProgress),
		}

		progress := course.UserCourseProgress{
			ID:              progressID,
			UserID:          userID,
			CourseID:        courseID,
			CurrentLessonID: lessonID,
			State:           progressState,
			CreatedAt:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:       time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		}

		repoMock.On("GetUserCourseProgress", mock.Anything, userID, courseID).Return(progress, nil)

		req := &pb.GetUserStateRequest{
			CourseId: courseID.String(),
			UserId:   userID.String(),
		}

		res, err := srv.GetUserState(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, progressID.String(), res.UserState.Id)
		assert.Equal(t, userID.String(), res.UserState.UserId)
		assert.Equal(t, courseID.String(), res.UserState.CourseId)
		assert.Equal(t, lessonID.String(), res.UserState.CurrentLessonId)
		assert.Contains(t, res.UserState.StateJson, "lesson1")
	})

	t.Run("failure_invalid_user_id", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		req := &pb.GetUserStateRequest{
			CourseId: "fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72",
			UserId:   "invalid-uuid",
		}

		res, err := srv.GetUserState(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("failure_invalid_course_id", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		req := &pb.GetUserStateRequest{
			CourseId: "invalid-uuid",
			UserId:   "4e37a600-c29e-4d0f-af44-66f2cd8cc1c9",
		}

		res, err := srv.GetUserState(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("failure_user_not_enrolled", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

		repoMock.On("GetUserCourseProgress", mock.Anything, userID, courseID).Return(course.UserCourseProgress{}, course.ErrUserProgressNotFound)

		req := &pb.GetUserStateRequest{
			CourseId: courseID.String(),
			UserId:   userID.String(),
		}

		res, err := srv.GetUserState(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}

func TestServer_SetUserState(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		newLessonID := uuid.MustParse("445ecf09-2bdd-516c-9012-3456cdefab78")
		progressID := uuid.MustParse("8a5bc922-3def-4567-8901-2345abcdef67")

		existingProgress := course.UserCourseProgress{
			ID:              progressID,
			UserID:          userID,
			CourseID:        courseID,
			CurrentLessonID: lessonID,
			State:           course.NewProgressState(),
			CreatedAt:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:       time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		}

		repoMock.On("GetUserCourseProgress", mock.Anything, userID, courseID).Return(existingProgress, nil)
		repoMock.On("UpdateUserProgress", mock.Anything, mock.MatchedBy(func(p course.UserCourseProgress) bool {
			return p.UserID == userID && p.CourseID == courseID && p.CurrentLessonID == newLessonID
		})).Return(nil)

		newStateJSON := `{"lessons":{"lesson1":{"decks":{},"is_completed":true}}}`
		req := &pb.SetUserStateRequest{
			CourseId:        courseID.String(),
			UserId:          userID.String(),
			CurrentLessonId: newLessonID.String(),
			StateJson:       newStateJSON,
			SyncAfter:       false,
		}

		res, err := srv.SetUserState(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, userID.String(), res.UserState.UserId)
		assert.Equal(t, newLessonID.String(), res.UserState.CurrentLessonId)
	})

	t.Run("success_with_sync", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		syncerMock := &course.StateSyncerMock{}
		srv := &Server{Repository: repoMock, StateSyncer: syncerMock}

		userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		progressID := uuid.MustParse("8a5bc922-3def-4567-8901-2345abcdef67")

		existingProgress := course.UserCourseProgress{
			ID:       progressID,
			UserID:   userID,
			CourseID: courseID,
			State:    course.NewProgressState(),
		}

		repoMock.On("GetUserCourseProgress", mock.Anything, userID, courseID).Return(existingProgress, nil)
		repoMock.On("UpdateUserProgress", mock.Anything, mock.Anything).Return(nil)
		syncerMock.On("Sync", mock.Anything, userID, courseID).Return(nil)

		newStateJSON := `{"lessons":{}}`
		req := &pb.SetUserStateRequest{
			CourseId:  courseID.String(),
			UserId:    userID.String(),
			StateJson: newStateJSON,
			SyncAfter: true,
		}

		res, err := srv.SetUserState(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		syncerMock.AssertCalled(t, "Sync", mock.Anything, userID, courseID)
	})

	t.Run("failure_invalid_state_json", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

		existingProgress := course.UserCourseProgress{
			ID:       uuid.New(),
			UserID:   userID,
			CourseID: courseID,
			State:    course.NewProgressState(),
		}

		repoMock.On("GetUserCourseProgress", mock.Anything, userID, courseID).Return(existingProgress, nil)

		req := &pb.SetUserStateRequest{
			CourseId:  courseID.String(),
			UserId:    userID.String(),
			StateJson: "invalid json {{{",
		}

		res, err := srv.SetUserState(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, res)
		assert.Contains(t, err.Error(), "invalid state_json")
	})

	t.Run("failure_invalid_current_lesson_id", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

		existingProgress := course.UserCourseProgress{
			ID:       uuid.New(),
			UserID:   userID,
			CourseID: courseID,
			State:    course.NewProgressState(),
		}

		repoMock.On("GetUserCourseProgress", mock.Anything, userID, courseID).Return(existingProgress, nil)

		req := &pb.SetUserStateRequest{
			CourseId:        courseID.String(),
			UserId:          userID.String(),
			CurrentLessonId: "invalid-uuid",
			StateJson:       `{"lessons":{}}`,
		}

		res, err := srv.SetUserState(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("failure_user_not_enrolled", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
		courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

		repoMock.On("GetUserCourseProgress", mock.Anything, userID, courseID).Return(course.UserCourseProgress{}, course.ErrUserProgressNotFound)

		req := &pb.SetUserStateRequest{
			CourseId:  courseID.String(),
			UserId:    userID.String(),
			StateJson: `{"lessons":{}}`,
		}

		res, err := srv.SetUserState(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}

func TestServer_GetAllCourses(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		courses := course.CoursesConnection{
			Edges: []course.CourseEdge{
				{
					Course: course.Course{
						ID:          uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72"),
						Order:       1,
						Title:       "Go Mastery",
						Description: "Master the Go programming language",
						CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					},
					Cursor: "cursor1",
				},
				{
					Course: course.Course{
						ID:          uuid.MustParse("1f30a72f-5d7a-48da-a5c2-42efece6972a"),
						Order:       2,
						Title:       "Advanced Go",
						Description: "Advanced Go concepts",
						CreatedAt:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
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

		repoMock.On("GetAllCourses", mock.Anything, mock.Anything).Return(courses, nil)
		repoMock.On("CountCourses", mock.Anything).Return(2, nil)

		req := &pb.GetAllCoursesRequest{
			Pagination: &pb.Pagination{First: 20},
		}

		res, err := srv.GetAllCourses(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Len(t, res.Courses.Edges, 2)
		assert.Equal(t, int32(2), res.TotalCount)
		assert.Equal(t, "Go Mastery", res.Courses.Edges[0].Node.Title)
		assert.Equal(t, "Advanced Go", res.Courses.Edges[1].Node.Title)
	})

	t.Run("success_empty", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		repoMock.On("GetAllCourses", mock.Anything, mock.Anything).Return(course.CoursesConnection{}, nil)
		repoMock.On("CountCourses", mock.Anything).Return(0, nil)

		req := &pb.GetAllCoursesRequest{
			Pagination: &pb.Pagination{First: 20},
		}

		res, err := srv.GetAllCourses(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Empty(t, res.Courses.Edges)
		assert.Equal(t, int32(0), res.TotalCount)
	})

	t.Run("success_with_nil_pagination", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		repoMock.On("GetAllCourses", mock.Anything, mock.Anything).Return(course.CoursesConnection{}, nil)
		repoMock.On("CountCourses", mock.Anything).Return(0, nil)

		req := &pb.GetAllCoursesRequest{
			Pagination: nil, // Should use default pagination
		}

		res, err := srv.GetAllCourses(context.Background(), req)
		assert.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("failure_get_courses_error", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		repoMock.On("GetAllCourses", mock.Anything, mock.Anything).Return(course.CoursesConnection{}, assert.AnError)

		req := &pb.GetAllCoursesRequest{
			Pagination: &pb.Pagination{First: 20},
		}

		res, err := srv.GetAllCourses(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("failure_count_courses_error", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}

		repoMock.On("GetAllCourses", mock.Anything, mock.Anything).Return(course.CoursesConnection{}, nil)
		repoMock.On("CountCourses", mock.Anything).Return(0, assert.AnError)

		req := &pb.GetAllCoursesRequest{
			Pagination: &pb.Pagination{First: 20},
		}

		res, err := srv.GetAllCourses(context.Background(), req)
		assert.Error(t, err)
		assert.Nil(t, res)
	})
}

func TestServer_AdminGetLessons(t *testing.T) {
	courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
	createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("success_returns_lessons_with_body_without_progress", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		browserMock := &course.LessonsBrowserMock{}
		srv := &Server{Repository: repoMock, LessonsBrowser: browserMock}

		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		repoMock.On("GetCourse", mock.Anything, courseID).Return(course.Course{ID: courseID}, nil)
		browserMock.On("Browse", mock.Anything, courseID, pagination.NewOldestFirstPagination(pagination.WithFirst(10)), course.BrowseOptions{}).Return(course.BrowseResult{
			PublicLessons: &course.LessonsConnection{
				Edges: []course.LessonEdge{
					{
						Lesson: course.Lesson{
							ID:          lessonID,
							CourseID:    courseID,
							Order:       1,
							Title:       "Introduction",
							Description: "Getting started",
							Body:        "Lesson body",
							CreatedAt:   createdAt,
						},
						Cursor: "cursor1",
					},
				},
				PageInfo: pagination.PageInfo{
					HasNextPage: true,
					StartCursor: "cursor1",
					EndCursor:   "cursor1",
				},
			},
		}, nil)

		actual, err := srv.AdminGetLessons(context.Background(), &pb.AdminGetLessonsRequest{
			CourseId:   courseID.String(),
			Pagination: &pb.Pagination{First: 10},
		})

		require.NoError(t, err)

		expected := &pb.AdminGetLessonsResponse{
			Lessons: &pb.LessonsConnection{
				Edges: []*pb.LessonsConnection_Edge{
					{
						Node: &pb.Lesson{
							Id:          lessonID.String(),
							CourseId:    courseID.String(),
							Order:       1,
							Title:       "Introduction",
							Description: "Getting started",
							Body:        "Lesson body",
							CreatedAt:   timestamppb.New(createdAt),
						},
						Cursor: "cursor1",
					},
				},
				PageInfo: &pb.PageInfo{
					HasNextPage: true,
					StartCursor: "cursor1",
					EndCursor:   "cursor1",
				},
			},
		}
		assert.Equal(t, expected, actual)
		browserMock.AssertExpectations(t)
	})

	t.Run("success_nil_pagination_uses_defaults", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		browserMock := &course.LessonsBrowserMock{}
		srv := &Server{Repository: repoMock, LessonsBrowser: browserMock}

		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		repoMock.On("GetCourse", mock.Anything, courseID).Return(course.Course{ID: courseID}, nil)
		browserMock.On("Browse", mock.Anything, courseID, pagination.NewOldestFirstPagination(), course.BrowseOptions{}).Return(course.BrowseResult{
			PublicLessons: &course.LessonsConnection{
				Edges: []course.LessonEdge{
					{
						Lesson: course.Lesson{
							ID:          lessonID,
							CourseID:    courseID,
							Order:       1,
							Title:       "Introduction",
							Description: "Getting started",
							Body:        "Lesson body",
							CreatedAt:   createdAt,
						},
						Cursor: "cursor1",
					},
				},
				PageInfo: pagination.PageInfo{
					StartCursor: "cursor1",
					EndCursor:   "cursor1",
				},
			},
		}, nil)

		actual, err := srv.AdminGetLessons(context.Background(), &pb.AdminGetLessonsRequest{
			CourseId: courseID.String(),
		})

		require.NoError(t, err)

		expected := &pb.AdminGetLessonsResponse{
			Lessons: &pb.LessonsConnection{
				Edges: []*pb.LessonsConnection_Edge{
					{
						Node: &pb.Lesson{
							Id:          lessonID.String(),
							CourseId:    courseID.String(),
							Order:       1,
							Title:       "Introduction",
							Description: "Getting started",
							Body:        "Lesson body",
							CreatedAt:   timestamppb.New(createdAt),
						},
						Cursor: "cursor1",
					},
				},
				PageInfo: &pb.PageInfo{
					StartCursor: "cursor1",
					EndCursor:   "cursor1",
				},
			},
		}
		assert.Equal(t, expected, actual)
		browserMock.AssertExpectations(t)
	})

	t.Run("failure_invalid_course_id", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		browserMock := &course.LessonsBrowserMock{}
		srv := &Server{Repository: repoMock, LessonsBrowser: browserMock}

		actual, err := srv.AdminGetLessons(context.Background(), &pb.AdminGetLessonsRequest{
			CourseId:   "not-a-uuid",
			Pagination: &pb.Pagination{First: 10},
		})

		assert.Equal(t, codes.InvalidArgument, status.Code(err))
		assert.Nil(t, actual)
		repoMock.AssertNotCalled(t, "GetCourse", mock.Anything, mock.Anything)
		browserMock.AssertNotCalled(t, "Browse")
	})

	t.Run("failure_course_not_found", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		browserMock := &course.LessonsBrowserMock{}
		srv := &Server{Repository: repoMock, LessonsBrowser: browserMock}

		repoMock.On("GetCourse", mock.Anything, courseID).Return(course.Course{}, errors.Trace(course.ErrCourseNotFound))

		actual, err := srv.AdminGetLessons(context.Background(), &pb.AdminGetLessonsRequest{
			CourseId:   courseID.String(),
			Pagination: &pb.Pagination{First: 10},
		})

		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Nil(t, actual)
		browserMock.AssertNotCalled(t, "Browse")
	})

	t.Run("failure_get_course_error", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		browserMock := &course.LessonsBrowserMock{}
		srv := &Server{Repository: repoMock, LessonsBrowser: browserMock}

		repoMock.On("GetCourse", mock.Anything, courseID).Return(course.Course{}, assert.AnError)

		actual, err := srv.AdminGetLessons(context.Background(), &pb.AdminGetLessonsRequest{
			CourseId:   courseID.String(),
			Pagination: &pb.Pagination{First: 10},
		})

		assert.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, actual)
		browserMock.AssertNotCalled(t, "Browse")
	})

	t.Run("failure_browse_error", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		browserMock := &course.LessonsBrowserMock{}
		srv := &Server{Repository: repoMock, LessonsBrowser: browserMock}

		repoMock.On("GetCourse", mock.Anything, courseID).Return(course.Course{ID: courseID}, nil)
		browserMock.On("Browse", mock.Anything, courseID, pagination.NewOldestFirstPagination(pagination.WithFirst(10)), course.BrowseOptions{}).Return(course.BrowseResult{}, assert.AnError)

		actual, err := srv.AdminGetLessons(context.Background(), &pb.AdminGetLessonsRequest{
			CourseId:   courseID.String(),
			Pagination: &pb.Pagination{First: 10},
		})

		assert.ErrorIs(t, err, assert.AnError)
		assert.Nil(t, actual)
	})
}

func TestInvalidCursorInterceptor(t *testing.T) {
	t.Run("invalid_cursor", func(t *testing.T) {
		repoMock := &course.RepositoryMock{}
		srv := &Server{Repository: repoMock}
		repoMock.On("GetAllCourses", mock.Anything, mock.Anything).
			Return(course.CoursesConnection{}, errors.Trace(pagination.FromCursor("Ây", &course.CourseCursor{})))

		handler := func(ctx context.Context, req any) (any, error) {
			return srv.GetAllCourses(ctx, req.(*pb.GetAllCoursesRequest))
		}

		res, err := invalidCursorInterceptor(context.Background(), &pb.GetAllCoursesRequest{}, nil, handler)
		assert.Nil(t, res)
		assert.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("other_error", func(t *testing.T) {
		handler := func(ctx context.Context, req any) (any, error) {
			return nil, assert.AnError
		}

		res, err := invalidCursorInterceptor(context.Background(), &pb.GetAllCoursesRequest{}, nil, handler)
		assert.Nil(t, res)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("success", func(t *testing.T) {
		expected := &pb.GetAllCoursesResponse{TotalCount: 1}
		handler := func(ctx context.Context, req any) (any, error) {
			return expected, nil
		}

		actual, err := invalidCursorInterceptor(context.Background(), &pb.GetAllCoursesRequest{}, nil, handler)
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}
