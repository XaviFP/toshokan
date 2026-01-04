package course

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/juju/errors"
	"github.com/mediocregopher/radix/v4"
	"github.com/stretchr/testify/assert"
)

func TestRedisRepository_GetCourse_CacheHit(t *testing.T) {
	h := newTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	courseID := uuid.New()
	expectedCourse := Course{
		ID:          courseID,
		Title:       "Go Fundamentals",
		Description: "Learn Go",
		CreatedAt:   time.Now().UTC(),
	}

	// Pre-populate cache
	courseJSON, _ := json.Marshal(expectedCourse)
	key := "course:" + courseID.String()
	err := h.redisClient.Do(ctx, radix.FlatCmd(nil, "SETEX", key, 3600, string(courseJSON)))
	assert.NoError(t, err)

	result, err := repo.GetCourse(ctx, courseID)

	assert.NoError(t, err)
	assert.Equal(t, expectedCourse.ID, result.ID)
	assert.Equal(t, expectedCourse.Title, result.Title)
	mockDB.AssertNotCalled(t, "GetCourse")
}

func TestRedisRepository_GetCourse_CacheMiss(t *testing.T) {
	h := newTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	courseID := uuid.New()
	expectedCourse := Course{
		ID:          courseID,
		Title:       "Go Fundamentals",
		Description: "Learn Go",
		CreatedAt:   time.Now().UTC(),
	}

	mockDB.On("GetCourse", ctx, courseID).Return(expectedCourse, nil)

	result, err := repo.GetCourse(ctx, courseID)

	assert.NoError(t, err)
	assert.Equal(t, expectedCourse.ID, result.ID)
	assert.Equal(t, expectedCourse.Title, result.Title)

	// Verify it was cached
	var cached string
	key := "course:" + courseID.String()
	mb := radix.Maybe{Rcv: &cached}
	err = h.redisClient.Do(ctx, radix.Cmd(&mb, "GET", key))
	assert.NoError(t, err)
	assert.False(t, mb.Null)
	assert.NotEmpty(t, cached)

	var cachedCourse Course
	err = json.Unmarshal([]byte(cached), &cachedCourse)
	assert.NoError(t, err)
	assert.Equal(t, expectedCourse.ID, cachedCourse.ID)

	mockDB.AssertExpectations(t)
}

func TestRedisRepository_GetCourse_DBError(t *testing.T) {
	h := newTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	courseID := uuid.New()

	mockDB.On("GetCourse", ctx, courseID).Return(Course{}, errors.New("db error"))

	result, err := repo.GetCourse(ctx, courseID)

	assert.Error(t, err)
	assert.Empty(t, result.ID)
	mockDB.AssertExpectations(t)
}

func TestRedisRepository_StoreCourse(t *testing.T) {
	h := newTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	course := Course{
		ID:          uuid.New(),
		Title:       "Go Fundamentals",
		Description: "Learn Go",
		CreatedAt:   time.Now().UTC(),
	}

	// Pre-populate cache to verify invalidation
	courseJSON, _ := json.Marshal(course)
	key := "course:" + course.ID.String()
	err := h.redisClient.Do(ctx, radix.FlatCmd(nil, "SETEX", key, 3600, string(courseJSON)))
	assert.NoError(t, err)

	mockDB.On("StoreCourse", ctx, course).Return(nil)

	err = repo.StoreCourse(ctx, course)

	assert.NoError(t, err)

	// Verify cache was invalidated
	var cached string
	mb := radix.Maybe{Rcv: &cached}
	err = h.redisClient.Do(ctx, radix.Cmd(&mb, "GET", key))
	assert.NoError(t, err)
	assert.True(t, mb.Null, "Cache should be invalidated")

	mockDB.AssertExpectations(t)
}

func TestRedisRepository_StoreCourse_DBError(t *testing.T) {
	h := newTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	course := Course{
		ID:          uuid.New(),
		Title:       "Go Fundamentals",
		Description: "Learn Go",
		CreatedAt:   time.Now().UTC(),
	}

	mockDB.On("StoreCourse", ctx, course).Return(errors.New("db error"))

	err := repo.StoreCourse(ctx, course)

	assert.Error(t, err)
	mockDB.AssertExpectations(t)
}

func TestRedisRepository_GetLesson_CacheHit(t *testing.T) {
	h := newTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	lessonID := uuid.New()
	expectedLesson := Lesson{
		ID:       lessonID,
		CourseID: uuid.New(),
		Title:    "Introduction",
		Body:     "Content",
		Order:    1,
	}

	// Pre-populate cache
	lessonJSON, _ := json.Marshal(expectedLesson)
	key := "lesson:" + lessonID.String()
	err := h.redisClient.Do(ctx, radix.FlatCmd(nil, "SETEX", key, 3600, string(lessonJSON)))
	assert.NoError(t, err)

	result, err := repo.GetLesson(ctx, lessonID)

	assert.NoError(t, err)
	assert.Equal(t, expectedLesson.ID, result.ID)
	assert.Equal(t, expectedLesson.Title, result.Title)
	mockDB.AssertNotCalled(t, "GetLesson")
}

func TestRedisRepository_GetLesson_CacheMiss(t *testing.T) {
	h := newTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	lessonID := uuid.New()
	expectedLesson := Lesson{
		ID:       lessonID,
		CourseID: uuid.New(),
		Title:    "Introduction",
		Body:     "Content",
		Order:    1,
	}

	mockDB.On("GetLesson", ctx, lessonID).Return(expectedLesson, nil)

	result, err := repo.GetLesson(ctx, lessonID)

	assert.NoError(t, err)
	assert.Equal(t, expectedLesson.ID, result.ID)
	assert.Equal(t, expectedLesson.Title, result.Title)

	// Verify it was cached
	var cached string
	key := "lesson:" + lessonID.String()
	mb := radix.Maybe{Rcv: &cached}
	err = h.redisClient.Do(ctx, radix.Cmd(&mb, "GET", key))
	assert.NoError(t, err)
	assert.False(t, mb.Null)
	assert.NotEmpty(t, cached)

	mockDB.AssertExpectations(t)
}

func TestRedisRepository_StoreLesson(t *testing.T) {
	h := newTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	lesson := Lesson{
		ID:       uuid.New(),
		CourseID: uuid.New(),
		Title:    "Introduction",
		Body:     "Content",
		Order:    1,
	}

	// Pre-populate cache to verify invalidation
	lessonJSON, _ := json.Marshal(lesson)
	key := "lesson:" + lesson.ID.String()
	err := h.redisClient.Do(ctx, radix.FlatCmd(nil, "SETEX", key, 3600, string(lessonJSON)))
	assert.NoError(t, err)

	mockDB.On("StoreLesson", ctx, lesson).Return(nil)

	err = repo.StoreLesson(ctx, lesson)

	assert.NoError(t, err)

	// Verify cache was invalidated
	var cached string
	mb := radix.Maybe{Rcv: &cached}
	err = h.redisClient.Do(ctx, radix.Cmd(&mb, "GET", key))
	assert.NoError(t, err)
	assert.True(t, mb.Null, "Cache should be invalidated")

	mockDB.AssertExpectations(t)
}

func TestRedisRepository_GetUserCourseProgress_CacheHit(t *testing.T) {
	h := newTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	expectedProgress := UserCourseProgress{
		ID:              uuid.New(),
		UserID:          userID,
		CourseID:        courseID,
		CurrentLessonID: uuid.New(),
		State:           newTestProgressState(),
	}

	// Pre-populate cache
	progressJSON, _ := json.Marshal(expectedProgress)
	key := fmt.Sprintf("user_progress:%s:%s", userID, courseID)
	err := h.redisClient.Do(ctx, radix.FlatCmd(nil, "SETEX", key, 1800, string(progressJSON)))
	assert.NoError(t, err)

	result, err := repo.GetUserCourseProgress(ctx, userID, courseID)

	assert.NoError(t, err)
	assert.Equal(t, expectedProgress.ID, result.ID)
	assert.Equal(t, expectedProgress.UserID, result.UserID)
	mockDB.AssertNotCalled(t, "GetUserCourseProgress")
}

func TestRedisRepository_GetUserCourseProgress_CacheMiss(t *testing.T) {
	h := newTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	expectedProgress := UserCourseProgress{
		ID:              uuid.New(),
		UserID:          userID,
		CourseID:        courseID,
		CurrentLessonID: uuid.New(),
		State:           newTestProgressState(),
	}

	mockDB.On("GetUserCourseProgress", ctx, userID, courseID).Return(expectedProgress, nil)

	result, err := repo.GetUserCourseProgress(ctx, userID, courseID)

	assert.NoError(t, err)
	assert.Equal(t, expectedProgress.ID, result.ID)
	assert.Equal(t, expectedProgress.UserID, result.UserID)

	// Verify it was cached
	var cached string
	key := fmt.Sprintf("user_progress:%s:%s", userID, courseID)
	mb := radix.Maybe{Rcv: &cached}
	err = h.redisClient.Do(ctx, radix.Cmd(&mb, "GET", key))
	assert.NoError(t, err)
	assert.False(t, mb.Null)
	assert.NotEmpty(t, cached)

	mockDB.AssertExpectations(t)
}

func TestRedisRepository_EnrollUserInCourse(t *testing.T) {
	h := newTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	state := *newTestProgressState()

	// Pre-populate cache to verify invalidation
	progressJSON, _ := json.Marshal(UserCourseProgress{UserID: userID, CourseID: courseID})
	key := fmt.Sprintf("user_progress:%s:%s", userID, courseID)
	err := h.redisClient.Do(ctx, radix.FlatCmd(nil, "SETEX", key, 1800, string(progressJSON)))
	assert.NoError(t, err)

	mockDB.On("EnrollUserInCourse", ctx, userID, courseID, state).Return(nil)

	err = repo.EnrollUserInCourse(ctx, userID, courseID, state)

	assert.NoError(t, err)

	// Verify cache was invalidated
	var cached string
	mb := radix.Maybe{Rcv: &cached}
	err = h.redisClient.Do(ctx, radix.Cmd(&mb, "GET", key))
	assert.NoError(t, err)
	assert.True(t, mb.Null, "Cache should be invalidated")

	mockDB.AssertExpectations(t)
}

func TestRedisRepository_EnrollUserInCourse_DBError(t *testing.T) {
	h := newTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	userID := uuid.New()
	courseID := uuid.New()
	state := *newTestProgressState()

	mockDB.On("EnrollUserInCourse", ctx, userID, courseID, state).Return(errors.New("db error"))

	err := repo.EnrollUserInCourse(ctx, userID, courseID, state)

	assert.Error(t, err)
	mockDB.AssertExpectations(t)
}

func TestRedisRepository_UpdateUserProgress(t *testing.T) {
	h := newTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	progress := UserCourseProgress{
		ID:              uuid.New(),
		UserID:          uuid.New(),
		CourseID:        uuid.New(),
		CurrentLessonID: uuid.New(),
		State:           newTestProgressState(),
	}

	// Pre-populate cache to verify invalidation
	progressJSON, _ := json.Marshal(progress)
	key := fmt.Sprintf("user_progress:%s:%s", progress.UserID, progress.CourseID)
	err := h.redisClient.Do(ctx, radix.FlatCmd(nil, "SETEX", key, 1800, string(progressJSON)))
	assert.NoError(t, err)

	mockDB.On("UpdateUserProgress", ctx, progress).Return(nil)

	err = repo.UpdateUserProgress(ctx, progress)

	assert.NoError(t, err)

	// Verify cache was invalidated
	var cached string
	mb := radix.Maybe{Rcv: &cached}
	err = h.redisClient.Do(ctx, radix.Cmd(&mb, "GET", key))
	assert.NoError(t, err)
	assert.True(t, mb.Null, "Cache should be invalidated")

	mockDB.AssertExpectations(t)
}

func TestRedisRepository_UpdateUserProgress_DBError(t *testing.T) {
	h := newTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	progress := UserCourseProgress{
		ID:              uuid.New(),
		UserID:          uuid.New(),
		CourseID:        uuid.New(),
		CurrentLessonID: uuid.New(),
		State:           newTestProgressState(),
	}

	mockDB.On("UpdateUserProgress", ctx, progress).Return(errors.New("db error"))

	err := repo.UpdateUserProgress(ctx, progress)

	assert.Error(t, err)
	mockDB.AssertExpectations(t)
}
