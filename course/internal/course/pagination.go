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
