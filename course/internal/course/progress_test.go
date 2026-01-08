package course

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestState holds a pre-configured progress state for testing
type TestState struct {
	State             *ProgressState
	Lesson1ID         string
	Lesson2ID         string
	Lesson3ID         string
	Lesson1Deck1ID    string
	Lesson1Deck2ID    string
	Lesson2Deck1ID    string
	Lesson2Deck2ID    string
	Lesson3Deck1ID    string
	Lesson1Deck1Cards []string
	Lesson1Deck2Cards []string
	Lesson2Deck1Cards []string
	Lesson2Deck2Cards []string
	Lesson3Deck1Cards []string
}

// newTestState creates a fully-featured progress state for testing with 3 lessons,
// 5 decks total, and multiple cards per deck. All objects are set to not completed.
func newTestState(t *testing.T) *TestState {
	state := NewProgressState()

	ts := &TestState{
		State:     state,
		Lesson1ID: uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72").String(),
		Lesson2ID: uuid.MustParse("6363e2c6-d89e-4610-92e8-1e1d2fea49ec").String(),
		Lesson3ID: uuid.MustParse("f79aea77-9aa0-4a84-b4c8-d000a27d2c52").String(),

		Lesson1Deck1ID: uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c").String(),
		Lesson1Deck2ID: uuid.MustParse("60766223-ff9f-4871-a497-f765c05a0c5e").String(),
		Lesson2Deck1ID: uuid.MustParse("dfcb1c81-f590-486e-9b7e-a44f0c436933").String(),
		Lesson2Deck2ID: uuid.MustParse("06be1892-4765-4f60-9d47-1489419dc316").String(),
		Lesson3Deck1ID: uuid.MustParse("a0f3a2e4-7263-855d-ebcf-fffa0a96f451").String(),

		Lesson1Deck1Cards: []string{
			uuid.MustParse("72bdff92-5bc8-4e1d-9217-d0b23e22ff33").String(),
			uuid.MustParse("c924f7e0-efd8-4c2d-9c43-8eafb7102ebc").String(),
		},
		Lesson1Deck2Cards: []string{
			uuid.MustParse("d42a90dd-818c-4eed-8e9f-9e8af1a654f4").String(),
			uuid.MustParse("7e6926da-82b2-4ae8-99b4-1b803ebf1877").String(),
		},
		Lesson2Deck1Cards: []string{
			uuid.MustParse("9403ad3e-45e6-4b23-8f63-b751de8576cc").String(),
			uuid.MustParse("3b1bbdb3-b84a-4f59-8f02-2a21586cf6ca").String(),
		},
		Lesson2Deck2Cards: []string{
			uuid.MustParse("d23d0201-55f3-40da-8718-853a6cea419d").String(),
			uuid.MustParse("a0f3a2e4-7263-855d-ebcf-fffa0a96f450").String(),
		},
		Lesson3Deck1Cards: []string{
			uuid.MustParse("a0f3a2e4-7263-855d-ebcf-fffa0a96f452").String(),
			uuid.MustParse("a0f3a2e4-7263-855d-ebcf-fffa0a96f453").String(),
		},
	}

	// Initialize all lessons, decks, and cards in the state
	initializeProgressState(t, state, ts.Lesson1ID, ts.Lesson1Deck1ID, ts.Lesson1Deck1Cards)
	initializeProgressState(t, state, ts.Lesson1ID, ts.Lesson1Deck2ID, ts.Lesson1Deck2Cards)
	initializeProgressState(t, state, ts.Lesson2ID, ts.Lesson2Deck1ID, ts.Lesson2Deck1Cards)
	initializeProgressState(t, state, ts.Lesson2ID, ts.Lesson2Deck2ID, ts.Lesson2Deck2Cards)
	initializeProgressState(t, state, ts.Lesson3ID, ts.Lesson3Deck1ID, ts.Lesson3Deck1Cards)

	return ts
}

// initializeProgressState adds a lesson, deck, and its cards to the progress state
// This simulates what the enroller does when a user enrolls in a course
func initializeProgressState(t *testing.T, state *ProgressState, lessonID, deckID string, cardIDs []string) {
	t.Helper()

	if state.Lessons[lessonID] == nil {
		state.Lessons[lessonID] = &LessonProgress{
			Decks:       make(map[string]*DeckProgress),
			IsCompleted: false,
		}
	}

	state.Lessons[lessonID].Decks[deckID] = &DeckProgress{
		Cards:       make(map[string]*CardProgress),
		IsCompleted: false,
	}

	for _, cardID := range cardIDs {
		state.Lessons[lessonID].Decks[deckID].Cards[cardID] = &CardProgress{
			CorrectAnswers:   0,
			IncorrectAnswers: 0,
			IsCompleted:      false,
		}
	}
}

func TestProgressState_NewProgressState(t *testing.T) {
	state := NewProgressState()
	assert.NotNil(t, state)
	assert.NotNil(t, state.Lessons)
	assert.Empty(t, state.Lessons)
}

func TestProgressState_AnswerCard(t *testing.T) {
	ts := newTestState(t)

	t.Run("correct_answer", func(t *testing.T) {
		err := ts.State.AnswerCard(ts.Lesson1ID, ts.Lesson1Deck1ID, ts.Lesson1Deck1Cards[0], true)
		assert.NoError(t, err)

		assert.True(t, ts.State.IsCardAllAnswersCorrect(ts.Lesson1ID, ts.Lesson1Deck1ID, ts.Lesson1Deck1Cards[0]))

		// Verify the card is marked completed
		lesson := ts.State.Lessons[ts.Lesson1ID]
		assert.NotNil(t, lesson)
		deck := lesson.Decks[ts.Lesson1Deck1ID]
		assert.NotNil(t, deck)
		card := deck.Cards[ts.Lesson1Deck1Cards[0]]
		assert.NotNil(t, card)
		assert.True(t, card.IsCompleted)
		assert.Equal(t, 1, card.CorrectAnswers)
		assert.Equal(t, 0, card.IncorrectAnswers)
	})

	t.Run("incorrect_answer", func(t *testing.T) {
		err := ts.State.AnswerCard(ts.Lesson1ID, ts.Lesson1Deck1ID, ts.Lesson1Deck1Cards[1], false)
		assert.NoError(t, err)

		assert.False(t, ts.State.IsCardAllAnswersCorrect(ts.Lesson1ID, ts.Lesson1Deck1ID, ts.Lesson1Deck1Cards[1]))

		card := ts.State.Lessons[ts.Lesson1ID].Decks[ts.Lesson1Deck1ID].Cards[ts.Lesson1Deck1Cards[1]]
		assert.False(t, card.IsCompleted)
		assert.Equal(t, 0, card.CorrectAnswers)
		assert.Equal(t, 1, card.IncorrectAnswers)
	})

	t.Run("lesson_not_found", func(t *testing.T) {
		unknownLessonID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff").String()
		err := ts.State.AnswerCard(unknownLessonID, ts.Lesson1Deck1ID, ts.Lesson1Deck1Cards[0], true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "lesson")
	})

	t.Run("deck_not_found", func(t *testing.T) {
		unknownDeckID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff").String()
		err := ts.State.AnswerCard(ts.Lesson1ID, unknownDeckID, ts.Lesson1Deck1Cards[0], true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "deck")
	})

	t.Run("card_not_found", func(t *testing.T) {
		unknownCardID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff").String()
		err := ts.State.AnswerCard(ts.Lesson1ID, ts.Lesson1Deck1ID, unknownCardID, true)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "card")
	})
}

func TestProgressState_IsDeckComplete(t *testing.T) {
	ts := newTestState(t)

	t.Run("not_started", func(t *testing.T) {
		assert.False(t, ts.State.IsDeckCompleted(ts.Lesson1ID, ts.Lesson1Deck1ID))
	})

	t.Run("partially_complete", func(t *testing.T) {
		// Mark first card completed
		err := ts.State.AnswerCard(ts.Lesson1ID, ts.Lesson1Deck1ID, ts.Lesson1Deck1Cards[0], true)
		assert.NoError(t, err)

		// Deck is not completed yet - card 2 is not completed
		assert.False(t, ts.State.IsDeckCompleted(ts.Lesson1ID, ts.Lesson1Deck1ID))

		// Mark second card with wrong answer
		err = ts.State.AnswerCard(ts.Lesson1ID, ts.Lesson1Deck1ID, ts.Lesson1Deck1Cards[1], false)
		assert.NoError(t, err)

		// Deck is still not completed - card 2 hasn't been answered correctly
		assert.False(t, ts.State.IsDeckCompleted(ts.Lesson1ID, ts.Lesson1Deck1ID))
	})
}

func TestProgressState_IsCardAllAnswersCorrect(t *testing.T) {
	ts := newTestState(t)

	t.Run("not_started", func(t *testing.T) {
		assert.False(t, ts.State.IsCardAllAnswersCorrect(ts.Lesson1ID, ts.Lesson1Deck1ID, ts.Lesson1Deck1Cards[0]))
	})

	t.Run("answered_correctly", func(t *testing.T) {
		err := ts.State.AnswerCard(ts.Lesson1ID, ts.Lesson1Deck1ID, ts.Lesson1Deck1Cards[0], true)
		assert.NoError(t, err)
		assert.True(t, ts.State.IsCardAllAnswersCorrect(ts.Lesson1ID, ts.Lesson1Deck1ID, ts.Lesson1Deck1Cards[0]))
	})

	t.Run("answered_incorrectly", func(t *testing.T) {
		err := ts.State.AnswerCard(ts.Lesson1ID, ts.Lesson1Deck1ID, ts.Lesson1Deck1Cards[1], false)
		assert.NoError(t, err)
		assert.False(t, ts.State.IsCardAllAnswersCorrect(ts.Lesson1ID, ts.Lesson1Deck1ID, ts.Lesson1Deck1Cards[1]))
	})
}

func TestProgressState_FullCourseProgression(t *testing.T) {
	ts := newTestState(t)

	// === LESSON 1 PROGRESSION ===
	// User starts lesson 1 with empty progress
	assert.False(t, ts.State.IsDeckCompleted(ts.Lesson1ID, ts.Lesson1Deck1ID))
	assert.False(t, ts.State.IsDeckCompleted(ts.Lesson1ID, ts.Lesson1Deck2ID))

	// User completes all cards in deck 1
	for _, cardID := range ts.Lesson1Deck1Cards {
		err := ts.State.AnswerCard(ts.Lesson1ID, ts.Lesson1Deck1ID, cardID, true)
		assert.NoError(t, err)
	}

	assert.True(t, ts.State.IsDeckCompleted(ts.Lesson1ID, ts.Lesson1Deck1ID))
	assert.False(t, ts.State.IsDeckCompleted(ts.Lesson1ID, ts.Lesson1Deck2ID))

	// User completes all cards in deck 2
	for _, cardID := range ts.Lesson1Deck2Cards {
		err := ts.State.AnswerCard(ts.Lesson1ID, ts.Lesson1Deck2ID, cardID, true)
		assert.NoError(t, err)
	}

	assert.True(t, ts.State.IsDeckCompleted(ts.Lesson1ID, ts.Lesson1Deck1ID))
	assert.True(t, ts.State.IsDeckCompleted(ts.Lesson1ID, ts.Lesson1Deck2ID))

	// Lesson 1 should be complete
	assert.True(t, ts.State.IsLessonCompleted(ts.Lesson1ID))

	// === LESSON 2 PROGRESSION ===
	// User starts lesson 2 - lesson 1 should remain complete
	assert.True(t, ts.State.IsLessonCompleted(ts.Lesson1ID))

	assert.False(t, ts.State.IsDeckCompleted(ts.Lesson2ID, ts.Lesson2Deck1ID))
	assert.False(t, ts.State.IsDeckCompleted(ts.Lesson2ID, ts.Lesson2Deck2ID))

	// User gets some cards wrong initially, then corrects them
	err := ts.State.AnswerCard(ts.Lesson2ID, ts.Lesson2Deck1ID, ts.Lesson2Deck1Cards[0], false)
	assert.NoError(t, err)
	err = ts.State.AnswerCard(ts.Lesson2ID, ts.Lesson2Deck1ID, ts.Lesson2Deck1Cards[0], true) // Correct on retry
	assert.NoError(t, err)

	// Complete first deck
	for _, cardID := range ts.Lesson2Deck1Cards {
		err := ts.State.AnswerCard(ts.Lesson2ID, ts.Lesson2Deck1ID, cardID, true)
		assert.NoError(t, err)
	}

	assert.True(t, ts.State.IsDeckCompleted(ts.Lesson2ID, ts.Lesson2Deck1ID))

	// Complete second deck
	for _, cardID := range ts.Lesson2Deck2Cards {
		err := ts.State.AnswerCard(ts.Lesson2ID, ts.Lesson2Deck2ID, cardID, true)
		assert.NoError(t, err)
	}

	assert.True(t, ts.State.IsDeckCompleted(ts.Lesson2ID, ts.Lesson2Deck2ID))

	// Lesson 2 should be complete
	assert.True(t, ts.State.IsLessonCompleted(ts.Lesson2ID))

	// === LESSON 3 FINAL PROGRESSION ===
	// Verify previous lessons remain complete
	assert.True(t, ts.State.IsLessonCompleted(ts.Lesson1ID))
	assert.True(t, ts.State.IsLessonCompleted(ts.Lesson2ID))

	// User completes final lesson
	assert.False(t, ts.State.IsDeckCompleted(ts.Lesson3ID, ts.Lesson3Deck1ID))

	for _, cardID := range ts.Lesson3Deck1Cards {
		err := ts.State.AnswerCard(ts.Lesson3ID, ts.Lesson3Deck1ID, cardID, true)
		assert.NoError(t, err)
	}

	assert.True(t, ts.State.IsDeckCompleted(ts.Lesson3ID, ts.Lesson3Deck1ID))

	assert.True(t, ts.State.IsLessonCompleted(ts.Lesson3ID))

	// === FINAL STATE VALIDATION ===
	// Verify all lessons remain complete
	assert.True(t, ts.State.IsLessonCompleted(ts.Lesson1ID))
	assert.True(t, ts.State.IsLessonCompleted(ts.Lesson2ID))
	assert.True(t, ts.State.IsLessonCompleted(ts.Lesson3ID))

	// Verify state structure
	assert.NotNil(t, ts.State.Lessons)
	assert.Equal(t, 3, len(ts.State.Lessons))

	// Lesson 1: 2 decks with 2 cards each, marked complete
	lesson1 := ts.State.Lessons[ts.Lesson1ID]
	assert.NotNil(t, lesson1)
	assert.Equal(t, 2, len(lesson1.Decks))
	assert.True(t, lesson1.IsCompleted)
	assert.NotNil(t, lesson1.CompletedAt)

	deck1_1 := lesson1.Decks[ts.Lesson1Deck1ID]
	assert.NotNil(t, deck1_1)
	assert.Equal(t, 2, len(deck1_1.Cards))

	// Lesson 3: 1 deck with 2 cards, should be complete when all cards are done
	lesson3 := ts.State.Lessons[ts.Lesson3ID]
	assert.NotNil(t, lesson3, "lesson3 should exist in state.Lessons")
	assert.Equal(t, 1, len(lesson3.Decks))
	assert.True(t, lesson3.IsCompleted)
	assert.NotNil(t, lesson3.CompletedAt)

	deck3 := lesson3.Decks[ts.Lesson3Deck1ID]
	assert.NotNil(t, deck3)
	assert.Equal(t, 2, len(deck3.Cards))

	// Verify all cards in lesson 3 are marked complete
	for _, card := range deck3.Cards {
		assert.True(t, card.IsCompleted)
		assert.NotNil(t, card.CompletedAt)
		assert.Greater(t, card.CorrectAnswers, 0)
	}
}

func TestProgressState_GetLessonState(t *testing.T) {
	ts := newTestState(t)

	t.Run("lesson_exists", func(t *testing.T) {
		lessonState := ts.State.GetLessonState(ts.Lesson1ID)
		assert.NotNil(t, lessonState)
		assert.Equal(t, ts.Lesson1ID, ts.Lesson1ID)
	})

	t.Run("lesson_not_found", func(t *testing.T) {
		unknownLessonID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff").String()
		lessonState := ts.State.GetLessonState(unknownLessonID)
		assert.Empty(t, lessonState)
	})

}
