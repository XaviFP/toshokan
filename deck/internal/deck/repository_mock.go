package deck

import (
	"context"

	"github.com/XaviFP/toshokan/common/pagination"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type RepositoryMock struct {
	mock.Mock
}

func (m *RepositoryMock) StoreDeck(ctx context.Context, d Deck) error {
	return m.Called(ctx, d).Error(0)
}

func (m *RepositoryMock) StoreCard(ctx context.Context, c Card, dID uuid.UUID) error {
	return m.Called(ctx, c, dID).Error(0)
}

func (m *RepositoryMock) GetDeck(ctx context.Context, id uuid.UUID) (Deck, error) {
	args := m.Called(ctx, id)

	return args[0].(Deck), args.Error(1)
}

func (m *RepositoryMock) GetDecks(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Deck, error) {
	args := m.Called(ctx, ids)

	return args[0].(map[uuid.UUID]Deck), args.Error(1)
}

func (m *RepositoryMock) DeleteDeck(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func (m *RepositoryMock) GetDeckCards(ctx context.Context, id uuid.UUID) ([]Card, error) {
	args := m.Called(ctx, id)

	return args[0].([]Card), args.Error(1)
}

func (m *RepositoryMock) GetCardAnswers(ctx context.Context, id uuid.UUID) ([]Answer, error) {
	args := m.Called(ctx, id)

	return args[0].([]Answer), args.Error(1)
}

func (m *RepositoryMock) GetPopularDecks(ctx context.Context, userID uuid.UUID, p pagination.Pagination) (PopularDecksConnection, error) {
	args := m.Called(ctx, userID, p)

	return args.Get(0).(PopularDecksConnection), args.Error(1)
}

func (m *RepositoryMock) GetCards(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Card, error) {
	args := m.Called(ctx, ids)

	return args[0].(map[uuid.UUID]Card), args.Error(1)
}
