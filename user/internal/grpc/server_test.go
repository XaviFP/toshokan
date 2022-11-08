package grpc

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	pb "github.com/XaviFP/toshokan/user/api/proto/v1"
	"github.com/XaviFP/toshokan/user/internal/user"
)

func TestServer_GetUserID(t *testing.T) {
	repo := &user.RepositoryMock{}
	tokenRepo := &user.TokenRepositoryMock{}
	srv := &Server{
		Repository:      repo,
		TokenRepository: tokenRepo,
	}

	t.Run("by_token", func(t *testing.T) {
		token := "token"
		userID := uuid.MustParse("45a733e7-bcb0-4100-b495-c03c119927be")
		tokenRepo.On("GetUserID", mock.Anything, token).Return(userID, nil)

		res, err := srv.GetUserID(context.Background(), &pb.GetUserIDRequest{
			By: &pb.GetUserIDRequest_Token{Token: token},
		})
		assert.NoError(t, err)
		assert.Equal(t, &pb.GetUserIDResponse{Id: userID.String()}, res)
	})

	t.Run("by_username", func(t *testing.T) {
		userID := uuid.MustParse("45a733e7-bcb0-4100-b495-c03c119927be")
		repo.On("GetUserByUsername", mock.Anything, "Uncle Ben").Return(user.User{ID: userID}, nil)

		res, err := srv.GetUserID(context.Background(), &pb.GetUserIDRequest{
			By: &pb.GetUserIDRequest_Username{
				Username: "Uncle Ben",
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, &pb.GetUserIDResponse{Id: userID.String()}, res)
	})

	t.Run("invalid_argument", func(t *testing.T) {
		_, err := srv.GetUserID(context.Background(), &pb.GetUserIDRequest{})
		assert.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid by argument"))
	})
}

func TestServer_LogIn(t *testing.T) {
	authorizer := &user.AuthorizerMock{}
	srv := &Server{Authorizer: authorizer}

	t.Run("success", func(t *testing.T) {
		token := "GeneratedToken"
		authorizer.On("Authorize", mock.Anything, user.AuthorizationRequest{
			Username: "Uncle Ben",
			Password: "XXX",
		}).Return(token, nil)

		res, err := srv.LogIn(
			context.Background(),
			&pb.LogInRequest{
				Username: "Uncle Ben",
				Password: "XXX",
			},
		)
		assert.NoError(t, err)
		assert.Equal(t, &pb.LogInResponse{Token: token}, res)
	})

	t.Run("not_found", func(t *testing.T) {
		authorizer.On("Authorize", mock.Anything, user.AuthorizationRequest{
			Username: "Aunt May",
		}).Return("", user.ErrUserNotFound)

		_, err := srv.LogIn(context.Background(), &pb.LogInRequest{Username: "Aunt May"})
		assert.ErrorIs(t, err, status.Error(codes.NotFound, "user not found"))
	})

}

func TestSignUp(t *testing.T) {
	creator := &user.CreatorMock{}
	tokenRepo := &user.TokenRepositoryMock{}
	srv := &Server{
		Creator:         creator,
		TokenRepository: tokenRepo,
	}

	t.Run("success", func(t *testing.T) {
		u := user.User{
			ID:       uuid.MustParse("45a733e7-bcb0-4100-b495-c03c119927be"),
			Username: "Uncle Ben",
			Bio:      "Great power comes with great responsability.",
		}
		creator.On("Create", mock.Anything, user.CreateUserRequest{
			Username: u.Username,
			Password: "XXX",
			Bio:      u.Bio,
		}).Return(u, nil)

		token := "GeneratedToken"
		tokenRepo.On("Generate", mock.Anything, u.ID).Return(token, nil)

		userResponse, err := srv.SignUp(
			context.Background(),
			&pb.SignUpRequest{
				Username: u.Username,
				Password: "XXX",
				Bio:      u.Bio,
			})
		assert.NoError(t, err)
		assert.Equal(t, &pb.SignUpResponse{Token: token}, userResponse)
	})

	t.Run("failure", func(t *testing.T) {
		creator.On("Create", mock.Anything, user.CreateUserRequest{}).Return(user.User{}, assert.AnError)
		_, err := srv.SignUp(context.Background(), &pb.SignUpRequest{})
		assert.Error(t, err)
	})

}
