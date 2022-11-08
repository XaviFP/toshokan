package user

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreator_Create(t *testing.T) {
	expected := User{
		ID:       uuid.MustParse("be71305e-4fec-4e7a-845a-d76b0b5ed0f1"),
		Username: "Uncle Ben",
		Bio:      "Great power comes with great responsability.",
		Nick:     "Benny",
	}

	repo := &RepositoryMock{}
	repo.On("Create", mock.Anything, mock.Anything).Return(expected, nil)

	creator := NewCreator(repo)

	req := CreateUserRequest{
		Username: "Uncle Ben",
		Password: "XXX",
		Bio:      "Great power comes with great responsability.",
		Nick:     "Benny",
	}

	actual, err := creator.Create(context.Background(), req)
	assert.NoError(t, err)

	assert.Equal(t, expected, actual)
}

func TestCreateUserRequest_Validate(t *testing.T) {
	t.Run("ErrNoUserName", func(t *testing.T) {
		req := CreateUserRequest{}

		assert.ErrorIs(t, req.Validate(), ErrNoUserName)
	})

	t.Run("ErrNoPassword", func(t *testing.T) {
		req := CreateUserRequest{
			Username: "Uncle Ben",
		}

		assert.ErrorIs(t, req.Validate(), ErrNoPassword)
	})
}
