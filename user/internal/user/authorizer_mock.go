package user

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type AuthorizerMock struct {
	mock.Mock
}

func (m *AuthorizerMock) Authorize(ctx context.Context, req AuthorizationRequest) (string, error) {
	args := m.Called(ctx, req)
	return args.String(0), args.Error(1)
}
