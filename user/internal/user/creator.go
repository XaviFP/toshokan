package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/juju/errors"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNoUserName = errors.New("user: username is missing")
	ErrNoPassword = errors.New("user: password is missing")
)

type CreateUserRequest struct {
	ID       uuid.UUID
	Username string
	Password string
	Nick     string
	Bio      string
}

func (req CreateUserRequest) User() User {
	return User{
		ID:       req.ID,
		Username: req.Username,
		Nick:     req.Nick,
		Bio:      req.Bio,
	}
}

func (req CreateUserRequest) GetHashedPassword() ([]byte, error) {
	password, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return []byte{}, errors.Trace(err)
	}

	return password, nil
}

func (req CreateUserRequest) Validate() error {
	if req.Username == "" {
		return ErrNoUserName
	}

	if req.Password == "" {
		return ErrNoPassword
	}

	return nil
}

type Creator interface {
	Create(context.Context, CreateUserRequest) (User, error)
}

type creator struct {
	repo Repository
}

func NewCreator(repo Repository) Creator {
	return &creator{
		repo: repo,
	}
}

func (c *creator) Create(ctx context.Context, req CreateUserRequest) (User, error) {
	if err := req.Validate(); err != nil {
		return User{}, errors.Trace(err)
	}

	req.ID = uuid.New()

	u, err := c.repo.Create(ctx, req)
	if err != nil {
		return User{}, errors.Trace(err)
	}

	return u, nil
}
