package user

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/juju/errors"
	"github.com/mediocregopher/radix/v4"

	"github.com/XaviFP/toshokan/common/db"
)

var (
	ErrUserNotFound      = errors.New("user: user not found")
	ErrUserAlreadyExists = errors.New("user: user already exists")
)

type User struct {
	ID       uuid.UUID `json:"id,omitempty"`
	Username string    `json:"userName"`
	Bio      string    `json:"bio"`
	Nick     string    `json:"name"`
}

type Repository interface {
	Create(ctx context.Context, req CreateUserRequest) (User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (User, error)
	GetUserByUsername(ctx context.Context, userName string) (User, error)
	GetUserPassword(ctx context.Context, userName string) ([]byte, error)
}

type redisRepository struct {
	cache  radix.Client
	pgRepo Repository
}

func NewRedisRepository(cache radix.Client, pgRepo Repository) Repository {
	return &redisRepository{cache: cache, pgRepo: pgRepo}
}

func (r *redisRepository) Create(ctx context.Context, req CreateUserRequest) (User, error) {
	u, err := r.pgRepo.Create(ctx, req)
	if err != nil {
		return User{}, errors.Trace(err)
	}

	if err := r.doCache(ctx, u); err != nil {
		return User{}, errors.Trace(err)
	}

	return u, nil
}

func (r *redisRepository) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	var serialized string
	if err := r.cache.Do(
		ctx,
		radix.Cmd(&serialized, "GET", r.getUserCacheKey(id)),
	); err != nil {
		return User{}, errors.Trace(err)
	}

	if serialized != "" {
		var u User
		if err := json.Unmarshal([]byte(serialized), &u); err != nil {
			return User{}, errors.Trace(err)
		}

		return u, nil
	}

	u, err := r.pgRepo.GetUserByID(ctx, id)
	if err != nil {
		return User{}, errors.Trace(err)
	}

	if err := r.doCache(ctx, u); err != nil {
		return User{}, errors.Trace(err)
	}

	return u, nil
}

func (r *redisRepository) GetUserByUsername(ctx context.Context, userName string) (User, error) {
	return r.pgRepo.GetUserByUsername(ctx, userName)
}

func (r *redisRepository) GetUserPassword(ctx context.Context, userName string) ([]byte, error) {
	return r.pgRepo.GetUserPassword(ctx, userName)
}

func (r *redisRepository) doCache(ctx context.Context, u User) error {
	serialized, err := json.Marshal(u)
	if err != nil {
		return errors.Trace(err)
	}

	err = r.cache.Do(
		ctx,
		radix.Cmd(nil, "SET", r.getUserCacheKey(u.ID), string(serialized)),
	)

	return errors.Trace(err)
}

func (r *redisRepository) getUserCacheKey(id uuid.UUID) string {
	return fmt.Sprintf("cache:user:%s", id.String())
}

type pgRepository struct {
	db *sql.DB
}

func NewPGRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) Create(ctx context.Context, req CreateUserRequest) (User, error) {
	password, err := req.GetHashedPassword()
	if err != nil {
		return User{}, errors.Trace(err)
	}

	_, err = r.db.Exec(`
		INSERT INTO users (
			id,
			username,
			nick,
			password,
			bio,
			created_at
		) VALUES ($1, $2, $3, $4, $5, $6);`,
		req.ID,
		req.Username,
		req.Nick,
		password,
		req.Bio,
		time.Now().UTC(),
	)

	if err != nil {
		if db.IsConstraintError(err, "users_pkey") ||
			db.IsConstraintError(err, "users_username_key") {
			return User{}, ErrUserAlreadyExists
		}

		return User{}, errors.Trace(err)
	}

	return req.User(), nil
}

func (r *pgRepository) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	var u User
	row := r.db.QueryRow(`
		SELECT 
			id,
			username,
			nick,
			bio
		FROM users
		WHERE users.id = $1`,
		id.String(),
	)
	if err := row.Scan(&u.ID, &u.Username, &u.Nick, &u.Bio); err != nil {
		if err == sql.ErrNoRows {
			return u, ErrUserNotFound
		}

		return u, err
	}
	return u, nil
}

func (r *pgRepository) GetUserByUsername(ctx context.Context, userName string) (User, error) {
	var u User
	row := r.db.QueryRow(`
		SELECT
			id,
			username,
			nick,
			bio
		FROM users
		WHERE username = $1`,
		userName,
	)
	if err := row.Scan(&u.ID, &u.Username, &u.Nick, &u.Bio); err != nil {
		if err == sql.ErrNoRows {
			return u, ErrUserNotFound
		}

		return u, err
	}

	return u, nil
}

func (r *pgRepository) GetUserPassword(ctx context.Context, userName string) ([]byte, error) {
	var password []byte
	row := r.db.QueryRow(`
		SELECT
			password
		FROM users
		WHERE username = $1`,
		userName,
	)
	if err := row.Scan(&password); err != nil {
		if err == sql.ErrNoRows {
			return []byte{}, ErrUserNotFound
		}

		return []byte{}, err
	}

	return password, nil
}
