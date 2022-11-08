package user

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthorizer_Authorize(t *testing.T) {
	repo := &RepositoryMock{}
	tokenRepo := &TokenRepositoryMock{}
	authorizer := NewAuthorizer(repo, tokenRepo)

	t.Run("success", func(t *testing.T) {
		u := User{
			ID: uuid.MustParse("1f30a72f-5d7a-48da-a5c2-42efece6972a"),
		}
		repo.On("GetUserByUsername", mock.Anything, "Uncle Ben").Return(u, nil)

		password, err := bcrypt.GenerateFromPassword([]byte("XXX"), bcrypt.DefaultCost)
		assert.NoError(t, err)
		repo.On("GetUserPassword", mock.Anything, "Uncle Ben").Return(password, nil)

		expected := "GeneratedToken"
		tokenRepo.On("Generate", mock.Anything, u.ID).Return(expected, nil)

		actual, err := authorizer.Authorize(context.Background(), AuthorizationRequest{
			Username: "Uncle Ben",
			Password: "XXX",
		})
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("wrong_password", func(t *testing.T) {
		userID := uuid.MustParse("1f30a72f-5d7a-48da-a5c2-42efece6972a")
		passwordStr := "XXX"
		wrongPassword := "YYY"
		username := "Mary Jane"
		password, err := bcrypt.GenerateFromPassword([]byte(passwordStr), bcrypt.DefaultCost)
		assert.NoError(t, err)
		repo.On("GetUserByUsername", mock.Anything, username).Return(User{ID: userID}, nil)
		repo.On("GetUserPassword", mock.Anything, username).Return(password, nil)

		_, err = authorizer.Authorize(context.Background(), AuthorizationRequest{
			Username: username,
			Password: wrongPassword,
		})
		assert.Error(t, err)
	})

	t.Run("token_generation_error", func(t *testing.T) {
		userID := uuid.MustParse("84314b3c-7bf0-4efb-845b-ff35245f12d9")
		username := "Gwen Stacy"
		passwordStr := "XXX"
		password, err := bcrypt.GenerateFromPassword([]byte(passwordStr), bcrypt.DefaultCost)
		assert.NoError(t, err)

		repo.On("GetUserByUsername", mock.Anything, username).Return(User{ID: userID}, nil)
		repo.On("GetUserPassword", mock.Anything, username).Return(password, nil)
		tokenRepo.On("Generate", mock.Anything, userID).Return("", assert.AnError)

		_, err = authorizer.Authorize(context.Background(), AuthorizationRequest{
			Username: username,
			Password: passwordStr,
		})
		assert.Error(t, err)
	})

}
