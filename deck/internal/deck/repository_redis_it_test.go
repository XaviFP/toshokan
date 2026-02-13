package deck

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/mediocregopher/radix/v4"
	"github.com/stretchr/testify/assert"

	"github.com/XaviFP/toshokan/common/pagination"
)

func TestRedisRepository_GetDeck_CacheHit(t *testing.T) {
	h := newRedisTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	deckID := uuid.New()
	expectedDeck := Deck{
		ID:          deckID,
		Title:       "Go Learning",
		Description: "Polish your Go skills",
	}

	deckJSON, _ := json.Marshal(expectedDeck)
	key := "cache:deck:" + deckID.String()
	err := h.redisClient.Do(ctx, radix.FlatCmd(nil, "SETEX", key, 3600, string(deckJSON)))
	assert.NoError(t, err)

	result, err := repo.GetDeck(ctx, deckID)
	assert.NoError(t, err)

	assert.Equal(t, expectedDeck.ID, result.ID)
	assert.Equal(t, expectedDeck.Title, result.Title)
	mockDB.AssertNotCalled(t, "GetDeck")
}

func TestRedisRepository_GetDeck_CacheMiss(t *testing.T) {
	h := newRedisTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	deckID := uuid.New()
	expectedDeck := Deck{
		ID:          deckID,
		Title:       "Go Learning",
		Description: "Polish your Go skills",
	}

	mockDB.On("GetDeck", ctx, deckID).Return(expectedDeck, nil)
	mockDB.On("GetDeckCards", ctx, deckID).Return([]Card{}, nil)

	result, err := repo.GetDeck(ctx, deckID)
	assert.NoError(t, err)

	assert.Equal(t, expectedDeck.ID, result.ID)
	assert.Equal(t, expectedDeck.Title, result.Title)

	var cached string
	key := "cache:deck:" + deckID.String()
	mb := radix.Maybe{Rcv: &cached}
	err = h.redisClient.Do(ctx, radix.Cmd(&mb, "GET", key))
	assert.NoError(t, err)

	assert.False(t, mb.Null)
	assert.NotEmpty(t, cached)

	mockDB.AssertExpectations(t)
}

func TestRedisRepository_GetDeck_DBError(t *testing.T) {
	h := newRedisTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	deckID := uuid.New()

	mockDB.On("GetDeck", ctx, deckID).Return(Deck{}, assert.AnError)

	result, err := repo.GetDeck(ctx, deckID)
	assert.ErrorIs(t, err, assert.AnError)

	assert.Empty(t, result.ID)
	mockDB.AssertExpectations(t)
}

func TestRedisRepository_StoreDeck_CachesNewDeck(t *testing.T) {
	h := newRedisTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	deck := Deck{
		ID:          uuid.New(),
		Title:       "Go Learning",
		Description: "Polish your Go skills",
	}

	mockDB.On("StoreDeck", ctx, deck).Return(nil)

	err := repo.StoreDeck(ctx, deck)

	assert.NoError(t, err)

	var cached string
	key := "cache:deck:" + deck.ID.String()
	mb := radix.Maybe{Rcv: &cached}
	err = h.redisClient.Do(ctx, radix.Cmd(&mb, "GET", key))
	assert.NoError(t, err)
	assert.False(t, mb.Null)

	mockDB.AssertExpectations(t)
}

func TestRedisRepository_DeleteDeck_InvalidatesCache(t *testing.T) {
	h := newRedisTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	deckID := uuid.New()

	deck := Deck{ID: deckID, Title: "Test Deck"}
	deckJSON, _ := json.Marshal(deck)
	key := "cache:deck:" + deckID.String()
	err := h.redisClient.Do(ctx, radix.FlatCmd(nil, "SETEX", key, 3600, string(deckJSON)))
	assert.NoError(t, err)

	mockDB.On("DeleteDeck", ctx, deckID).Return(nil)

	err = repo.DeleteDeck(ctx, deckID)
	assert.NoError(t, err)

	var cached string
	mb := radix.Maybe{Rcv: &cached}
	err = h.redisClient.Do(ctx, radix.Cmd(&mb, "GET", key))
	assert.NoError(t, err)

	assert.True(t, mb.Null)

	mockDB.AssertExpectations(t)
}

func TestRedisRepository_StoreCard_RefreshesCache(t *testing.T) {
	h := newRedisTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	deckID := uuid.New()
	card := Card{
		ID:    uuid.New(),
		Title: "Test Card",
		Kind:  "single_choice",
	}
	answers := []Answer{{ID: uuid.New(), Text: "Answer 1", IsCorrect: true}}

	deck := Deck{ID: deckID, Title: "Test Deck"}

	mockDB.On("StoreCard", ctx, card, deckID).Return(nil)
	mockDB.On("GetDeck", ctx, deckID).Return(deck, nil)
	mockDB.On("GetDeckCards", ctx, deckID).Return([]Card{card}, nil)
	mockDB.On("GetCardAnswers", ctx, card.ID).Return(answers, nil)

	err := repo.StoreCard(ctx, card, deckID)
	assert.NoError(t, err)

	mockDB.AssertExpectations(t)
}

func TestRedisRepository_UpdateDeck_InvalidatesCache(t *testing.T) {
	h := newRedisTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	deckID := uuid.New()
	newTitle := "Updated Title"
	updates := DeckUpdates{
		Title: &newTitle,
	}
	updatedDeck := Deck{
		ID:    deckID,
		Title: newTitle,
	}

	originalDeck := Deck{ID: deckID, Title: "Original Title"}
	deckJSON, _ := json.Marshal(originalDeck)
	key := "cache:deck:" + deckID.String()
	err := h.redisClient.Do(ctx, radix.FlatCmd(nil, "SETEX", key, 3600, string(deckJSON)))
	assert.NoError(t, err)

	mockDB.On("UpdateDeck", ctx, deckID, updates).Return(updatedDeck, nil)

	result, err := repo.UpdateDeck(ctx, deckID, updates)

	assert.NoError(t, err)
	assert.Equal(t, newTitle, result.Title)

	var cached string
	mb := radix.Maybe{Rcv: &cached}
	err = h.redisClient.Do(ctx, radix.Cmd(&mb, "GET", key))
	assert.NoError(t, err)
	assert.True(t, mb.Null, "Cache should be invalidated after UpdateDeck")

	mockDB.AssertExpectations(t)
}

func TestRedisRepository_UpdateDeck_DBError(t *testing.T) {
	h := newRedisTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	deckID := uuid.New()
	newTitle := "Updated Title"
	updates := DeckUpdates{
		Title: &newTitle,
	}

	mockDB.On("UpdateDeck", ctx, deckID, updates).Return(Deck{}, assert.AnError)

	result, err := repo.UpdateDeck(ctx, deckID, updates)
	assert.ErrorIs(t, err, assert.AnError)

	assert.Empty(t, result.ID)
	mockDB.AssertExpectations(t)
}

func TestRedisRepository_UpdateCard_InvalidatesCache(t *testing.T) {
	h := newRedisTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	deckID := uuid.New()
	cardID := uuid.New()
	newTitle := "Updated Card Title"
	updates := CardUpdates{
		Title: &newTitle,
	}
	updatedCard := Card{
		ID:    cardID,
		Title: newTitle,
		Kind:  "single_choice",
	}

	originalDeck := Deck{ID: deckID, Title: "Test Deck"}
	deckJSON, _ := json.Marshal(originalDeck)
	key := "cache:deck:" + deckID.String()
	err := h.redisClient.Do(ctx, radix.FlatCmd(nil, "SETEX", key, 3600, string(deckJSON)))
	assert.NoError(t, err)

	mockDB.On("UpdateCard", ctx, deckID, cardID, updates).Return(updatedCard, nil)

	result, err := repo.UpdateCard(ctx, deckID, cardID, updates)

	assert.NoError(t, err)
	assert.Equal(t, newTitle, result.Title)

	var cached string
	mb := radix.Maybe{Rcv: &cached}
	err = h.redisClient.Do(ctx, radix.Cmd(&mb, "GET", key))
	assert.NoError(t, err)
	assert.True(t, mb.Null)

	mockDB.AssertExpectations(t)
}

func TestRedisRepository_UpdateCard_DBError(t *testing.T) {
	h := newRedisTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	deckID := uuid.New()
	cardID := uuid.New()
	newTitle := "Updated Card Title"
	updates := CardUpdates{
		Title: &newTitle,
	}

	mockDB.On("UpdateCard", ctx, deckID, cardID, updates).Return(Card{}, assert.AnError)

	result, err := repo.UpdateCard(ctx, deckID, cardID, updates)
	assert.ErrorIs(t, err, assert.AnError)

	assert.Empty(t, result.ID)
	mockDB.AssertExpectations(t)
}

func TestRedisRepository_UpdateAnswer_InvalidatesCache(t *testing.T) {
	h := newRedisTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	deckID := uuid.New()
	cardID := uuid.New()
	answerID := uuid.New()
	newText := "Updated Answer Text"
	updates := AnswerUpdates{
		Text: &newText,
	}
	updatedAnswer := Answer{
		ID:        answerID,
		Text:      newText,
		IsCorrect: true,
	}

	originalDeck := Deck{ID: deckID, Title: "Test Deck"}
	deckJSON, _ := json.Marshal(originalDeck)
	key := "cache:deck:" + deckID.String()
	err := h.redisClient.Do(ctx, radix.FlatCmd(nil, "SETEX", key, 3600, string(deckJSON)))
	assert.NoError(t, err)

	mockDB.On("UpdateAnswer", ctx, deckID, cardID, answerID, updates).Return(updatedAnswer, nil)

	result, err := repo.UpdateAnswer(ctx, deckID, cardID, answerID, updates)
	assert.NoError(t, err)

	assert.Equal(t, newText, result.Text)

	var cached string
	mb := radix.Maybe{Rcv: &cached}
	err = h.redisClient.Do(ctx, radix.Cmd(&mb, "GET", key))
	assert.NoError(t, err)

	assert.True(t, mb.Null)

	mockDB.AssertExpectations(t)
}

func TestRedisRepository_UpdateAnswer_DBError(t *testing.T) {
	h := newRedisTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	deckID := uuid.New()
	cardID := uuid.New()
	answerID := uuid.New()
	newText := "Updated Answer Text"
	updates := AnswerUpdates{
		Text: &newText,
	}

	mockDB.On("UpdateAnswer", ctx, deckID, cardID, answerID, updates).Return(Answer{}, assert.AnError)

	result, err := repo.UpdateAnswer(ctx, deckID, cardID, answerID, updates)
	assert.ErrorIs(t, err, assert.AnError)

	assert.Empty(t, result.ID)

	mockDB.AssertExpectations(t)
}

func TestRedisRepository_GetDecks_PassThrough(t *testing.T) {
	h := newRedisTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	deckID := uuid.New()
	ids := []uuid.UUID{deckID}
	expectedDecks := map[uuid.UUID]Deck{
		deckID: {ID: deckID, Title: "Test Deck"},
	}

	mockDB.On("GetDecks", ctx, ids).Return(expectedDecks, nil)

	result, err := repo.GetDecks(ctx, ids)
	assert.NoError(t, err)

	assert.Equal(t, expectedDecks, result)

	mockDB.AssertExpectations(t)
}

func TestRedisRepository_GetDeckCards_PassThrough(t *testing.T) {
	h := newRedisTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	deckID := uuid.New()
	expectedCards := []Card{
		{ID: uuid.New(), Title: "Card 1"},
	}

	mockDB.On("GetDeckCards", ctx, deckID).Return(expectedCards, nil)

	result, err := repo.GetDeckCards(ctx, deckID)
	assert.NoError(t, err)

	assert.Equal(t, expectedCards, result)

	mockDB.AssertExpectations(t)
}

func TestRedisRepository_GetCardAnswers_PassThrough(t *testing.T) {
	h := newRedisTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	cardID := uuid.New()
	expectedAnswers := []Answer{
		{ID: uuid.New(), Text: "Answer 1", IsCorrect: true},
	}

	mockDB.On("GetCardAnswers", ctx, cardID).Return(expectedAnswers, nil)

	result, err := repo.GetCardAnswers(ctx, cardID)
	assert.NoError(t, err)

	assert.Equal(t, expectedAnswers, result)

	mockDB.AssertExpectations(t)
}

func TestRedisRepository_GetPopularDecks_PassThrough(t *testing.T) {
	h := newRedisTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	userID := uuid.New()
	pag := pagination.NewOldestFirstPagination(pagination.WithFirst(10))
	expectedConn := PopularDecksConnection{
		Edges: []PopularDeckEdge{
			{DeckID: uuid.New(), Cursor: "cursor1"},
		},
	}

	mockDB.On("GetPopularDecks", ctx, userID, pag).Return(expectedConn, nil)

	result, err := repo.GetPopularDecks(ctx, userID, pag)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(result.Edges))

	mockDB.AssertExpectations(t)
}

func TestRedisRepository_GetCards_PassThrough(t *testing.T) {
	h := newRedisTestHarness(t)
	mockDB := new(RepositoryMock)

	repo := NewRedisRepository(h.redisClient, mockDB)

	ctx := context.Background()
	cardID := uuid.New()
	ids := []uuid.UUID{cardID}
	expectedCards := map[uuid.UUID]Card{
		cardID: {ID: cardID, Title: "Test Card"},
	}

	mockDB.On("GetCards", ctx, ids).Return(expectedCards, nil)

	result, err := repo.GetCards(ctx, ids)
	assert.NoError(t, err)

	assert.Equal(t, expectedCards, result)

	mockDB.AssertExpectations(t)
}

type redisTestHarness struct {
	redisClient radix.Client
}

func newRedisTestHarness(t *testing.T) redisTestHarness {
	ctx := context.Background()

	redisClient, err := (radix.PoolConfig{}).New(ctx, "tcp", "localhost:6379")
	if err != nil {
		t.Fatalf("Failed to connect to Redis: %v", err)
	}

	if err := redisClient.Do(ctx, radix.Cmd(nil, "FLUSHDB")); err != nil {
		t.Fatalf("Failed to flush Redis: %v", err)
	}

	t.Cleanup(func() {
		redisClient.Close()
	})

	return redisTestHarness{
		redisClient: redisClient,
	}
}
