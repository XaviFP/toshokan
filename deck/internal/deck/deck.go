package deck

import (
	"github.com/google/uuid"
)

// Card kind constants
const (
	CardKindSingleChoice    = "single_choice"
	CardKindFillInTheBlanks = "fill_in_the_blanks"
)

type Deck struct {
	ID          uuid.UUID `json:"id,omitempty"`
	AuthorID    uuid.UUID `json:"authorId,omitempty"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Cards       []Card    `json:"cards"`
	Public      bool      `json:"isPublic"`
}

func (d *Deck) GenerateUUIDs() {
	d.ID = uuid.New()

	for i := 0; i < len(d.Cards); i++ {
		d.Cards[i].GenerateUUIDs()
	}
}

func (c *Card) GenerateUUIDs() {
	c.ID = uuid.New()

	for i := 0; i < len(c.PossibleAnswers); i++ {
		c.PossibleAnswers[i].ID = uuid.New()
	}
}

type Card struct {
	ID              uuid.UUID `json:"id,omitempty"`
	Title           string    `json:"title"`
	PossibleAnswers []Answer  `json:"possibleAnswers"`
	Explanation     string    `json:"explanation"`
	Kind            string    `json:"kind"`
}

type Answer struct {
	ID        uuid.UUID `json:"id,omitempty"`
	Text      string    `json:"text"`
	IsCorrect bool      `json:"isCorrect"`
}

// DeckUpdates contains optional fields for updating a deck
type DeckUpdates struct {
	Title       *string
	Description *string
	IsPublic    *bool
}

// HasUpdates returns true if at least one field is set
func (u DeckUpdates) HasUpdates() bool {
	return u.Title != nil || u.Description != nil || u.IsPublic != nil
}

// CardUpdates contains optional fields for updating a card
type CardUpdates struct {
	Title       *string
	Explanation *string
	Kind        *string
}

// HasUpdates returns true if at least one field is set
func (u CardUpdates) HasUpdates() bool {
	return u.Title != nil || u.Explanation != nil || u.Kind != nil
}

// AnswerUpdates contains optional fields for updating an answer
type AnswerUpdates struct {
	Text      *string
	IsCorrect *bool
}

// HasUpdates returns true if at least one field is set
func (u AnswerUpdates) HasUpdates() bool {
	return u.Text != nil || u.IsCorrect != nil
}

type ErroredDeck struct {
	D            Deck          `json:"deck"`
	ErroredCards []ErroredCard `json:"erroredCards"`
	Errors       []string      `json:"errors"`
	Errs         []error       `json:"-"`
}

type ErroredCard struct {
	C      Card     `json:"card"`
	Errors []string `json:"errors"`
	Errs   []error  `json:"-"`
}
