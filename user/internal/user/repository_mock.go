package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type RepositoryMock struct {
	mock.Mock
}

func (m *RepositoryMock) Create(ctx context.Context, req CreateUserRequest) (User, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(User), args.Error(1)
}

func (m *RepositoryMock) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	args := m.Called(ctx, id)

	return args[0].(User), args.Error(1)
}

func (m *RepositoryMock) GetUserByUsername(ctx context.Context, username string) (User, error) {
	args := m.Called(ctx, username)

	return args[0].(User), args.Error(1)
}

func (m *RepositoryMock) GetUserPassword(ctx context.Context, username string) ([]byte, error) {
	args := m.Called(ctx, username)

	return args.Get(0).([]byte), args.Error(1)
}
