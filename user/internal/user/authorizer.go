package user

import (
	"context"

	"github.com/juju/errors"
	"golang.org/x/crypto/bcrypt"
)

type AuthorizationRequest struct {
	Username string
	Password string
}
type Authorizer interface {
	Authorize(context.Context, AuthorizationRequest) (string, error)
}

func NewAuthorizer(repo Repository, tokenRepo TokenRepository) Authorizer {
	return &authorizer{
		repo:      repo,
		tokenRepo: tokenRepo,
	}
}

type authorizer struct {
	repo      Repository
	tokenRepo TokenRepository
}

func (a *authorizer) Authorize(ctx context.Context, req AuthorizationRequest) (string, error) {
	u, err := a.repo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		return "", errors.Trace(err)
	}

	storedPassword, err := a.repo.GetUserPassword(ctx, req.Username)
	if err != nil {
		return "", errors.Trace(err)
	}

	if err := bcrypt.CompareHashAndPassword(storedPassword, []byte(req.Password)); err != nil {
		return "", errors.Trace(err)
	}

	token, err := a.tokenRepo.Generate(ctx, u.ID)
	if err != nil {
		return "", errors.Trace(err)
	}

	return token, nil
}
