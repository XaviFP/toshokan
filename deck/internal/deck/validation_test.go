package deck

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidation_ValidateDecks(t *testing.T) {
	var testDecks []Deck = []Deck{
		{Title: "Go Learning", Description: "Polish your Go skills", Cards: []Card{
			{
				Title: "What does CSP stand for?",
				PossibleAnswers: []Answer{
					{Text: "Communicating Sequential Processes", IsCorrect: true},
				}},
			{
				Title: "Which is the underlying data type of a slice in Go?",
				PossibleAnswers: []Answer{
					{Text: "Map", IsCorrect: false},
					{Text: "Linked list", IsCorrect: false},
					{Text: "Array", IsCorrect: true},
				}},
		}},
		{Description: "Polish your Go skills"},
		{Title: "Go Learning"},
		{},
	}

	isValid, erroredDecks := ValidateDecks(testDecks)
	assert.False(t, isValid)
	assert.Equal(t, []error{ErrNoTitle}, erroredDecks[0].Errs)
	assert.Equal(t, []error{ErrNoDescription}, erroredDecks[1].Errs)
	assert.Equal(t, []error{ErrNoTitle, ErrNoDescription}, erroredDecks[2].Errs)
}

func TestValidation_ValidateCards(t *testing.T) {
	testCards := []Card{
		{
			Title: "Which is the underlying data type of a slice in Go?",
			Kind:  CardKindSingleChoice,
			PossibleAnswers: []Answer{
				{Text: "Map", IsCorrect: false},
				{Text: "Linked list", IsCorrect: false},
				{Text: "Array", IsCorrect: true},
			},
		},
		{
			Title: "What does CSP stand for?",
			Kind:  CardKindFillInTheBlanks,
			PossibleAnswers: []Answer{
				{Text: "Communicating Sequential Processes", IsCorrect: false},
			},
		},
		{
			Kind: CardKindSingleChoice,
			PossibleAnswers: []Answer{
				{Text: "Communicating Sequential Processes", IsCorrect: true},
			},
		},
		{
			Title: "What does CSP stand for?",
			Kind:  CardKindSingleChoice,
		},
		{},
		{
			Title: "Invalid kind test",
			Kind:  "invalid_kind",
			PossibleAnswers: []Answer{
				{Text: "Answer", IsCorrect: true},
			},
		},
	}

	isValid, erroredCards := ValidateCards(testCards)
	assert.False(t, isValid)
	assert.Equal(t, []error{ErrNoCorrectAnswer}, erroredCards[0].Errs)
	assert.Equal(t, []error{ErrNoTitle}, erroredCards[1].Errs)
	assert.Equal(t, []error{ErrNoAnswersProvided, ErrNoCorrectAnswer}, erroredCards[2].Errs)
	assert.Equal(t, []error{ErrNoTitle, ErrInvalidKind, ErrNoAnswersProvided, ErrNoCorrectAnswer}, erroredCards[3].Errs)
	assert.Equal(t, []error{ErrInvalidKind}, erroredCards[4].Errs)
}
