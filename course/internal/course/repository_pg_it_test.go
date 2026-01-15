package course

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/XaviFP/toshokan/common/config"
	"github.com/XaviFP/toshokan/common/db"
	"github.com/XaviFP/toshokan/common/pagination"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
	"github.com/mediocregopher/radix/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_StoreCourse(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(h.db)

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
	repo := NewPGRepository(h.db)

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
	repo := NewPGRepository(h.db)

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

		// update if exists
		lesson.Title = "Advanced Patterns Updated"
		err = repo.StoreLesson(context.Background(), lesson)
		assert.NoError(t, err)

		row = h.db.QueryRow(`SELECT id, course_id, "order", title, description, body FROM lessons WHERE id = $1 AND deleted_at IS NULL`, lesson.ID)
		err = row.Scan(&out.ID, &out.CourseID, &out.Order, &out.Title, &out.Description, &out.Body)
		assert.NoError(t, err)
		assert.Equal(t, lesson.ID, out.ID)
		assert.Equal(t, "Advanced Patterns Updated", out.Title)
	})
}

func TestRepository_UpdateCourse(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(h.db)

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
	repo := NewPGRepository(h.db)

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
	repo := NewPGRepository(h.db)

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

func TestRepository_GetLessonsByCourseID(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(h.db)

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
	repo := NewPGRepository(h.db)

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
	repo := NewPGRepository(h.db)

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
	repo := NewPGRepository(h.db)

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
	repo := NewPGRepository(h.db)

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
