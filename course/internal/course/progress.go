package course

import (
	"time"

	"github.com/google/uuid"
	"github.com/juju/errors"
)

var (
	ErrUnitProgressStateNotInitialized = errors.New("course: progress state not initialized")
)

type ProgressState struct {
	Lessons         map[string]*LessonProgress `json:"lessons"`
	CurrentLessonID uuid.UUID                  `json:"current_lesson_id,omitempty"`
}

type LessonProgress struct {
	Decks       map[string]*DeckProgress `json:"decks"`
	IsCompleted bool                     `json:"is_completed"`
	CompletedAt *time.Time               `json:"completed_at,omitempty"`
}

type DeckProgress struct {
	Cards       map[string]*CardProgress `json:"cards"`
	IsCompleted bool                     `json:"is_completed"`
	CompletedAt *time.Time               `json:"completed_at,omitempty"`
}

type CardProgress struct {
	CorrectAnswers   int        `json:"correct_answers"`
	IncorrectAnswers int        `json:"incorrect_answers"`
	IsCompleted      bool       `json:"is_completed"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
}

// NewProgressState creates a new empty progress state
func NewProgressState() *ProgressState {
	return &ProgressState{
		Lessons: make(map[string]*LessonProgress),
	}
}

// AnswerCard marks a card as answered and tracks correctness.
// Returns an error if the lesson, deck, or card do not exist in the progress state.
func (ps *ProgressState) AnswerCard(lessonID, deckID, cardID string, correct bool) error {
	if ps.Lessons == nil {
		return errors.New("progress state not initialized")
	}

	lesson, exists := ps.Lessons[lessonID]
	if !exists {
		return errors.Errorf("lesson %s not found in progress state", lessonID)
	}

	deck, exists := lesson.Decks[deckID]
	if !exists {
		return errors.Errorf("deck %s not found in lesson %s", deckID, lessonID)
	}

	card, exists := deck.Cards[cardID]
	if !exists {
		return errors.Errorf("card %s not found in state's deck %s", cardID, deckID)
	}

	if correct {
		card.CorrectAnswers++
		// Card is completed once answered correctly at least once
		if !card.IsCompleted {
			card.IsCompleted = true
			now := time.Now()
			card.CompletedAt = &now

			// Check if all cards in the deck are now completed
			allCardsComplete := true
			for _, c := range deck.Cards {
				if !c.IsCompleted {
					allCardsComplete = false
					break
				}
			}

			deck.IsCompleted = allCardsComplete
			if allCardsComplete {
				now := time.Now()
				deck.CompletedAt = &now
			}

			// Check if all decks in the lesson are now completed
			allDecksComplete := true
			for _, d := range lesson.Decks {
				if !d.IsCompleted {
					allDecksComplete = false
					break
				}
			}

			lesson.IsCompleted = allDecksComplete
			if allDecksComplete {
				now := time.Now()
				lesson.CompletedAt = &now
				// TODO Move To Next Lesson?
				// Currently Answerer and Syncher take care of this
			}
		}
	} else {
		card.IncorrectAnswers++
	}

	return nil
}

// IsCardAllAnswersCorrect checks if all answers for a card are correct
func (ps *ProgressState) IsCardAllAnswersCorrect(lessonID, deckID, cardID string) bool {
	if ps.Lessons == nil ||
		ps.Lessons[lessonID] == nil ||
		ps.Lessons[lessonID].Decks[deckID] == nil ||
		ps.Lessons[lessonID].Decks[deckID].Cards[cardID] == nil {
		return false
	}

	return ps.Lessons[lessonID].Decks[deckID].Cards[cardID].IsCompleted
}

// IsDeckCompleted checks if a deck is completed based on state
// A deck is considered completed if all tracked cards for that deck are marked completed.
func (ps *ProgressState) IsDeckCompleted(lessonID, deckID string) bool {
	if ps.Lessons == nil ||
		ps.Lessons[lessonID] == nil ||
		ps.Lessons[lessonID].Decks[deckID] == nil {
		return false
	}

	deck := ps.Lessons[lessonID].Decks[deckID]

	// Check if all tracked cards are completed
	if len(deck.Cards) == 0 {
		return false
	}
	for _, card := range deck.Cards {
		if !card.IsCompleted {
			return false
		}
	}
	return true
}

// IsLessonCompleted checks if all decks in a lesson are completed
func (ps *ProgressState) IsLessonCompleted(lessonID string) bool {
	if ps.Lessons == nil || ps.Lessons[lessonID] == nil {
		return false
	}

	lesson := ps.Lessons[lessonID]
	// Check if lesson is already marked completed
	if lesson.IsCompleted {
		return true
	}

	// Check if all decks are completed
	for deckID := range lesson.Decks {
		if !ps.IsDeckCompleted(lessonID, deckID) {
			return false
		}
	}

	lesson.IsCompleted = true
	now := time.Now()
	lesson.CompletedAt = &now

	return true
}

type LessonState map[string]LessonProgress

// GetLessonState returns the lesson progress as a map with lessonID as key
func (ps *ProgressState) GetLessonState(lessonID string) LessonState {
	if ps.Lessons == nil || ps.Lessons[lessonID] == nil {
		return LessonState{}
	}

	return LessonState{
		lessonID: *ps.Lessons[lessonID],
	}
}
