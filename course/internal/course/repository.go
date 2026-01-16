package course

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/juju/errors"
	"github.com/mediocregopher/radix/v4"

	cachelib "github.com/XaviFP/toshokan/common/cache"
	"github.com/XaviFP/toshokan/common/db"
	"github.com/XaviFP/toshokan/common/pagination"
)

var ErrNotFound = errors.New("not found")

// Repository defines the interface for courses data access
type Repository interface {
	// Courses
	GetCourse(ctx context.Context, id uuid.UUID) (Course, error)
	StoreCourse(ctx context.Context, course Course) error
	UpdateCourse(ctx context.Context, id uuid.UUID, updates CourseUpdates) (Course, error)
	GetEnrolledCourses(ctx context.Context, userID uuid.UUID, p pagination.Pagination) (CoursesWithProgressConnection, error)

	// Lessons
	GetLesson(ctx context.Context, id uuid.UUID) (Lesson, error)
	// GetFirstLessonInCourse(ctx context.Context, courseID uuid.UUID) (Lesson, error)

	// GetLessonsByCourseID retrieves lessons for a course with pagination
	// If bodyless is true, the body field is not fetched (intended for faster pagination)
	GetLessonsByCourseID(ctx context.Context, courseID uuid.UUID, p pagination.Pagination, bodyless bool) (LessonsConnection, error)
	StoreLesson(ctx context.Context, lesson Lesson) error
	UpdateLesson(ctx context.Context, id uuid.UUID, updates LessonUpdates) (Lesson, error)

	// User Progress
	GetUserCourseProgress(ctx context.Context, userID uuid.UUID, courseID uuid.UUID) (UserCourseProgress, error)
	EnrollUserInCourse(ctx context.Context, userID uuid.UUID, courseID uuid.UUID, progressState ProgressState) error
	UpdateUserProgress(ctx context.Context, progress UserCourseProgress) error
}

// CourseUpdates contains optional fields for updating a course
type CourseUpdates struct {
	Order       *int64
	Title       *string
	Description *string
}

// HasUpdates returns true if at least one field is set
func (u CourseUpdates) HasUpdates() bool {
	return u.Order != nil || u.Title != nil || u.Description != nil
}

// LessonUpdates contains optional fields for updating a lesson
type LessonUpdates struct {
	Order       *int64
	Title       *string
	Description *string
	Body        *string
}

// HasUpdates returns true if at least one field is set
func (u LessonUpdates) HasUpdates() bool {
	return u.Order != nil || u.Title != nil || u.Description != nil || u.Body != nil
}

type redisRepository struct {
	cache cachelib.Cache
	db    Repository
}

func (r *redisRepository) getCachedJSON(ctx context.Context, key string, dest interface{}) error {
	cached, err := r.cache.Get(ctx, key)
	if err != nil {
		if errors.Is(err, cachelib.ErrNoValueForKey) {
			return errors.Trace(err)
		}

		return errors.Trace(err)
	}

	if err := json.Unmarshal([]byte(cached), dest); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// NewRedisRepository creates a new cached repository
func NewRedisRepository(cache radix.Client, pg Repository) Repository {
	return &redisRepository{
		cache: cachelib.NewCache(cache),
		db:    pg,
	}
}

func (r *redisRepository) courseKey(id uuid.UUID) string {
	return fmt.Sprintf("course:%s", id.String())
}

func (r *redisRepository) lessonKey(id uuid.UUID) string {
	return fmt.Sprintf("lesson:%s", id.String())
}

func (r *redisRepository) userProgressKey(userID, courseID uuid.UUID) string {
	return fmt.Sprintf("user_progress:%s:%s", userID.String(), courseID.String())
}

// GetCourse retrieves a course, checking cache first
func (r *redisRepository) GetCourse(ctx context.Context, id uuid.UUID) (Course, error) {
	key := r.courseKey(id)

	var course Course
	err := r.getCachedJSON(ctx, key, &course)
	if err != nil && !errors.Is(err, cachelib.ErrNoValueForKey) {
		return Course{}, errors.Trace(err)
	}

	// Cache hit
	if err == nil {
		return course, nil
	}

	// Cache miss, fetch from DB
	course, err = r.db.GetCourse(ctx, id)
	if err != nil {
		return Course{}, errors.Trace(err)
	}

	// Cache it
	data, err := json.Marshal(course)
	if err != nil {
		return Course{}, errors.Trace(err)
	}

	if err := r.cache.SetEx(ctx, key, string(data), 3600); err != nil {
		return Course{}, errors.Trace(err)
	}

	return course, nil
}

// StoreCourse saves a course and invalidates cache
func (r *redisRepository) StoreCourse(ctx context.Context, course Course) error {
	err := r.db.StoreCourse(ctx, course)
	if err != nil {
		return errors.Trace(err)
	}

	// Invalidate cache
	if err := r.cache.Delete(ctx, r.courseKey(course.ID)); err != nil {
		log.Printf("cache delete failed for course %s: %v", course.ID, err)
	}

	return nil
}

// UpdateCourse updates a course and invalidates cache
func (r *redisRepository) UpdateCourse(ctx context.Context, id uuid.UUID, updates CourseUpdates) (Course, error) {
	course, err := r.db.UpdateCourse(ctx, id, updates)
	if err != nil {
		return Course{}, errors.Trace(err)
	}

	// Invalidate cache
	if err := r.cache.Delete(ctx, r.courseKey(id)); err != nil {
		log.Printf("cache delete failed for course %s: %v", id, err)
	}

	return course, nil
}

// GetLesson retrieves a lesson, checking cache first
func (r *redisRepository) GetLesson(ctx context.Context, id uuid.UUID) (Lesson, error) {
	key := r.lessonKey(id)

	var lesson Lesson
	err := r.getCachedJSON(ctx, key, &lesson)
	if err != nil && !errors.Is(err, cachelib.ErrNoValueForKey) {
		return Lesson{}, errors.Trace(err)
	}

	// Cache hit
	if err == nil {
		return lesson, nil
	}

	// Cache miss, fetch from DB
	lesson, err = r.db.GetLesson(ctx, id)
	if err != nil {
		return Lesson{}, errors.Trace(err)
	}

	// Cache it
	data, err := json.Marshal(lesson)
	if err == nil {
		if err := r.cache.SetEx(ctx, key, string(data), 3600); err != nil {
			log.Printf("cache set failed for lesson %s: %v", lesson.ID, err)
		}
	}

	return lesson, nil
}

// GetLessonsByCourseID retrieves lessons for a course
// If bodyless is true, the body field is not fetched for faster pagination
func (r *redisRepository) GetLessonsByCourseID(ctx context.Context, courseID uuid.UUID, p pagination.Pagination, bodyless bool) (LessonsConnection, error) {
	return r.db.GetLessonsByCourseID(ctx, courseID, p, bodyless)
}

// GetEnrolledCourses retrieves enrolled courses for a user with progress
func (r *redisRepository) GetEnrolledCourses(ctx context.Context, userID uuid.UUID, p pagination.Pagination) (CoursesWithProgressConnection, error) {
	// For now, pass through to DB without caching the list
	// Individual courses are still cached via GetCourse
	return r.db.GetEnrolledCourses(ctx, userID, p)
}

// StoreLesson saves a lesson and invalidates cache
func (r *redisRepository) StoreLesson(ctx context.Context, lesson Lesson) error {
	err := r.db.StoreLesson(ctx, lesson)
	if err != nil {
		return errors.Trace(err)
	}

	// Invalidate cache
	if err := r.cache.Delete(ctx, r.lessonKey(lesson.ID)); err != nil {
		log.Printf("cache delete failed for lesson %s: %v", lesson.ID, err)
	}

	return nil
}

// UpdateLesson updates a lesson and invalidates cache
func (r *redisRepository) UpdateLesson(ctx context.Context, id uuid.UUID, updates LessonUpdates) (Lesson, error) {
	lesson, err := r.db.UpdateLesson(ctx, id, updates)
	if err != nil {
		return Lesson{}, errors.Trace(err)
	}

	// Invalidate cache
	if err := r.cache.Delete(ctx, r.lessonKey(id)); err != nil {
		log.Printf("cache delete failed for lesson %s: %v", id, err)
	}

	return lesson, nil
}

// GetUserCourseProgress retrieves user progress, checking cache first
func (r *redisRepository) GetUserCourseProgress(ctx context.Context, userID uuid.UUID, courseID uuid.UUID) (UserCourseProgress, error) {
	key := r.userProgressKey(userID, courseID)

	var progress UserCourseProgress
	err := r.getCachedJSON(ctx, key, &progress)
	if err != nil && !errors.Is(err, cachelib.ErrNoValueForKey) {
		return UserCourseProgress{}, errors.Trace(err)
	}

	// Cache hit
	if err == nil {
		return progress, nil
	}

	// Cache miss, fetch from DB
	progress, err = r.db.GetUserCourseProgress(ctx, userID, courseID)
	if err != nil {
		return UserCourseProgress{}, errors.Trace(err)
	}

	// Cache it
	data, err := json.Marshal(progress)
	if err == nil {
		if err := r.cache.SetEx(ctx, key, string(data), 1800); err != nil {
			log.Printf("cache set failed for user_progress %s:%s: %v", userID, courseID, err)
		}
	}

	return progress, nil
}

// EnrollUserInCourse enrolls a user
func (r *redisRepository) EnrollUserInCourse(ctx context.Context, userID uuid.UUID, courseID uuid.UUID, progressState ProgressState) error {
	err := r.db.EnrollUserInCourse(ctx, userID, courseID, progressState)
	if err != nil {
		return errors.Trace(err)
	}

	// Invalidate cache
	if err := r.cache.Delete(ctx, r.userProgressKey(userID, courseID)); err != nil {
		log.Printf("cache delete failed for user_progress %s:%s: %v", userID, courseID, err)
	}

	return nil
}

// UpdateUserProgress updates user progress and invalidates cache
func (r *redisRepository) UpdateUserProgress(ctx context.Context, progress UserCourseProgress) error {
	err := r.db.UpdateUserProgress(ctx, progress)
	if err != nil {
		return errors.Trace(err)
	}

	// Invalidate cache
	if err := r.cache.Delete(ctx, r.userProgressKey(progress.UserID, progress.CourseID)); err != nil {
		log.Printf("cache delete failed for user_progress %s:%s: %v", progress.UserID, progress.CourseID, err)
	}

	return nil
}

type pgRepository struct {
	db *sql.DB
}

// NewPGRepository creates a new PostgreSQL repository
func NewPGRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

// GetCourse retrieves a course by ID
func (r *pgRepository) GetCourse(ctx context.Context, id uuid.UUID) (Course, error) {
	var course Course
	err := r.db.QueryRowContext(ctx,
		`SELECT id, "order", title, description, created_at, updated_at, deleted_at 
		 FROM courses WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(&course.ID, &course.Order, &course.Title, &course.Description, &course.CreatedAt, &course.UpdatedAt, &course.DeletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Course{}, errors.Trace(ErrCourseNotFound)
		}

		return Course{}, errors.Trace(err)
	}

	return course, nil
}

// StoreCourse saves a course to the database
func (r *pgRepository) StoreCourse(ctx context.Context, course Course) error {
	if course.Title == "" {
		return errors.Trace(ErrNoTitle)
	}
	if course.Description == "" {
		return errors.Trace(ErrNoDescription)
	}

	if course.ID == uuid.Nil {
		course.ID = uuid.New()
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO courses (id, "order", title, description, created_at) 
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (id) DO UPDATE SET "order" = $2, title = $3, description = $4, updated_at = $5`,
		course.ID, course.Order, course.Title, course.Description, time.Now(),
	)

	return errors.Trace(err)
}

// UpdateCourse updates a course with the provided fields
func (r *pgRepository) UpdateCourse(ctx context.Context, id uuid.UUID, updates CourseUpdates) (Course, error) {
	var arger db.Argumenter
	setClauses := []string{fmt.Sprintf("updated_at = %s", arger.Add(time.Now()))}

	if updates.Order != nil {
		setClauses = append(setClauses, fmt.Sprintf(`"order" = %s`, arger.Add(*updates.Order)))
	}
	if updates.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = %s", arger.Add(*updates.Title)))
	}
	if updates.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = %s", arger.Add(*updates.Description)))
	}

	query := fmt.Sprintf(
		`UPDATE courses SET %s WHERE id = %s AND deleted_at IS NULL
		 RETURNING id, "order", title, description, created_at, updated_at, deleted_at`,
		strings.Join(setClauses, ", "),
		arger.Add(id),
	)

	var course Course
	err := r.db.QueryRowContext(ctx, query, arger.Values()...).Scan(
		&course.ID, &course.Order, &course.Title, &course.Description,
		&course.CreatedAt, &course.UpdatedAt, &course.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Course{}, errors.Trace(ErrNotFound)
		}

		return Course{}, errors.Trace(err)
	}

	return course, nil
}

// GetLesson retrieves a lesson by ID
func (r *pgRepository) GetLesson(ctx context.Context, id uuid.UUID) (Lesson, error) {
	var lesson Lesson
	err := r.db.QueryRowContext(ctx,
		`SELECT id, course_id, "order", title, description, body, created_at, updated_at, deleted_at 
		 FROM lessons WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(&lesson.ID, &lesson.CourseID, &lesson.Order, &lesson.Title, &lesson.Description, &lesson.Body, &lesson.CreatedAt, &lesson.UpdatedAt, &lesson.DeletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Lesson{}, errors.Trace(ErrLessonNotFound)
		}

		return Lesson{}, errors.Trace(err)
	}

	return lesson, nil
}

// GetLessonsByCourseID retrieves lessons for a course with pagination
// If bodyless is true, the body field is not fetched for faster pagination
func (r *pgRepository) GetLessonsByCourseID(ctx context.Context, courseID uuid.UUID, p pagination.Pagination, bodyless bool) (LessonsConnection, error) {
	var (
		out   LessonsConnection
		arger db.Argumenter
	)

	whereClauses := []string{
		"deleted_at IS NULL",
		fmt.Sprintf("course_id = %s", arger.Add(courseID)),
	}

	if !p.Cursor().IsEmpty() {
		var cursor LessonCursor
		if err := pagination.FromCursor(p.Cursor(), &cursor); err != nil {
			return out, errors.Trace(err)
		}
		whereClauses = append(whereClauses, fmt.Sprintf(`"order" %s %s`, p.Comparator(), arger.Add(cursor.Order)))
	}

	// Build query with or without body based on bodyless flag
	var query string
	if bodyless {
		query = fmt.Sprintf(
			`SELECT id, course_id, "order", title, description, created_at, updated_at, deleted_at
			 FROM lessons
			 WHERE %s
			 ORDER BY "order" %s
			 LIMIT %s`,
			strings.Join(whereClauses, " AND "),
			p.OrderBy(),
			arger.Add(p.Limit()+1),
		)
	} else {
		query = fmt.Sprintf(
			`SELECT id, course_id, "order", title, description, body, created_at, updated_at, deleted_at
			 FROM lessons
			 WHERE %s
			 ORDER BY "order" %s
			 LIMIT %s`,
			strings.Join(whereClauses, " AND "),
			p.OrderBy(),
			arger.Add(p.Limit()+1),
		)
	}

	rows, err := r.db.QueryContext(ctx, query, arger.Values()...)
	if err != nil {
		return out, errors.Trace(err)
	}
	defer rows.Close()

	for rows.Next() {
		var l Lesson
		if bodyless {
			if err := rows.Scan(&l.ID, &l.CourseID, &l.Order, &l.Title, &l.Description, &l.CreatedAt, &l.UpdatedAt, &l.DeletedAt); err != nil {
				return out, errors.Trace(err)
			}
		} else {
			if err := rows.Scan(&l.ID, &l.CourseID, &l.Order, &l.Title, &l.Description, &l.Body, &l.CreatedAt, &l.UpdatedAt, &l.DeletedAt); err != nil {
				return out, errors.Trace(err)
			}
		}

		cursor, err := pagination.ToCursor(LessonCursor{Order: l.Order})
		if err != nil {
			return out, errors.Trace(err)
		}

		out.Edges = append(out.Edges, LessonEdge{
			Lesson: l,
			Cursor: cursor,
		})
	}

	if err := rows.Err(); err != nil {
		return out, errors.Trace(err)
	}

	hasMore := len(out.Edges) > p.Limit()

	pageInfo := pagination.PageInfo{
		HasPreviousPage: hasMore && !p.IsForward(),
		HasNextPage:     hasMore && p.IsForward(),
	}

	if hasMore {
		out.Edges = out.Edges[:len(out.Edges)-1]
	}

	// If backward pagination, reverse to restore natural order
	if !p.IsForward() {
		for i, j := 0, len(out.Edges)-1; i < j; i, j = i+1, j-1 {
			out.Edges[i], out.Edges[j] = out.Edges[j], out.Edges[i]
		}
	}

	if len(out.Edges) > 0 {
		pageInfo.StartCursor = out.Edges[0].Cursor
		pageInfo.EndCursor = out.Edges[len(out.Edges)-1].Cursor
	}

	out.PageInfo = pageInfo

	return out, nil
}

// GetEnrolledCourses retrieves courses a user is enrolled in with progress information
func (r *pgRepository) GetEnrolledCourses(ctx context.Context, userID uuid.UUID, p pagination.Pagination) (CoursesWithProgressConnection, error) {
	var (
		out   CoursesWithProgressConnection
		arger db.Argumenter
	)

	// Join user_course_progress with courses to get enrolled courses
	whereClauses := []string{
		"c.deleted_at IS NULL",
		fmt.Sprintf("ucp.user_id = %s", arger.Add(userID)),
	}

	if !p.Cursor().IsEmpty() {
		var cursor CourseCursor
		if err := pagination.FromCursor(p.Cursor(), &cursor); err != nil {
			return out, errors.Trace(err)
		}
		whereClauses = append(whereClauses, fmt.Sprintf(`c."order" %s %s`, p.Comparator(), arger.Add(cursor.Order)))
	}

	query := fmt.Sprintf(
		`SELECT c.id, c."order", c.title, c.description, c.created_at, c.updated_at, c.deleted_at,
		        ucp.current_lesson
		 FROM user_course_progress ucp
		 JOIN courses c ON c.id = ucp.course_id
		 WHERE %s
		 ORDER BY c."order" %s
		 LIMIT %s`,
		strings.Join(whereClauses, " AND "),
		p.OrderBy(),
		arger.Add(p.Limit()+1),
	)

	rows, err := r.db.QueryContext(ctx, query, arger.Values()...)
	if err != nil {
		return out, errors.Trace(err)
	}
	defer rows.Close()

	for rows.Next() {
		var c Course
		var currentLessonID uuid.UUID

		if err := rows.Scan(&c.ID, &c.Order, &c.Title, &c.Description, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt, &currentLessonID); err != nil {
			return out, errors.Trace(err)
		}

		cursor, err := pagination.ToCursor(CourseCursor{Order: c.Order})
		if err != nil {
			return out, errors.Trace(err)
		}

		out.Edges = append(out.Edges, CourseWithProgressEdge{
			Course: &CourseWithProgress{
				Course:          c,
				CurrentLessonID: currentLessonID.String(),
			},
			Cursor: cursor,
		})
	}

	if err := rows.Err(); err != nil {
		return out, errors.Trace(err)
	}

	hasMore := len(out.Edges) > p.Limit()

	pageInfo := pagination.PageInfo{
		HasPreviousPage: hasMore && !p.IsForward(),
		HasNextPage:     hasMore && p.IsForward(),
	}

	if hasMore {
		out.Edges = out.Edges[:len(out.Edges)-1]
	}

	// If backward pagination, reverse to restore natural order
	if !p.IsForward() {
		for i, j := 0, len(out.Edges)-1; i < j; i, j = i+1, j-1 {
			out.Edges[i], out.Edges[j] = out.Edges[j], out.Edges[i]
		}
	}

	if len(out.Edges) > 0 {
		pageInfo.StartCursor = out.Edges[0].Cursor
		pageInfo.EndCursor = out.Edges[len(out.Edges)-1].Cursor
	}

	out.PageInfo = pageInfo

	return out, nil
}

// StoreLesson saves a lesson to the database
func (r *pgRepository) StoreLesson(ctx context.Context, lesson Lesson) error {
	if lesson.Title == "" {
		return errors.Trace(ErrNoTitle)
	}
	if lesson.Description == "" {
		return errors.Trace(ErrNoDescription)
	}

	if lesson.ID == uuid.Nil {
		lesson.ID = uuid.New()
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO lessons (id, course_id, "order", title, description, body, created_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (id) DO UPDATE SET title = $4, description = $5, body = $6, updated_at = $7`,
		lesson.ID, lesson.CourseID, lesson.Order, lesson.Title, lesson.Description, lesson.Body, time.Now(),
	)

	return errors.Trace(err)
}

// UpdateLesson updates a lesson with the provided fields
func (r *pgRepository) UpdateLesson(ctx context.Context, id uuid.UUID, updates LessonUpdates) (Lesson, error) {
	var arger db.Argumenter
	setClauses := []string{fmt.Sprintf("updated_at = %s", arger.Add(time.Now()))}

	if updates.Order != nil {
		setClauses = append(setClauses, fmt.Sprintf(`"order" = %s`, arger.Add(*updates.Order)))
	}
	if updates.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = %s", arger.Add(*updates.Title)))
	}
	if updates.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = %s", arger.Add(*updates.Description)))
	}
	if updates.Body != nil {
		setClauses = append(setClauses, fmt.Sprintf("body = %s", arger.Add(*updates.Body)))
	}

	query := fmt.Sprintf(
		`UPDATE lessons SET %s WHERE id = %s AND deleted_at IS NULL
		 RETURNING id, course_id, "order", title, description, body, created_at, updated_at, deleted_at`,
		strings.Join(setClauses, ", "),
		arger.Add(id),
	)

	var lesson Lesson
	err := r.db.QueryRowContext(ctx, query, arger.Values()...).Scan(
		&lesson.ID, &lesson.CourseID, &lesson.Order, &lesson.Title, &lesson.Description,
		&lesson.Body, &lesson.CreatedAt, &lesson.UpdatedAt, &lesson.DeletedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Lesson{}, errors.Trace(ErrNotFound)
		}

		return Lesson{}, errors.Trace(err)
	}

	return lesson, nil
}

// GetUserCourseProgress retrieves a user's progress in a course
func (r *pgRepository) GetUserCourseProgress(ctx context.Context, userID uuid.UUID, courseID uuid.UUID) (UserCourseProgress, error) {
	var progress UserCourseProgress
	var stateJSON []byte

	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, course_id, current_lesson, state, created_at, updated_at 
		 FROM user_course_progress WHERE user_id = $1 AND course_id = $2`,
		userID, courseID,
	).Scan(&progress.ID, &progress.UserID, &progress.CourseID, &progress.CurrentLessonID, &stateJSON, &progress.CreatedAt, &progress.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return UserCourseProgress{}, errors.Trace(ErrUserProgressNotFound)
	}
	if err != nil {
		return UserCourseProgress{}, errors.Trace(err)
	}

	progress.State = &ProgressState{}
	if err := json.Unmarshal(stateJSON, progress.State); err != nil {
		return UserCourseProgress{}, errors.Trace(err)
	}

	return progress, nil
}

// EnrollUserInCourse enrolls a user in a course
func (r *pgRepository) EnrollUserInCourse(ctx context.Context, userID uuid.UUID, courseID uuid.UUID, progressState ProgressState) error {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM user_course_progress WHERE user_id = $1 AND course_id = $2)`,
		userID, courseID,
	).Scan(&exists)
	if err != nil {
		return errors.Trace(err)
	}

	if exists {
		return errors.Trace(ErrUserAlreadyEnrolled)
	}

	initialState, err := json.Marshal(progressState)
	if err != nil {
		return errors.Trace(err)
	}

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO user_course_progress (id, user_id, course_id, current_lesson, state, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uuid.New(), userID, courseID, progressState.CurrentLessonID, initialState, time.Now().UTC(), time.Now().UTC(),
	)

	return errors.Trace(err)
}

// UpdateUserProgress updates a user's progress in a course
func (r *pgRepository) UpdateUserProgress(ctx context.Context, progress UserCourseProgress) error {
	stateJSON, err := json.Marshal(progress.State)
	if err != nil {
		return errors.Trace(err)
	}

	_, err = r.db.ExecContext(ctx,
		`UPDATE user_course_progress SET state = $1, current_lesson = $2, updated_at = $3 
		 WHERE user_id = $4 AND course_id = $5`,
		stateJSON, progress.CurrentLessonID, time.Now().UTC(), progress.UserID, progress.CourseID,
	)

	return errors.Trace(err)
}
