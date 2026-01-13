package course

import (
	"context"

	"github.com/XaviFP/toshokan/common/pagination"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type RepositoryMock struct {
	mock.Mock
}

func (m *RepositoryMock) GetCourse(ctx context.Context, id uuid.UUID) (Course, error) {
	args := m.Called(ctx, id)
	return args[0].(Course), args.Error(1)
}

func (m *RepositoryMock) StoreCourse(ctx context.Context, course Course) error {
	return m.Called(ctx, course).Error(0)
}

func (m *RepositoryMock) GetEnrolledCourses(ctx context.Context, userID uuid.UUID, p pagination.Pagination) (CoursesWithProgressConnection, error) {
	args := m.Called(ctx, userID, p)
	return args.Get(0).(CoursesWithProgressConnection), args.Error(1)
}

func (m *RepositoryMock) GetLesson(ctx context.Context, id uuid.UUID) (Lesson, error) {
	args := m.Called(ctx, id)
	return args[0].(Lesson), args.Error(1)
}

func (m *RepositoryMock) GetLessonsByCourseID(ctx context.Context, courseID uuid.UUID, p pagination.Pagination) (LessonsConnection, error) {
	args := m.Called(ctx, courseID, p)
	return args[0].(LessonsConnection), args.Error(1)
}

func (m *RepositoryMock) GetFocusedLessons(ctx context.Context, courseID uuid.UUID, currentLessonID uuid.UUID, contextSize int) ([]Lesson, error) {
	args := m.Called(ctx, courseID, currentLessonID, contextSize)
	return args[0].([]Lesson), args.Error(1)
}

func (m *RepositoryMock) StoreLesson(ctx context.Context, lesson Lesson) error {
	return m.Called(ctx, lesson).Error(0)
}

func (m *RepositoryMock) GetUserCourseProgress(ctx context.Context, userID uuid.UUID, courseID uuid.UUID) (UserCourseProgress, error) {
	args := m.Called(ctx, userID, courseID)
	return args[0].(UserCourseProgress), args.Error(1)
}

func (m *RepositoryMock) EnrollUserInCourse(ctx context.Context, userID uuid.UUID, courseID uuid.UUID, progressState ProgressState) error {
	return m.Called(ctx, userID, courseID, progressState).Error(0)
}

func (m *RepositoryMock) UpdateUserProgress(ctx context.Context, progress UserCourseProgress) error {
	return m.Called(ctx, progress).Error(0)
}

type EnrollerMock struct {
	mock.Mock
}

func (m *EnrollerMock) Enroll(ctx context.Context, userID, courseID uuid.UUID) (UserCourseProgress, error) {
	args := m.Called(ctx, userID, courseID)
	return args[0].(UserCourseProgress), args.Error(1)
}

type LessonsBrowserMock struct {
	mock.Mock
}

func (m *LessonsBrowserMock) Browse(ctx context.Context, courseID uuid.UUID, p pagination.Pagination, opts BrowseOptions) (BrowseResult, error) {
	args := m.Called(ctx, courseID, p, opts)
	return args[0].(BrowseResult), args.Error(1)
}

type CoursesBrowserMock struct {
	mock.Mock
}

func (m *CoursesBrowserMock) BrowseEnrolled(ctx context.Context, userID uuid.UUID, p pagination.Pagination) (CoursesWithProgressConnection, error) {
	args := m.Called(ctx, userID, p)
	return args.Get(0).(CoursesWithProgressConnection), args.Error(1)
}

type StateSyncerMock struct {
	mock.Mock
}

func (m *StateSyncerMock) Sync(ctx context.Context, userID uuid.UUID, courseID uuid.UUID) error {
	args := m.Called(ctx, userID, courseID)
	return args.Error(0)
}
