package deck

import (
	"github.com/google/uuid"
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
}

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
