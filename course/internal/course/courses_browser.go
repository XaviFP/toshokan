package course

import (
	"context"

	"github.com/google/uuid"
	"github.com/juju/errors"

	"github.com/XaviFP/toshokan/common/pagination"
)

// CoursesBrowser handles fetching enrolled courses with user progress
type CoursesBrowser interface {
	BrowseEnrolled(ctx context.Context, userID uuid.UUID, p pagination.Pagination) (CoursesWithProgressConnection, error)
}

type coursesBrowser struct {
	repo Repository
}

func NewCoursesBrowser(repo Repository) *coursesBrowser {
	return &coursesBrowser{
		repo: repo,
	}
}

// BrowseEnrolled fetches enrolled courses with progress enrichment
func (b *coursesBrowser) BrowseEnrolled(ctx context.Context, userID uuid.UUID, p pagination.Pagination) (CoursesWithProgressConnection, error) {
	// Fetch enrolled courses with progress from repository
	conn, err := b.repo.GetEnrolledCourses(ctx, userID, p)
	if err != nil {
		return CoursesWithProgressConnection{}, errors.Trace(err)
	}

	return conn, nil
}
