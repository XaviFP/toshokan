package course

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
	"github.com/mediocregopher/radix/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/XaviFP/toshokan/common/config"
	"github.com/XaviFP/toshokan/common/db"
	"github.com/XaviFP/toshokan/common/pagination"
)

func TestRepository_StoreCourse(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(slog.Default(), h.db)

	t.Run("success", func(t *testing.T) {
		id := uuid.MustParse("ebcfffa0-a96f-450b-a0f3-a2e47263855d")
		course := Course{
			ID:          id,
			Title:       "Go Mastery",
			Description: "Master the Go programming language",
			CreatedAt:   time.Now(),
		}
		err := repo.StoreCourse(context.Background(), course)
		assert.NoError(t, err)

		var out Course
		row := h.db.QueryRow(`SELECT id, title, description FROM courses WHERE id = $1 AND deleted_at IS NULL`, id)
		err = row.Scan(&out.ID, &out.Title, &out.Description)
		assert.NoError(t, err)
		assert.Equal(t, course.ID, out.ID)
		assert.Equal(t, course.Title, out.Title)
		assert.Equal(t, course.Description, out.Description)

		// update if exists
		course.Title = "Go Mastery Updated"
		err = repo.StoreCourse(context.Background(), course)
		assert.NoError(t, err)

		row = h.db.QueryRow(`SELECT id, title, description FROM courses WHERE id = $1 AND deleted_at IS NULL`, id)
		err = row.Scan(&out.ID, &out.Title, &out.Description)
		assert.NoError(t, err)
		assert.Equal(t, course.ID, out.ID)
		assert.Equal(t, "Go Mastery Updated", out.Title)
	})

}

func TestRepository_GetCourse(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(slog.Default(), h.db)

	t.Run("success", func(t *testing.T) {
		id := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

		course, err := repo.GetCourse(context.Background(), id)
		assert.NoError(t, err)
		assert.Equal(t, "Go Fundamentals", course.Title)
		assert.Equal(t, "Learn the basics of Go", course.Description)
	})

	t.Run("failure_not_found", func(t *testing.T) {
		id := uuid.MustParse("00000000-0000-0000-0000-000000000000")

		course, err := repo.GetCourse(context.Background(), id)
		assert.Error(t, err)
		assert.Empty(t, course.ID)
	})
}

func TestRepository_StoreLesson(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(slog.Default(), h.db)

	courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

	t.Run("success", func(t *testing.T) {
		lesson := Lesson{
			ID:          uuid.MustParse("bc8a13b3-c257-497f-9e80-02e9a50a2fbe"),
			CourseID:    courseID,
			Order:       10,
			Title:       "Advanced Patterns",
			Description: "Learn advanced Go patterns",
			Body:        "Content with ![deck](60766223-ff9f-4871-a497-f765c05a0c5e)",
			CreatedAt:   time.Now(),
		}

		err := repo.StoreLesson(context.Background(), lesson)
		assert.NoError(t, err)

		var out Lesson
		row := h.db.QueryRow(`SELECT id, course_id, "order", title, description, body FROM lessons WHERE id = $1 AND deleted_at IS NULL`, lesson.ID)
		err = row.Scan(&out.ID, &out.CourseID, &out.Order, &out.Title, &out.Description, &out.Body)
		assert.NoError(t, err)
		assert.Equal(t, lesson.ID, out.ID)
		assert.Equal(t, lesson.Title, out.Title)

		// Check lesson_decks after insert
		var deckCount int
		err = h.db.QueryRow(`SELECT COUNT(*) FROM lesson_decks WHERE lesson_id = $1`, lesson.ID).Scan(&deckCount)
		assert.NoError(t, err)
		assert.Equal(t, 1, deckCount)

		// update if exists (change title, keep deck)
		lesson.Title = "Advanced Patterns Updated"
		err = repo.StoreLesson(context.Background(), lesson)
		assert.NoError(t, err)

		row = h.db.QueryRow(`SELECT id, course_id, "order", title, description, body FROM lessons WHERE id = $1 AND deleted_at IS NULL`, lesson.ID)
		err = row.Scan(&out.ID, &out.CourseID, &out.Order, &out.Title, &out.Description, &out.Body)
		assert.NoError(t, err)
		assert.Equal(t, lesson.ID, out.ID)
		assert.Equal(t, "Advanced Patterns Updated", out.Title)

		// update with new deck reference
		lesson.Body = "Now with two decks: ![deck](60766223-ff9f-4871-a497-f765c05a0c5e) and ![deck](11111111-1111-1111-1111-111111111111)"
		err = repo.StoreLesson(context.Background(), lesson)
		assert.NoError(t, err)
		err = h.db.QueryRow(`SELECT COUNT(*) FROM lesson_decks WHERE lesson_id = $1`, lesson.ID).Scan(&deckCount)
		assert.NoError(t, err)
		assert.Equal(t, 2, deckCount)

		// update with only one deck (remove one)
		lesson.Body = "Just one deck: ![deck](11111111-1111-1111-1111-111111111111)"
		err = repo.StoreLesson(context.Background(), lesson)
		assert.NoError(t, err)
		err = h.db.QueryRow(`SELECT COUNT(*) FROM lesson_decks WHERE lesson_id = $1`, lesson.ID).Scan(&deckCount)
		assert.NoError(t, err)
		assert.Equal(t, 1, deckCount)

		// update with no decks
		// Lessons without decks aren't allowed, so this scenario is not likely in practice,
		// but we test it to ensure the code handles it gracefully.
		lesson.Body = "No decks here!"
		err = repo.StoreLesson(context.Background(), lesson)
		assert.NoError(t, err)
		err = h.db.QueryRow(`SELECT COUNT(*) FROM lesson_decks WHERE lesson_id = $1`, lesson.ID).Scan(&deckCount)
		assert.NoError(t, err)
		assert.Equal(t, 0, deckCount)
	})
}

func TestRepository_UpdateCourse(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(slog.Default(), h.db)

	t.Run("update_title_only", func(t *testing.T) {
		id := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		newTitle := "Updated Go Fundamentals"

		course, err := repo.UpdateCourse(context.Background(), id, CourseUpdates{
			Title: &newTitle,
		})
		require.NoError(t, err)
		assert.Equal(t, newTitle, course.Title)
		assert.Equal(t, "Learn the basics of Go", course.Description) // unchanged
		assert.NotNil(t, course.UpdatedAt)
	})

	t.Run("update_description_only", func(t *testing.T) {
		id := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		newDesc := "A comprehensive guide to Go basics"

		course, err := repo.UpdateCourse(context.Background(), id, CourseUpdates{
			Description: &newDesc,
		})
		require.NoError(t, err)
		assert.Equal(t, newDesc, course.Description)
	})

	t.Run("update_order", func(t *testing.T) {
		id := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		newOrder := int64(99)

		course, err := repo.UpdateCourse(context.Background(), id, CourseUpdates{
			Order: &newOrder,
		})
		require.NoError(t, err)
		assert.Equal(t, newOrder, course.Order)
	})

	t.Run("update_multiple_fields", func(t *testing.T) {
		id := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		newTitle := "Go Programming 101"
		newDesc := "Start your Go journey"
		newOrder := int64(1)

		course, err := repo.UpdateCourse(context.Background(), id, CourseUpdates{
			Title:       &newTitle,
			Description: &newDesc,
			Order:       &newOrder,
		})
		require.NoError(t, err)
		assert.Equal(t, newTitle, course.Title)
		assert.Equal(t, newDesc, course.Description)
		assert.Equal(t, newOrder, course.Order)
	})

	t.Run("update_with_empty_string", func(t *testing.T) {
		id := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")
		emptyTitle := ""

		course, err := repo.UpdateCourse(context.Background(), id, CourseUpdates{
			Title: &emptyTitle,
		})
		require.NoError(t, err)
		assert.Equal(t, "", course.Title)
	})

	t.Run("not_found", func(t *testing.T) {
		id := uuid.MustParse("00000000-0000-0000-0000-000000000000")
		newTitle := "Does not exist"

		_, err := repo.UpdateCourse(context.Background(), id, CourseUpdates{
			Title: &newTitle,
		})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotFound))
	})
}

func TestRepository_UpdateLesson(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(slog.Default(), h.db)

	t.Run("update_title_only", func(t *testing.T) {
		id := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		newTitle := "Updated Goroutines Tutorial"

		lesson, err := repo.UpdateLesson(context.Background(), id, LessonUpdates{
			Title: &newTitle,
		})
		require.NoError(t, err)
		assert.Equal(t, newTitle, lesson.Title)
		assert.Equal(t, "Learn about concurrent programming", lesson.Description) // unchanged
		assert.NotNil(t, lesson.UpdatedAt)
	})

	t.Run("update_body", func(t *testing.T) {
		id := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		newBody := "New body content with ![deck](60766223-ff9f-4871-a497-f765c05a0c5e)"

		lesson, err := repo.UpdateLesson(context.Background(), id, LessonUpdates{
			Body: &newBody,
		})
		require.NoError(t, err)
		assert.Equal(t, newBody, lesson.Body)

		// Check lesson_decks updated
		var deckCount int
		err = h.db.QueryRow(`SELECT COUNT(*) FROM lesson_decks WHERE lesson_id = $1`, id).Scan(&deckCount)
		assert.NoError(t, err)
		assert.Equal(t, 1, deckCount)

		// Add a second deck
		newBody2 := "Two decks: ![deck](60766223-ff9f-4871-a497-f765c05a0c5e) ![deck](11111111-1111-1111-1111-111111111111)"
		lesson, err = repo.UpdateLesson(context.Background(), id, LessonUpdates{
			Body: &newBody2,
		})
		require.NoError(t, err)
		assert.Equal(t, newBody2, lesson.Body)
		err = h.db.QueryRow(`SELECT COUNT(*) FROM lesson_decks WHERE lesson_id = $1`, id).Scan(&deckCount)
		assert.NoError(t, err)
		assert.Equal(t, 2, deckCount)

		// Remove all decks
		newBody3 := "No decks now!"
		lesson, err = repo.UpdateLesson(context.Background(), id, LessonUpdates{
			Body: &newBody3,
		})
		require.NoError(t, err)
		assert.Equal(t, newBody3, lesson.Body)
		err = h.db.QueryRow(`SELECT COUNT(*) FROM lesson_decks WHERE lesson_id = $1`, id).Scan(&deckCount)
		assert.NoError(t, err)
		assert.Equal(t, 0, deckCount)
	})

	t.Run("update_order", func(t *testing.T) {
		id := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		newOrder := int64(50)

		lesson, err := repo.UpdateLesson(context.Background(), id, LessonUpdates{
			Order: &newOrder,
		})
		require.NoError(t, err)
		assert.Equal(t, int(newOrder), lesson.Order)
	})

	t.Run("update_multiple_fields", func(t *testing.T) {
		id := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
		newTitle := "Concurrency Masterclass"
		newDesc := "Deep dive into Go concurrency"
		newBody := "Updated content ![deck](60766223-ff9f-4871-a497-f765c05a0c5e)"
		newOrder := int64(5)

		lesson, err := repo.UpdateLesson(context.Background(), id, LessonUpdates{
			Title:       &newTitle,
			Description: &newDesc,
			Body:        &newBody,
			Order:       &newOrder,
		})
		require.NoError(t, err)
		assert.Equal(t, newTitle, lesson.Title)
		assert.Equal(t, newDesc, lesson.Description)
		assert.Equal(t, newBody, lesson.Body)
		assert.Equal(t, int(newOrder), lesson.Order)
	})

	t.Run("not_found", func(t *testing.T) {
		id := uuid.MustParse("00000000-0000-0000-0000-000000000000")
		newTitle := "Does not exist"

		_, err := repo.UpdateLesson(context.Background(), id, LessonUpdates{
			Title: &newTitle,
		})
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrNotFound))
	})
}

func TestRepository_GetLesson(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(slog.Default(), h.db)

	t.Run("success", func(t *testing.T) {
		id := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")

		lesson, err := repo.GetLesson(context.Background(), id)
		assert.NoError(t, err)
		assert.Equal(t, "Introduction to Goroutines", lesson.Title)
		assert.Equal(t, "Learn about concurrent programming", lesson.Description)
	})

	t.Run("failure_not_found", func(t *testing.T) {
		id := uuid.MustParse("00000000-0000-0000-0000-000000000000")

		lesson, err := repo.GetLesson(context.Background(), id)
		assert.Error(t, err)
		assert.Empty(t, lesson.ID)
	})
}

func TestRepository_DeleteCourse(t *testing.T) {
	courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

	t.Run("success_deletes_course_and_its_lessons", func(t *testing.T) {
		h := newTestHarness(t)
		repo := NewPGRepository(slog.Default(), h.db)

		actual, err := repo.DeleteCourse(context.Background(), courseID)
		require.NoError(t, err)

		expected := []uuid.UUID{
			uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c"),
			uuid.MustParse("60766223-ff9f-4871-a497-f765c05a0c5e"),
			uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		}
		assert.ElementsMatch(t, expected, actual)

		_, err = repo.GetCourse(context.Background(), courseID)
		assert.ErrorIs(t, err, ErrCourseNotFound)

		var undeletedLessons int
		err = h.db.QueryRow(`SELECT COUNT(*) FROM lessons WHERE course_id = $1 AND deleted_at IS NULL`, courseID).Scan(&undeletedLessons)
		require.NoError(t, err)

		assert.Equal(t, 0, undeletedLessons)
	})

	t.Run("success_skips_lessons_already_deleted", func(t *testing.T) {
		h := newTestHarness(t)
		repo := NewPGRepository(slog.Default(), h.db)

		err := repo.DeleteLesson(context.Background(), courseID, uuid.MustParse("33333333-3333-3333-3333-333333333333"))
		require.NoError(t, err)

		actual, err := repo.DeleteCourse(context.Background(), courseID)
		require.NoError(t, err)

		expected := []uuid.UUID{
			uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c"),
			uuid.MustParse("60766223-ff9f-4871-a497-f765c05a0c5e"),
			uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		}
		assert.ElementsMatch(t, expected, actual)
	})

	t.Run("failure_already_deleted", func(t *testing.T) {
		h := newTestHarness(t)
		repo := NewPGRepository(slog.Default(), h.db)

		_, err := repo.DeleteCourse(context.Background(), courseID)
		require.NoError(t, err)

		actual, err := repo.DeleteCourse(context.Background(), courseID)
		assert.ErrorIs(t, err, ErrCourseNotFound)
		assert.Empty(t, actual)
	})

	t.Run("failure_not_found", func(t *testing.T) {
		h := newTestHarness(t)
		repo := NewPGRepository(slog.Default(), h.db)

		actual, err := repo.DeleteCourse(context.Background(), uuid.MustParse("00000000-0000-0000-0000-000000000000"))
		assert.ErrorIs(t, err, ErrCourseNotFound)
		assert.Empty(t, actual)
	})
}

func TestRepository_DeleteLesson(t *testing.T) {
	courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

	t.Run("success", func(t *testing.T) {
		h := newTestHarness(t)
		repo := NewPGRepository(slog.Default(), h.db)

		lessonID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

		err := repo.DeleteLesson(context.Background(), courseID, lessonID)
		require.NoError(t, err)

		_, err = repo.GetLesson(context.Background(), lessonID)
		assert.ErrorIs(t, err, ErrLessonNotFound)
	})

	t.Run("failure_already_deleted", func(t *testing.T) {
		h := newTestHarness(t)
		repo := NewPGRepository(slog.Default(), h.db)

		lessonID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
		err := repo.DeleteLesson(context.Background(), courseID, lessonID)
		require.NoError(t, err)

		err = repo.DeleteLesson(context.Background(), courseID, lessonID)
		assert.ErrorIs(t, err, ErrLessonNotFound)
	})

	t.Run("failure_not_found", func(t *testing.T) {
		h := newTestHarness(t)
		repo := NewPGRepository(slog.Default(), h.db)

		err := repo.DeleteLesson(context.Background(), courseID, uuid.MustParse("00000000-0000-0000-0000-000000000000"))
		assert.ErrorIs(t, err, ErrLessonNotFound)
	})

	t.Run("failure_lesson_of_another_course", func(t *testing.T) {
		h := newTestHarness(t)
		repo := NewPGRepository(slog.Default(), h.db)

		lessonID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
		expected, err := repo.GetLesson(context.Background(), lessonID)
		require.NoError(t, err)

		err = repo.DeleteLesson(context.Background(), uuid.MustParse("1f30a72f-5d7a-48da-a5c2-42efece6972a"), lessonID)
		assert.ErrorIs(t, err, ErrLessonNotFound)

		actual, err := repo.GetLesson(context.Background(), lessonID)
		require.NoError(t, err)

		assert.Equal(t, expected, actual)
	})
}

func TestRepository_GetCourseDeckIDs(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(slog.Default(), h.db)

	courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

	// Lesson 1 already has deck 60766223-... at order 1 (see populateTestDB).
	_, err := h.db.Exec(`
		INSERT INTO lesson_decks (lesson_id, deck_id, "order") VALUES
			('60766223-ff9f-4871-a497-f765c05a0c5e', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 1),
			('334ddbf8-1acc-405b-86d8-49f0d1ca636c', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 2),
			('11111111-1111-1111-1111-111111111111', '60766223-ff9f-4871-a497-f765c05a0c5e', 1),
			('22222222-2222-2222-2222-222222222222', 'cccccccc-cccc-cccc-cccc-cccccccccccc', 1)
	`)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		expected := []uuid.UUID{
			uuid.MustParse("60766223-ff9f-4871-a497-f765c05a0c5e"),
			uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
			uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"),
		}

		actual, err := repo.GetCourseDeckIDs(context.Background(), courseID)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("skips_deleted_lessons", func(t *testing.T) {
		_, err := h.db.Exec(`UPDATE lessons SET deleted_at = NOW() WHERE id = '22222222-2222-2222-2222-222222222222'`)
		require.NoError(t, err)

		expected := []uuid.UUID{
			uuid.MustParse("60766223-ff9f-4871-a497-f765c05a0c5e"),
			uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		}

		actual, err := repo.GetCourseDeckIDs(context.Background(), courseID)
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("empty", func(t *testing.T) {
		actual, err := repo.GetCourseDeckIDs(context.Background(), uuid.MustParse("00000000-0000-0000-0000-000000000000"))
		require.NoError(t, err)
		assert.Empty(t, actual)
	})

	t.Run("skips_deleted_course", func(t *testing.T) {
		_, err := h.db.Exec(`UPDATE courses SET deleted_at = NOW() WHERE id = $1`, courseID)
		require.NoError(t, err)

		actual, err := repo.GetCourseDeckIDs(context.Background(), courseID)
		require.NoError(t, err)
		assert.Empty(t, actual)
	})
}

func TestRepository_GetLessonDeckIDs(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(slog.Default(), h.db)

	// The lesson already has deck 60766223-... at order 1 (see populateTestDB).
	_, err := h.db.Exec(`
		INSERT INTO lesson_decks (lesson_id, deck_id, "order") VALUES
			('334ddbf8-1acc-405b-86d8-49f0d1ca636c', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 0)
	`)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		expected := []uuid.UUID{
			uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
			uuid.MustParse("60766223-ff9f-4871-a497-f765c05a0c5e"),
		}

		actual, err := repo.GetLessonDeckIDs(context.Background(), uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c"))
		require.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("empty", func(t *testing.T) {
		actual, err := repo.GetLessonDeckIDs(context.Background(), uuid.MustParse("60766223-ff9f-4871-a497-f765c05a0c5e"))
		require.NoError(t, err)
		assert.Empty(t, actual)
	})

	t.Run("skips_deleted_lesson", func(t *testing.T) {
		_, err := h.db.Exec(`UPDATE lessons SET deleted_at = NOW() WHERE id = '334ddbf8-1acc-405b-86d8-49f0d1ca636c'`)
		require.NoError(t, err)

		actual, err := repo.GetLessonDeckIDs(context.Background(), uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c"))
		require.NoError(t, err)
		assert.Empty(t, actual)
	})

	t.Run("skips_deleted_course", func(t *testing.T) {
		_, err := h.db.Exec(`UPDATE lessons SET deleted_at = NULL WHERE id = '334ddbf8-1acc-405b-86d8-49f0d1ca636c'`)
		require.NoError(t, err)
		_, err = h.db.Exec(`UPDATE courses SET deleted_at = NOW() WHERE id = 'fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72'`)
		require.NoError(t, err)

		actual, err := repo.GetLessonDeckIDs(context.Background(), uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c"))
		require.NoError(t, err)
		assert.Empty(t, actual)
	})
}

func TestRepository_GetLessonsByCourseID(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(slog.Default(), h.db)

	courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

	t.Run("success", func(t *testing.T) {
		pag := pagination.NewOldestFirstPagination(pagination.WithFirst(10))

		conn, err := repo.GetLessonsByCourseID(context.Background(), courseID, pag, false)
		assert.NoError(t, err)
		assert.NotEmpty(t, conn.Edges)
		assert.Equal(t, "Introduction to Goroutines", conn.Edges[0].Lesson.Title)
	})

	t.Run("empty_course", func(t *testing.T) {
		emptyID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
		pag := pagination.NewOldestFirstPagination(pagination.WithFirst(10))

		conn, err := repo.GetLessonsByCourseID(context.Background(), emptyID, pag, false)
		assert.NoError(t, err)
		assert.Empty(t, conn.Edges)
	})

	t.Run("pagination_forward_and_backward", func(t *testing.T) {
		ctx := context.Background()

		forward := pagination.NewOldestFirstPagination(pagination.WithFirst(2))

		page1, err := repo.GetLessonsByCourseID(ctx, courseID, forward, false)
		assert.NoError(t, err)
		require.Len(t, page1.Edges, 2)
		assert.Equal(t, []string{"Introduction to Goroutines", "Channels and Select"}, []string{
			page1.Edges[0].Lesson.Title,
			page1.Edges[1].Lesson.Title,
		})

		forward.After = page1.Edges[len(page1.Edges)-1].Cursor
		page2, err := repo.GetLessonsByCourseID(ctx, courseID, forward, false)
		assert.NoError(t, err)
		require.Len(t, page2.Edges, 2)
		assert.Equal(t, []string{"Interfaces and Structs", "Generics Basics"}, []string{
			page2.Edges[0].Lesson.Title,
			page2.Edges[1].Lesson.Title,
		})

		forward.After = page2.Edges[len(page2.Edges)-1].Cursor
		page3, err := repo.GetLessonsByCourseID(ctx, courseID, forward, false)
		assert.NoError(t, err)
		require.Len(t, page3.Edges, 1)
		assert.Equal(t, "Concurrency Patterns", page3.Edges[0].Lesson.Title)

		backward := pagination.NewOldestFirstPagination(pagination.WithLast(2))
		b1, err := repo.GetLessonsByCourseID(ctx, courseID, backward, false)
		assert.NoError(t, err)
		require.Len(t, b1.Edges, 2)
		assert.Equal(t, []string{"Generics Basics", "Concurrency Patterns"}, []string{
			b1.Edges[0].Lesson.Title,
			b1.Edges[1].Lesson.Title,
		})

		backward.Before = b1.Edges[0].Cursor
		b2, err := repo.GetLessonsByCourseID(ctx, courseID, backward, false)
		assert.NoError(t, err)
		require.Len(t, b2.Edges, 2)
		assert.Equal(t, []string{"Channels and Select", "Interfaces and Structs"}, []string{
			b2.Edges[0].Lesson.Title,
			b2.Edges[1].Lesson.Title,
		})

		backward.Before = b2.Edges[0].Cursor
		b3, err := repo.GetLessonsByCourseID(ctx, courseID, backward, false)
		assert.NoError(t, err)
		require.Len(t, b3.Edges, 1)
		assert.Equal(t, "Introduction to Goroutines", b3.Edges[0].Lesson.Title)
	})
}

func TestRepository_EnrollUserInCourse(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(slog.Default(), h.db)

	userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
	courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

	t.Run("success", func(t *testing.T) {
		err := repo.EnrollUserInCourse(context.Background(), userID, courseID, *newTestProgressState())
		assert.NoError(t, err)

		// Verify enrollment exists
		var exists bool
		row := h.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM user_course_progress WHERE user_id = $1 AND course_id = $2)`, userID, courseID)
		err = row.Scan(&exists)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("failure_already_enrolled", func(t *testing.T) {
		err := repo.EnrollUserInCourse(context.Background(), userID, courseID, *newTestProgressState())
		assert.ErrorIs(t, err, ErrUserAlreadyEnrolled)
	})
}

func TestRepository_GetUserCourseProgress(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(slog.Default(), h.db)

	userID := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
	courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

	t.Run("success", func(t *testing.T) {
		err := repo.EnrollUserInCourse(context.Background(), userID, courseID, *newTestProgressState())
		assert.NoError(t, err)

		progress, err := repo.GetUserCourseProgress(context.Background(), userID, courseID)
		assert.NoError(t, err)
		assert.Equal(t, userID, progress.UserID)
		assert.Equal(t, courseID, progress.CourseID)
		assert.NotNil(t, progress.State)
	})

	t.Run("failure_not_enrolled", func(t *testing.T) {
		unknownUser := uuid.MustParse("00000000-0000-0000-0000-000000000000")

		progress, err := repo.GetUserCourseProgress(context.Background(), unknownUser, courseID)
		assert.Error(t, err)
		assert.Empty(t, progress.ID)
	})
}

func TestRepository_UpdateUserProgress(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(slog.Default(), h.db)

	userID := uuid.MustParse("6363e2c6-d89e-4610-92e8-1e1d2fea49ec")
	courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

	t.Run("success", func(t *testing.T) {
		// First enroll
		err := repo.EnrollUserInCourse(context.Background(), userID, courseID, *newTestProgressState())
		assert.NoError(t, err)

		// Get progress
		progress, err := repo.GetUserCourseProgress(context.Background(), userID, courseID)
		assert.NoError(t, err)

		// Update state
		lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c").String()
		deckID := uuid.MustParse("60766223-ff9f-4871-a497-f765c05a0c5e").String()
		cardID := uuid.MustParse("72bdff92-5bc8-4e1d-9217-d0b23e22ff33").String()

		progress.State.AnswerCard(lessonID, deckID, cardID, true)
		progress.UpdatedAt = time.Now()

		err = repo.UpdateUserProgress(context.Background(), progress)
		assert.NoError(t, err)

		// Verify update
		updatedProgress, err := repo.GetUserCourseProgress(context.Background(), userID, courseID)
		assert.NoError(t, err)
		assert.True(t, updatedProgress.State.IsCardAllAnswersCorrect(lessonID, deckID, cardID))
	})
}

func TestRepository_GetEnrolledCourses(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(slog.Default(), h.db)

	userID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	courseID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

	t.Run("success", func(t *testing.T) {
		// Enroll user in course
		err := repo.EnrollUserInCourse(context.Background(), userID, courseID, *newTestProgressState())
		assert.NoError(t, err)

		// Get enrolled courses
		pag := pagination.NewOldestFirstPagination(pagination.WithFirst(10))
		conn, err := repo.GetEnrolledCourses(context.Background(), userID, pag)
		assert.NoError(t, err)
		assert.NotEmpty(t, conn.Edges)
		assert.Equal(t, 1, len(conn.Edges))
		assert.Equal(t, "Go Fundamentals", conn.Edges[0].Course.Course.Title)
		assert.NotEmpty(t, conn.Edges[0].Course.CurrentLessonID)
	})

	t.Run("empty_result", func(t *testing.T) {
		emptyUserID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
		pag := pagination.NewOldestFirstPagination(pagination.WithFirst(10))

		conn, err := repo.GetEnrolledCourses(context.Background(), emptyUserID, pag)
		assert.NoError(t, err)
		assert.Empty(t, conn.Edges)
	})

	t.Run("pagination_forward", func(t *testing.T) {
		// Create another course and enroll
		course2ID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
		course2 := Course{
			ID:          course2ID,
			Title:       "Advanced Go",
			Description: "Master advanced Go techniques",
			CreatedAt:   time.Now(),
		}
		err := repo.StoreCourse(context.Background(), course2)
		require.NoError(t, err)

		time.Sleep(time.Millisecond * 10) // Ensure different updated_at

		err = repo.EnrollUserInCourse(context.Background(), userID, course2ID, *newTestProgressState())
		require.NoError(t, err)

		// Get first page
		ctx := context.Background()
		forward := pagination.NewOldestFirstPagination(pagination.WithFirst(1))

		page1, err := repo.GetEnrolledCourses(ctx, userID, forward)
		assert.NoError(t, err)
		require.Len(t, page1.Edges, 1)
		assert.True(t, page1.PageInfo.HasNextPage)
		assert.False(t, page1.PageInfo.HasPreviousPage)

		// Get second page
		forward.After = page1.PageInfo.EndCursor
		page2, err := repo.GetEnrolledCourses(ctx, userID, forward)
		assert.NoError(t, err)
		require.Len(t, page2.Edges, 1)
		assert.False(t, page2.PageInfo.HasNextPage)
	})
}

// ============================================
// Backoffice Integration Tests
// ============================================

func TestRepository_CountEnrolledUsers(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(slog.Default(), h.db)

	// Create a dedicated course for this test
	courseID := uuid.MustParse("11110000-0000-0000-0000-000000000001")
	course := Course{
		ID:          courseID,
		Order:       200,
		Title:       "Count Test Course",
		Description: "For testing CountEnrolledUsers",
		CreatedAt:   time.Now(),
	}
	err := repo.StoreCourse(context.Background(), course)
	require.NoError(t, err)

	t.Run("empty_course", func(t *testing.T) {
		count, err := repo.CountEnrolledUsers(context.Background(), courseID)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("with_enrolled_users", func(t *testing.T) {
		// Enroll 3 users
		userID1 := uuid.MustParse("11111111-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
		userID2 := uuid.MustParse("22222222-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
		userID3 := uuid.MustParse("33333333-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

		err := repo.EnrollUserInCourse(context.Background(), userID1, courseID, *newTestProgressState())
		require.NoError(t, err)
		err = repo.EnrollUserInCourse(context.Background(), userID2, courseID, *newTestProgressState())
		require.NoError(t, err)
		err = repo.EnrollUserInCourse(context.Background(), userID3, courseID, *newTestProgressState())
		require.NoError(t, err)

		count, err := repo.CountEnrolledUsers(context.Background(), courseID)
		assert.NoError(t, err)
		assert.Equal(t, 3, count)
	})
}

func TestRepository_GetAllCourses(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(slog.Default(), h.db)
	ctx := context.Background()

	require.NoError(t, repo.StoreCourse(ctx, Course{
		ID: uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000000"), Order: 2, Title: "B", Description: "d", CreatedAt: time.Now(),
	}))
	require.NoError(t, repo.StoreCourse(ctx, Course{
		ID: uuid.MustParse("cccccccc-0000-0000-0000-000000000000"), Order: 3, Title: "C", Description: "d", CreatedAt: time.Now(),
	}))

	titles := func(conn CoursesConnection) []string {
		out := make([]string, len(conn.Edges))
		for i, e := range conn.Edges {
			out[i] = e.Course.Title
		}
		return out
	}

	t.Run("forward_with_unique_order", func(t *testing.T) {
		p := pagination.NewOldestFirstPagination(pagination.WithFirst(2))

		page1, err := repo.GetAllCourses(ctx, p)
		require.NoError(t, err)
		assert.Equal(t, []string{"Go Fundamentals", "B"}, titles(page1))
		assert.True(t, page1.PageInfo.HasNextPage)

		p.After = page1.PageInfo.EndCursor
		page2, err := repo.GetAllCourses(ctx, p)
		require.NoError(t, err)
		assert.Equal(t, []string{"C"}, titles(page2))
		assert.False(t, page2.PageInfo.HasNextPage)
	})

	t.Run("backward_with_unique_order", func(t *testing.T) {
		p := pagination.NewOldestFirstPagination(pagination.WithLast(2))

		page1, err := repo.GetAllCourses(ctx, p)
		require.NoError(t, err)
		assert.Equal(t, []string{"B", "C"}, titles(page1))
		assert.True(t, page1.PageInfo.HasPreviousPage)

		p.Before = page1.PageInfo.StartCursor
		page2, err := repo.GetAllCourses(ctx, p)
		require.NoError(t, err)
		assert.Equal(t, []string{"Go Fundamentals"}, titles(page2))
		assert.False(t, page2.PageInfo.HasPreviousPage)
	})

	// A shares order 2 with B, so each page boundary below falls between them.
	require.NoError(t, repo.StoreCourse(ctx, Course{
		ID: uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000000"), Order: 2, Title: "A", Description: "d", CreatedAt: time.Now(),
	}))

	t.Run("forward_with_duplicate_order", func(t *testing.T) {
		p := pagination.NewOldestFirstPagination(pagination.WithFirst(2))

		page1, err := repo.GetAllCourses(ctx, p)
		require.NoError(t, err)
		assert.Equal(t, []string{"Go Fundamentals", "A"}, titles(page1))
		assert.True(t, page1.PageInfo.HasNextPage)

		p.After = page1.PageInfo.EndCursor
		page2, err := repo.GetAllCourses(ctx, p)
		require.NoError(t, err)
		assert.Equal(t, []string{"B", "C"}, titles(page2))
		assert.False(t, page2.PageInfo.HasNextPage)
	})

	t.Run("backward_with_duplicate_order", func(t *testing.T) {
		p := pagination.NewOldestFirstPagination(pagination.WithLast(2))

		page1, err := repo.GetAllCourses(ctx, p)
		require.NoError(t, err)
		assert.Equal(t, []string{"B", "C"}, titles(page1))
		assert.True(t, page1.PageInfo.HasPreviousPage)

		p.Before = page1.PageInfo.StartCursor
		page2, err := repo.GetAllCourses(ctx, p)
		require.NoError(t, err)
		assert.Equal(t, []string{"Go Fundamentals", "A"}, titles(page2))
		assert.False(t, page2.PageInfo.HasPreviousPage)
	})

	t.Run("excludes_deleted", func(t *testing.T) {
		_, err := h.db.Exec(`UPDATE courses SET deleted_at = NOW() WHERE id = 'cccccccc-0000-0000-0000-000000000000'`)
		require.NoError(t, err)

		page, err := repo.GetAllCourses(ctx, pagination.NewOldestFirstPagination(pagination.WithFirst(10)))
		require.NoError(t, err)
		assert.Equal(t, []string{"Go Fundamentals", "A", "B"}, titles(page))

		actual, err := repo.CountCourses(ctx)
		require.NoError(t, err)
		assert.Equal(t, 3, actual)
	})
}

func TestRepository_GetEnrolledUsersProgress(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(slog.Default(), h.db)

	// Create a dedicated course for this test
	courseID := uuid.MustParse("22220000-0000-0000-0000-000000000002")
	course := Course{
		ID:          courseID,
		Order:       201,
		Title:       "Progress Test Course",
		Description: "For testing GetEnrolledUsersProgress",
		CreatedAt:   time.Now(),
	}
	err := repo.StoreCourse(context.Background(), course)
	require.NoError(t, err)

	// Create lessons for the course
	lesson1ID := uuid.MustParse("22220000-1111-1111-1111-111111111111")
	lesson2ID := uuid.MustParse("22220000-2222-2222-2222-222222222222")
	lesson1 := Lesson{
		ID:          lesson1ID,
		CourseID:    courseID,
		Order:       1,
		Title:       "Lesson 1",
		Description: "First lesson",
		Body:        "Body 1",
		CreatedAt:   time.Now(),
	}
	lesson2 := Lesson{
		ID:          lesson2ID,
		CourseID:    courseID,
		Order:       2,
		Title:       "Lesson 2",
		Description: "Second lesson",
		Body:        "Body 2",
		CreatedAt:   time.Now(),
	}
	err = repo.StoreLesson(context.Background(), lesson1)
	require.NoError(t, err)
	err = repo.StoreLesson(context.Background(), lesson2)
	require.NoError(t, err)

	t.Run("empty_course", func(t *testing.T) {
		pag := pagination.NewOldestFirstPagination(pagination.WithFirst(10))
		conn, err := repo.GetEnrolledUsersProgress(context.Background(), courseID, pag)
		assert.NoError(t, err)
		assert.Empty(t, conn.Edges)
	})

	t.Run("with_enrolled_users_progress", func(t *testing.T) {
		// Enroll user with completed first lesson
		userID := uuid.MustParse("22220000-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
		state := NewProgressState()
		state.CurrentLessonID = lesson2ID

		// Mark lesson 1 as completed
		state.Lessons[lesson1ID.String()] = &LessonProgress{
			IsCompleted: true,
			Decks:       make(map[string]*DeckProgress),
		}
		// Lesson 2 in progress
		state.Lessons[lesson2ID.String()] = &LessonProgress{
			IsCompleted: false,
			Decks:       make(map[string]*DeckProgress),
		}

		// Add some deck/card progress
		deckID := "22220000-dddd-dddd-dddd-dddddddddddd"
		cardID := "22220000-cccc-cccc-cccc-cccccccccccc"
		state.Lessons[lesson2ID.String()].Decks[deckID] = &DeckProgress{
			IsCompleted: false,
			Cards: map[string]*CardProgress{
				cardID: {
					CorrectAnswers:   3,
					IncorrectAnswers: 1,
					IsCompleted:      false,
				},
			},
		}

		err := repo.EnrollUserInCourse(context.Background(), userID, courseID, *state)
		require.NoError(t, err)

		// Get progress
		pag := pagination.NewOldestFirstPagination(pagination.WithFirst(10))
		conn, err := repo.GetEnrolledUsersProgress(context.Background(), courseID, pag)
		assert.NoError(t, err)
		require.Len(t, conn.Edges, 1)

		student := conn.Edges[0].Student
		assert.Equal(t, userID, student.UserID)
		assert.Equal(t, 1, student.LessonsCompleted)
		assert.Equal(t, 2, student.TotalLessons)
		assert.Equal(t, lesson2ID, student.CurrentLessonID)
		assert.Equal(t, 3, student.TotalCorrectAnswers)
		assert.Equal(t, 1, student.TotalIncorrectAnswers)
		assert.Len(t, student.LessonDetails, 2)
	})

	t.Run("pagination_forward", func(t *testing.T) {
		// Create a new course for pagination test
		pagCourseID := uuid.MustParse("33330000-0000-0000-0000-000000000003")
		pagCourse := Course{
			ID:          pagCourseID,
			Order:       202,
			Title:       "Pagination Test Course",
			Description: "For testing pagination",
			CreatedAt:   time.Now(),
		}
		err := repo.StoreCourse(context.Background(), pagCourse)
		require.NoError(t, err)

		// Create a lesson for the course (required for foreign key)
		pagLessonID := uuid.MustParse("33330000-1111-1111-1111-111111111111")
		pagLesson := Lesson{
			ID:          pagLessonID,
			CourseID:    pagCourseID,
			Order:       1,
			Title:       "Pagination Lesson",
			Description: "For testing pagination",
			Body:        "Body",
			CreatedAt:   time.Now(),
		}
		err = repo.StoreLesson(context.Background(), pagLesson)
		require.NoError(t, err)

		// Enroll 5 users
		for i := 1; i <= 5; i++ {
			userID := uuid.MustParse(fmt.Sprintf("33330000-aaaa-aaaa-aaaa-%012d", i))
			state := NewProgressState()
			state.CurrentLessonID = pagLessonID
			err := repo.EnrollUserInCourse(context.Background(), userID, pagCourseID, *state)
			require.NoError(t, err)
			time.Sleep(10 * time.Millisecond) // Ensure different created_at
		}

		// Get first page (2 items)
		pag := pagination.NewOldestFirstPagination(pagination.WithFirst(2))
		page1, err := repo.GetEnrolledUsersProgress(context.Background(), pagCourseID, pag)
		assert.NoError(t, err)
		assert.Len(t, page1.Edges, 2)
		assert.True(t, page1.PageInfo.HasNextPage)
		assert.False(t, page1.PageInfo.HasPreviousPage)

		// Get second page
		pag2 := pagination.NewOldestFirstPagination(
			pagination.WithFirst(2),
			pagination.WithAfter(page1.PageInfo.EndCursor),
		)
		page2, err := repo.GetEnrolledUsersProgress(context.Background(), pagCourseID, pag2)
		assert.NoError(t, err)
		assert.Len(t, page2.Edges, 2)
		assert.True(t, page2.PageInfo.HasNextPage)

		// Get third page (only 1 left)
		pag3 := pagination.NewOldestFirstPagination(
			pagination.WithFirst(2),
			pagination.WithAfter(page2.PageInfo.EndCursor),
		)
		page3, err := repo.GetEnrolledUsersProgress(context.Background(), pagCourseID, pag3)
		assert.NoError(t, err)
		assert.Len(t, page3.Edges, 1)
		assert.False(t, page3.PageInfo.HasNextPage)
	})
}

type testHarness struct {
	db          *sql.DB
	redisClient radix.Client
}

func newTestProgressState() *ProgressState {
	state := NewProgressState()
	lessonID := uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c")
	deckID := uuid.MustParse("60766223-ff9f-4871-a497-f765c05a0c5e")
	cardID := uuid.MustParse("72bdff92-5bc8-4e1d-9217-d0b23e22ff33")

	state.CurrentLessonID = lessonID

	// Initialize lesson progress
	state.Lessons[lessonID.String()] = &LessonProgress{
		Decks:       make(map[string]*DeckProgress),
		IsCompleted: false,
	}

	// Initialize deck progress
	state.Lessons[lessonID.String()].Decks[deckID.String()] = &DeckProgress{
		Cards:       make(map[string]*CardProgress),
		IsCompleted: false,
	}

	// Initialize card progress
	state.Lessons[lessonID.String()].Decks[deckID.String()].Cards[cardID.String()] = &CardProgress{
		CorrectAnswers:   0,
		IncorrectAnswers: 0,
		IsCompleted:      false,
	}

	return state
}

func newTestHarness(t *testing.T) testHarness {
	ctx := context.Background()

	// Initialize PostgreSQL
	dbConfig := config.DBConfig{
		User:     "toshokan",
		Password: "t.o.s.h.o.k.a.n.",
		Name:     "test_course",
		Host:     "localhost",
		Port:     "5432",
	}

	database, err := db.InitDB(dbConfig)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		database.Close()
	})

	driver, err := postgres.WithInstance(database, &postgres.Config{})
	if err != nil {
		t.Fatal(err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://../../cmd/migrate/migrations", "toshokan", driver)
	if err != nil {
		t.Fatal(err)
	}

	err = m.Down()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatal(err)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatal(err)
	}

	if err := populateTestDB(database); err != nil {
		t.Fatal(err)
	}

	// Initialize Redis
	redisClient, err := (radix.PoolConfig{}).New(ctx, "tcp", "localhost:6379")
	if err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Flush the Redis test database
	if err := redisClient.Do(ctx, radix.Cmd(nil, "FLUSHDB")); err != nil {
		t.Fatalf("Failed to flush Redis: %v", err)
	}

	t.Cleanup(func() {
		redisClient.Close()
	})

	return testHarness{
		db:          database,
		redisClient: redisClient,
	}
}

func populateTestDB(db *sql.DB) error {
	_, err := db.Exec(`
		INSERT INTO courses (id, "order", title, description, created_at)
		VALUES (
			'fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72',
			1,
			'Go Fundamentals',
			'Learn the basics of Go',
			'2024-01-01'
		);

		INSERT INTO lessons (id, course_id, "order", title, description, body, created_at)
		VALUES (
			'334ddbf8-1acc-405b-86d8-49f0d1ca636c',
			'fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72',
			1,
			'Introduction to Goroutines',
			'Learn about concurrent programming',
			'Content about goroutines',
			'2024-01-01'
		),
		(
			'60766223-ff9f-4871-a497-f765c05a0c5e',
			'fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72',
			2,
			'Channels and Select',
			'Learn about channels',
			'Content about channels',
			'2024-01-01'
		),
		(
			'11111111-1111-1111-1111-111111111111',
			'fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72',
			3,
			'Interfaces and Structs',
			'Work with types',
			'Content about interfaces',
			'2024-01-01'
		),
		(
			'22222222-2222-2222-2222-222222222222',
			'fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72',
			4,
			'Generics Basics',
			'Learn generics',
			'Content about generics',
			'2024-01-01'
		),
		(
			'33333333-3333-3333-3333-333333333333',
			'fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72',
			5,
			'Concurrency Patterns',
			'Patterns with goroutines',
			'Content about concurrency patterns',
			'2024-01-01'
		);

		INSERT INTO lesson_decks (id, lesson_id, deck_id, "order", created_at)
		VALUES (
			'72bdff92-5bc8-4e1d-9217-d0b23e22ff33',
			'334ddbf8-1acc-405b-86d8-49f0d1ca636c',
			'60766223-ff9f-4871-a497-f765c05a0c5e',
			1,
			'2024-01-01'
		);
	`)

	return errors.Trace(err)
}
