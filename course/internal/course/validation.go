package course

import (
	"strings"
)

type ErrorKey string

var (
	ErrorKeyCourseNotFound       ErrorKey = "COURSE_NOT_FOUND"
	ErrorKeyLessonNotFound       ErrorKey = "LESSON_NOT_FOUND"
	ErrorKeyUserProgressNotFound ErrorKey = "USER_PROGRESS_NOT_FOUND"
	ErrorKeyInvalidCourse        ErrorKey = "INVALID_COURSE"
	ErrorKeyInvalidLesson        ErrorKey = "INVALID_LESSON"
	ErrorKeyUserAlreadyEnrolled  ErrorKey = "USER_ALREADY_ENROLLED"
	ErrorKeyNoTitle              ErrorKey = "NO_TITLE"
	ErrorKeyNoDescription        ErrorKey = "NO_DESCRIPTION"
	ErrorKeyNoBody               ErrorKey = "NO_BODY"
	ErrorKeyNoDecksReferenced    ErrorKey = "NO_DECKS_REFERENCED"
)

type ValidationErrors struct {
	ErrorKeys []ErrorKey `json:"errors"`
}

func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{
		ErrorKeys: []ErrorKey{},
	}
}

func (ve *ValidationErrors) Add(errKey ErrorKey) {
	ve.ErrorKeys = append(ve.ErrorKeys, errKey)
}

func (ve *ValidationErrors) HasErrors() bool {
	return len(ve.ErrorKeys) > 0
}

func (ve *ValidationErrors) Error() string {
	// json.Marshal is not used here to avoid ignoring its error return value
	errorStrings := make([]string, len(ve.ErrorKeys))
	for i, errKey := range ve.ErrorKeys {
		errorStrings[i] = `"` + string(errKey) + `"`
	}
	return `{"errors":[` + strings.Join(errorStrings, ",") + `]}`
}
