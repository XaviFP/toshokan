package cache

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type CacheMock struct {
	mock.Mock
}

func (m *CacheMock) SetEx(ctx context.Context, key, value string, seconds uint) error {
	return m.Called(ctx, key, value, seconds).Error(0)
}

func (m *CacheMock) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)

	return args.String(0), args.Error(1)
}

func (m *CacheMock) Delete(ctx context.Context, key string) error {
	return m.Called(ctx, key).Error(0)
}
