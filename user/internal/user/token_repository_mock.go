package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type TokenRepositoryMock struct {
	mock.Mock
}

func (m *TokenRepositoryMock) Generate(ctx context.Context, id uuid.UUID) (string, error) {
	args := m.Called(ctx, id)

	return args[0].(string), args.Error(1)
}

func (m *TokenRepositoryMock) GetUserID(ctx context.Context, token string) (uuid.UUID, error) {
	args := m.Called(ctx, token)

	return args[0].(uuid.UUID), args.Error(1)
}
