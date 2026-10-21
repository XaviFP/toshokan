package deck

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeck_GenerateUUIDs(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		d := Deck{
			Cards: []Card{
				{Title: "Golang", PossibleAnswers: []Answer{{Text: "Compiled"}, {Text: "Interpreted"}}},
				{Title: "Rust", PossibleAnswers: []Answer{{Text: "Compiled"}, {Text: "Interpreted"}}},
			},
		}

		d.GenerateUUIDs()

		require.Len(t, d.Cards, 2)
		require.Len(t, d.Cards[0].PossibleAnswers, 2)
		require.Len(t, d.Cards[1].PossibleAnswers, 2)

		assert.Equal(t, d.ID, d.Cards[0].DeckID)
		assert.Equal(t, d.ID, d.Cards[1].DeckID)

		assert.Equal(t, d.Cards[0].ID, d.Cards[0].PossibleAnswers[0].CardID)
		assert.Equal(t, d.Cards[0].ID, d.Cards[0].PossibleAnswers[1].CardID)
		assert.Equal(t, d.Cards[1].ID, d.Cards[1].PossibleAnswers[0].CardID)
		assert.Equal(t, d.Cards[1].ID, d.Cards[1].PossibleAnswers[1].CardID)
	})
}
