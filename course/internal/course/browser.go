package course

import (
	"context"

	"github.com/google/uuid"
	"github.com/juju/errors"

	"github.com/XaviFP/toshokan/common/pagination"
)

// LessonsBrowser handles fetching and, optionally, enriching lessons with user progress
type LessonsBrowser interface {
	Browse(ctx context.Context, courseID uuid.UUID, p pagination.Pagination, opts BrowseOptions) (BrowseResult, error)
}

type lessonsBrowser struct {
	repo Repository
}

func NewLessonsBrowser(repo Repository) *lessonsBrowser {
	return &lessonsBrowser{
		repo: repo,
	}
}

// BrowseOptions holds optional parameters for browsing lessons
type BrowseOptions struct {
	UserID *uuid.UUID // Optional user ID for progress enrichment
}

// BrowseResult holds the result of browsing lessons
// Depending on whether a user ID is provided, either PublicLessons or ProgressLessons will be populated
type BrowseResult struct {
	PublicLessons   *LessonsConnection
	ProgressLessons *LessonsWithProgressConnection
}

// Browse fetches lessons with optional user progress enrichment
func (b *lessonsBrowser) Browse(ctx context.Context, courseID uuid.UUID, p pagination.Pagination, opts BrowseOptions) (BrowseResult, error) {
	result := BrowseResult{}

	// If no user option, return public lessons
	if opts.UserID == nil {
		conn, err := b.repo.GetLessonsByCourseID(ctx, courseID, p)
		if err != nil {
			return result, errors.Trace(err)
		}
		result.PublicLessons = &conn
		return result, nil
	}

	// Fetch lessons and enrich with user progress
	conn, err := b.repo.GetLessonsByCourseID(ctx, courseID, p)
	if err != nil {
		return result, errors.Trace(err)
	}

	progress, err := b.repo.GetUserCourseProgress(ctx, *opts.UserID, courseID)
	if err != nil {
		return result, errors.Trace(err)
	}

	enrichedConn := b.enrichLessonsWithProgress(conn, progress)
	result.ProgressLessons = &enrichedConn

	return result, nil
}

// enrichLessonsWithProgress merges lessons with their completion status from progress state
func (b *lessonsBrowser) enrichLessonsWithProgress(
	conn LessonsConnection,
	progress UserCourseProgress,
) LessonsWithProgressConnection {
	enriched := LessonsWithProgressConnection{
		PageInfo: conn.PageInfo,
	}

	for _, edge := range conn.Edges {
		isCompleted := progress.State.IsLessonCompleted(edge.Lesson.ID.String())
		isCurrent := progress.CurrentLessonID == edge.Lesson.ID

		lessonWithProgress := &LessonWithProgress{
			Lesson:      edge.Lesson,
			IsCompleted: isCompleted,
			IsCurrent:   isCurrent,
		}

		enriched.Edges = append(enriched.Edges, LessonWithProgressEdge{
			Lesson: lessonWithProgress,
			Cursor: edge.Cursor,
		})
	}

	return enriched
}
