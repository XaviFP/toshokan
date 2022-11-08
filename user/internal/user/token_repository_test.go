package user

import (
	"context"
	"testing"
	"time"

	paseto "aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/tilinna/clock"
)

func TestTokenRepository_Generate(t *testing.T) {
	userID := uuid.MustParse("9e67068d-1c43-4e03-afd5-2b7a7c1b86dc")
	tokenExpiry := uint(60)

	clock := clock.NewMock(time.Date(2019, time.December, 21, 12, 0, 0, 0, time.UTC))

	keys := paseto.NewV4AsymmetricSecretKey()
	repo := tokenRepository{
		clock:         clock,
		tokenExpiry: tokenExpiry,
		publicKey:     keys.Public(),
		privateKey:    keys,
	}

	tokenStr, err := repo.Generate(context.Background(), userID)
	assert.NoError(t, err)

	token := parseToken(t, keys.Public(), tokenStr)

	expiration, err := token.GetExpiration()
	assert.NoError(t, err)

	expectedExpiration := clock.Now().Add(time.Duration(tokenExpiry) * time.Second)
	assert.Equal(t, expectedExpiration, expiration)

	issuedAt, err := token.GetIssuedAt()
	assert.NoError(t, err)
	assert.Equal(t, clock.Now(), issuedAt)

	notBefore, err := token.GetNotBefore()
	assert.NoError(t, err)
	assert.Equal(t, clock.Now(), notBefore)

	userIDStr, err := token.GetString("user-id")
	assert.NoError(t, err)
	assert.Equal(t, userID.String(), userIDStr)
}

func TestTokenRepository_GetUserID(t *testing.T) {
	expected := uuid.MustParse("9e67068d-1c43-4e03-afd5-2b7a7c1b86dc")
	tokenExpiry := uint(60000)

	keys := paseto.NewV4AsymmetricSecretKey()
	repo := tokenRepository{
		clock:         clock.Realtime(),
		tokenExpiry: tokenExpiry,
		publicKey:     keys.Public(),
		privateKey:    keys,
	}

	tokenStr, err := repo.Generate(context.Background(), expected)
	assert.NoError(t, err)

	actual, err := repo.GetUserID(context.Background(), tokenStr)
	assert.NoError(t, err)

	assert.Equal(t, expected, actual)
}

func parseToken(t *testing.T, publicKey paseto.V4AsymmetricPublicKey, tokenStr string) *paseto.Token {
	t.Helper()

	parser := paseto.NewParserWithoutExpiryCheck()
	token, err := parser.ParseV4Public(publicKey, tokenStr, nil)
	assert.NoError(t, err)

	return token
}
