package deck

import (
	"github.com/google/uuid"
)

// Deck represents a set of cards meant to go together
type Deck struct {
	ID          uuid.UUID `json:"id,omitempty"`
	AuthorID    uuid.UUID `json:"authorId,omitempty"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Cards       []Card    `json:"cards"`
	Public      bool      `jason:"isPublic"`
}

func (d *Deck) GenerateUUIDs() {
	d.ID = uuid.New()

	for i := 0; i < len(d.Cards); i++ {
		d.Cards[i].ID = uuid.New()

		for j := 0; j < len(d.Cards[i].PossibleAnswers); j++ {
			d.Cards[i].PossibleAnswers[j].ID = uuid.New()
		}
	}
}

// Card represents a card with its set of answers including the correct(s) one(s).
type Card struct {
	ID              uuid.UUID `json:"id,omitempty"`
	Title           string    `json:"title"`
	PossibleAnswers []Answer  `json:"possibleAnswers"`
}

// Answer represents one of the possible options to reply a Card
type Answer struct {
	ID        uuid.UUID `json:"id,omitempty"`
	Text      string    `json:"text"`
	IsCorrect bool      `json:"isCorrect"`
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
