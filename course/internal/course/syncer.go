package course

import (
	"context"
	"log"

	"github.com/XaviFP/toshokan/common/pagination"
	pbDeck "github.com/XaviFP/toshokan/deck/api/proto/v1"
	"github.com/google/uuid"
	"github.com/juju/errors"
)

type StateSyncer interface {
	Sync(ctx context.Context, userID, courseID uuid.UUID) error
	// AnswerCards()
}

type stateSyncer struct {
	decksClient pbDeck.DecksAPIClient
	repo        Repository
}

func NewStateSyncer(repo Repository, decksClient pbDeck.DecksAPIClient) StateSyncer {
	return &stateSyncer{
		repo:        repo,
		decksClient: decksClient,
	}
}

// Sync synchronizes the user's progress state for the given course
// When a user enrolls to a course, their progress state is initialized with all lessons and decks from the course at that time.
// As the course content may change over time (new lessons/decks added), this method ensures the user's progress state reflects those changes.
// If lessons or decks are removed from the course, they are also removed from the user's progress state.
// Sync sets the current lesson too, pointing to the first incomplete lesson.
func (s *stateSyncer) Sync(ctx context.Context, userID, courseID uuid.UUID) error {
	courseLessons, err := s.repo.GetLessonsByCourseID(ctx, courseID, pagination.NewOlderstFistPagination(pagination.WithFirst(1000))) // TODO: handle pagination
	if err != nil {
		return errors.Trace(err)
	}

	userProgress, err := s.repo.GetUserCourseProgress(ctx, userID, courseID)
	if err != nil {
		return errors.Trace(err)
	}

	state := userProgress.State
	if state == nil {
		return errors.New("courses: user progress state is nil")
	}

	courseLessonsList := make([]Lesson, 0, len(courseLessons.Edges))
	for _, edge := range courseLessons.Edges {
		lesson := edge.Lesson
		courseLessonsList = append(courseLessonsList, lesson)

		if _, exists := state.Lessons[lesson.ID.String()]; !exists {
			if err := s.ensureLessonInState(ctx, lesson, state); err != nil {
				return errors.Trace(err)
			}

			continue
		}

		// Lesson exists in progress state, check for new decks
		parsedDeckIDs := ParseDeckReferences(lesson.Body)
		if len(parsedDeckIDs) == 0 {
			log.Printf("course: no decks found for lesson %s during sync", lesson.ID.String())
			continue
		}

		deckIDs := make([]string, 0, len(parsedDeckIDs))
		for _, dID := range parsedDeckIDs {
			deckIDs = append(deckIDs, dID.String())
		}

		stateLesson := state.Lessons[lesson.ID.String()]

		for _, deckID := range deckIDs {
			res, err := s.decksClient.GetDeck(ctx, &pbDeck.GetDeckRequest{DeckId: deckID})
			if err != nil {
				return errors.Trace(err)
			}
			s.ensureDeckInState(stateLesson, res.Deck)
			for _, card := range res.Deck.Cards {
				s.ensureCardInState(stateLesson, res.Deck.Id, card)
			}
		}
	}

	if err := s.pruneState(ctx, courseLessonsList, state); err != nil {
		return errors.Trace(err)
	}

	// Course lessons are ordered by order already
	// Set current lesson to the first incompleted lesson
	// If all lessons are completed, keep/set last completed lesson as current
	incompletedFound := false
	var lastLessonID uuid.UUID

	for _, edge := range courseLessons.Edges {
		lesson := edge.Lesson
		stateLesson, exists := state.Lessons[lesson.ID.String()]
		if !exists {
			// This should not happen as we initialized missing lessons above
			log.Printf("course: lesson %s not found in user progress state during sync", lesson.ID.String())
			continue
		}

		// set lastLessonID this way as slice[len(slice)-1] might not be available
		// if we switch to paginated fetching
		lastLessonID = lesson.ID

		if !stateLesson.IsCompleted {
			state.CurrentLessonID = lesson.ID
			userProgress.CurrentLessonID = lesson.ID
			incompletedFound = true
			break
		}
	}

	if !incompletedFound {
		// All lessons completed, set current lesson to last lesson
		state.CurrentLessonID = lastLessonID
		userProgress.CurrentLessonID = lastLessonID
	}

	userProgress.State = state

	if err := s.repo.UpdateUserProgress(ctx, userProgress); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// ensureLessonInState makes sure a lesson progress exists in the state and initializes it with all decks and cards.
func (s *stateSyncer) ensureLessonInState(ctx context.Context, lesson Lesson, state *ProgressState) error {
	state.Lessons[lesson.ID.String()] = &LessonProgress{
		Decks:       make(map[string]*DeckProgress),
		IsCompleted: false,
	}

	// Get decks for this lesson
	parsedDeckIDs := ParseDeckReferences(lesson.Body)

	deckIDs := make([]string, 0, len(parsedDeckIDs))
	for _, dID := range parsedDeckIDs {
		deckIDs = append(deckIDs, dID.String())
	}

	if len(deckIDs) == 0 {
		return errors.New("courses: cannot enroll in a course with lessons that have no decks")
	}

	// Initialize progress for each deck and its cards
	stateLesson := state.Lessons[lesson.ID.String()]
	for _, deckID := range deckIDs {
		res, err := s.decksClient.GetDeck(ctx, &pbDeck.GetDeckRequest{DeckId: deckID})
		if err != nil {
			return errors.Trace(err)
		}
		s.ensureDeckInState(stateLesson, res.Deck)
		for _, card := range res.Deck.Cards {
			s.ensureCardInState(stateLesson, res.Deck.Id, card)
		}
	}

	return nil
}

// ensureDeckInState makes sure a deck progress exists for the lesson.
func (s *stateSyncer) ensureDeckInState(stateLesson *LessonProgress, deck *pbDeck.Deck) {
	if stateLesson.Decks == nil {
		stateLesson.Decks = make(map[string]*DeckProgress)
	}

	if _, ok := stateLesson.Decks[deck.Id]; !ok {
		stateLesson.IsCompleted = false
		deckProgress := &DeckProgress{
			Cards:       make(map[string]*CardProgress),
			IsCompleted: false,
		}
		stateLesson.Decks[deck.Id] = deckProgress
	}
}

// ensureCardInState makes sure a card progress exists inside a deck, marking deck/lesson as incomplete when created.
func (s *stateSyncer) ensureCardInState(stateLesson *LessonProgress, deckID string, card *pbDeck.Card) {
	stateDeck, ok := stateLesson.Decks[deckID]
	if !ok {
		log.Printf("course: deck %s not found in lesson during ensureCardInState", deckID)
		return
	}

	if stateDeck.Cards == nil {
		stateDeck.Cards = make(map[string]*CardProgress)
	}

	if _, ok := stateDeck.Cards[card.Id]; ok {
		return
	}

	stateLesson.IsCompleted = false
	stateDeck.IsCompleted = false
	stateDeck.Cards[card.Id] = &CardProgress{IsCompleted: false}
}

// pruneState removes lessons, decks, and cards from the user's state that are no longer present in the course content.
func (s *stateSyncer) pruneState(ctx context.Context, courseLessons []Lesson, state *ProgressState) error {
	lessonByID := make(map[string]Lesson, len(courseLessons))
	for _, lesson := range courseLessons {
		lessonByID[lesson.ID.String()] = lesson
	}

	for lessonID := range state.Lessons {
		if _, exists := lessonByID[lessonID]; !exists {
			delete(state.Lessons, lessonID)
		}
	}

	for _, courseLesson := range courseLessons {
		stateLesson, ok := state.Lessons[courseLesson.ID.String()]
		if !ok {
			log.Printf("course: lesson %s not found in user progress state during prune", courseLesson.ID.String())
			continue
		}

		courseDeckIDs := ParseDeckReferencesAsStrings(courseLesson.Body)
		s.pruneRemovedDecks(stateLesson, courseDeckIDs)

		if len(courseDeckIDs) == 0 {
			log.Printf("course: no decks found for lesson %s during prune", courseLesson.ID.String())
			continue
		}

		for _, deckID := range courseDeckIDs {
			res, err := s.decksClient.GetDeck(ctx, &pbDeck.GetDeckRequest{DeckId: deckID})
			if err != nil {
				return errors.Trace(err)
			}

			stateDeck, ok := stateLesson.Decks[res.Deck.Id]
			if !ok {
				log.Printf("course: deck %s not found in lesson %s during prune", res.Deck.Id, courseLesson.ID.String())
				continue
			}

			s.pruneRemovedCards(stateDeck, res.Deck.Cards)
		}
	}

	return nil
}

// pruneRemovedDecks removes decks that are no longer part of the course content.
func (s *stateSyncer) pruneRemovedDecks(stateLesson *LessonProgress, courseDeckIDs []string) {
	currentDeckIDs := make(map[string]struct{}, len(courseDeckIDs))
	for _, deckID := range courseDeckIDs {
		currentDeckIDs[deckID] = struct{}{}
	}

	for deckID := range stateLesson.Decks {
		if _, exists := currentDeckIDs[deckID]; !exists {
			delete(stateLesson.Decks, deckID)
		}
	}
}

// pruneRemovedCards removes cards that are no longer part of a deck.
func (s *stateSyncer) pruneRemovedCards(stateDeck *DeckProgress, courseCards []*pbDeck.Card) {
	currentCardIDs := make(map[string]struct{}, len(courseCards))
	for _, card := range courseCards {
		currentCardIDs[card.Id] = struct{}{}
	}

	for cardID := range stateDeck.Cards {
		if _, exists := currentCardIDs[cardID]; !exists {
			delete(stateDeck.Cards, cardID)
		}
	}
}
