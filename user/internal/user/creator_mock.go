package user

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type CreatorMock struct {
	mock.Mock
}

func (m *CreatorMock) Create(ctx context.Context, req CreateUserRequest) (User, error) {
	args := m.Called(ctx, req)

	return args[0].(User), args.Error(1)
}
