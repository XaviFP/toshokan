package user

import (
	"context"

	"time"

	paseto "aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
	"github.com/juju/errors"
	"github.com/tilinna/clock"

	"github.com/XaviFP/toshokan/common/config"
)

const userIDClaim = "user-id"

type TokenRepository interface {
	Generate(context.Context, uuid.UUID) (string, error)
	GetUserID(context.Context, string) (uuid.UUID, error)
}

type tokenRepository struct {
	publicKey   paseto.V4AsymmetricPublicKey
	privateKey  paseto.V4AsymmetricSecretKey
	tokenExpiry uint
	clock       clock.Clock
}

func NewTokenRepository(tc config.TokenConfig) (TokenRepository, error) {
	publicKey, err := paseto.NewV4AsymmetricPublicKeyFromBytes(tc.PublicKey)
	if err != nil {
		return nil, err
	}
	privateKey, err := paseto.NewV4AsymmetricSecretKeyFromBytes(tc.PrivateKey)
	if err != nil {
		return nil, err
	}

	return &tokenRepository{
		publicKey:   publicKey,
		privateKey:  privateKey,
		tokenExpiry: tc.SessionExpiry,
		clock:       clock.Realtime(),
	}, nil
}

func (r *tokenRepository) Generate(ctx context.Context, userID uuid.UUID) (string, error) {
	token := paseto.NewToken()
	token.SetIssuedAt(r.clock.Now())
	token.SetNotBefore(r.clock.Now())
	token.SetExpiration(r.clock.Now().Add(time.Duration(r.tokenExpiry) * time.Second))
	token.SetString(userIDClaim, userID.String())

	return token.V4Sign(r.privateKey, nil), nil
}

func (r *tokenRepository) GetUserID(ctx context.Context, token string) (uuid.UUID, error) {
	parser := paseto.NewParserForValidNow()
	parsed, err := parser.ParseV4Public(r.publicKey, token, nil)
	if err != nil {
		return uuid.UUID{}, errors.Trace(err)
	}

	userID, err := parsed.GetString(userIDClaim)
	if err != nil {
		return uuid.UUID{}, errors.Trace(err)
	}

	out, err := uuid.Parse(userID)
	if err != nil {
		return uuid.UUID{}, errors.Trace(err)
	}

	return out, nil
}
