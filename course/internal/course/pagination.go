package course

import (
	"github.com/XaviFP/toshokan/common/pagination"
)

type LessonCursor struct {
	Order int `json:"order"`
}

type LessonEdge struct {
	Lesson Lesson
	Cursor pagination.Cursor
}

type LessonsConnection struct {
	Edges    []LessonEdge
	PageInfo pagination.PageInfo
}

type LessonWithProgress struct {
	Lesson      Lesson
	IsCompleted bool
	IsCurrent   bool
}

type LessonWithProgressEdge struct {
	Lesson *LessonWithProgress
	Cursor pagination.Cursor
}

type LessonsWithProgressConnection struct {
	Edges    []LessonWithProgressEdge
	PageInfo pagination.PageInfo
}

// Course pagination types
type CourseCursor struct {
	Order int64 `json:"order"`
}

type CourseEdge struct {
	Course Course
	Cursor pagination.Cursor
}

type CoursesConnection struct {
	Edges    []CourseEdge
	PageInfo pagination.PageInfo
}

type CourseWithProgress struct {
	Course          Course
	CurrentLessonID string // UUID as string
}

type CourseWithProgressEdge struct {
	Course *CourseWithProgress
	Cursor pagination.Cursor
}

type CoursesWithProgressConnection struct {
	Edges    []CourseWithProgressEdge
	PageInfo pagination.PageInfo
}
