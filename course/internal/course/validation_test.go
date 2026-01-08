package course

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewValidationErrors(t *testing.T) {
	ve := NewValidationErrors()

	assert.NotNil(t, ve)
	assert.NotNil(t, ve.ErrorKeys)
	assert.Equal(t, 0, len(ve.ErrorKeys))
	assert.False(t, ve.HasErrors())
}

func TestValidationErrors_Add(t *testing.T) {
	ve := NewValidationErrors()

	ve.Add(ErrorKeyCourseNotFound)
	assert.Equal(t, 1, len(ve.ErrorKeys))
	assert.Equal(t, ErrorKeyCourseNotFound, ve.ErrorKeys[0])

	ve.Add(ErrorKeyLessonNotFound)
	assert.Equal(t, 2, len(ve.ErrorKeys))
	assert.Equal(t, ErrorKeyLessonNotFound, ve.ErrorKeys[1])

	ve.Add(ErrorKeyNoTitle)
	assert.Equal(t, 3, len(ve.ErrorKeys))
	assert.Equal(t, ErrorKeyNoTitle, ve.ErrorKeys[2])
}

func TestValidationErrors_HasErrors_Empty(t *testing.T) {
	ve := NewValidationErrors()

	assert.False(t, ve.HasErrors())
}

func TestValidationErrors_HasErrors_WithErrors(t *testing.T) {
	ve := NewValidationErrors()

	ve.Add(ErrorKeyCourseNotFound)
	assert.True(t, ve.HasErrors())

	ve.Add(ErrorKeyLessonNotFound)
	assert.True(t, ve.HasErrors())
}

func TestValidationErrors_Error_Empty(t *testing.T) {
	ve := NewValidationErrors()

	expected := `{"errors":[]}`
	assert.Equal(t, expected, ve.Error())
}

func TestValidationErrors_Error_SingleError(t *testing.T) {
	ve := NewValidationErrors()
	ve.Add(ErrorKeyCourseNotFound)

	errStr := ve.Error()
	assert.JSONEq(t, `{"errors":["COURSE_NOT_FOUND"]}`, errStr)
}

func TestValidationErrors_Error_MultipleErrors(t *testing.T) {
	ve := NewValidationErrors()
	ve.Add(ErrorKeyCourseNotFound)
	ve.Add(ErrorKeyLessonNotFound)
	ve.Add(ErrorKeyNoTitle)

	errStr := ve.Error()
	assert.JSONEq(t, `{"errors":["COURSE_NOT_FOUND","LESSON_NOT_FOUND","NO_TITLE"]}`, errStr)
}
