package deck

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/XaviFP/toshokan/common/db"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/juju/errors"
	"github.com/lib/pq"
	"github.com/mediocregopher/radix/v4"
)

var (
	ErrCards             = errors.New("deck: one or more cards are not valid")
	ErrDeckNotFound      = errors.New("deck: deck not found")
	ErrNoTitle           = errors.New("deck: title is missing")
	ErrNoDescription     = errors.New("deck: description is missing")
	ErrNoAnswersProvided = errors.New("deck: no answers provided")
	ErrNoCorrectAnswer   = errors.New("deck: at least one answer must be correct")
	ErrNoTextAnswer      = errors.New("deck: all answers must have a non-empty text")
	ErrDeckAlreadyExists = errors.New("deck: deck already exists")
	ErrDeckInvalid       = errors.New("deck: invalid deck")
)

type Repository interface {
	DeleteDeck(ctx context.Context, id uuid.UUID) error
	GetDecks(ctx context.Context) ([]Deck, error)
	GetDeck(ctx context.Context, id uuid.UUID) (Deck, error)
	GetDeckCards(ctx context.Context, id uuid.UUID) ([]Card, error)
	GetCardAnswers(ctx context.Context, id uuid.UUID) ([]Answer, error)
	StoreDeck(ctx context.Context, d Deck) error
}

type redisRepository struct {
	cache  radix.Client
	pgRepo Repository
}

func NewRedisRepository(cache radix.Client, pgRepo Repository) Repository {
	return &redisRepository{cache: cache, pgRepo: pgRepo}
}

func (r *redisRepository) GetDeck(ctx context.Context, id uuid.UUID) (Deck, error) {
	var serialized string
	if err := r.cache.Do(
		ctx,
		radix.Cmd(&serialized, "GET", r.getDeckCacheKey(id)),
	); err != nil {
		return Deck{}, errors.Trace(err)
	}

	// Found in cache
	if serialized != "" {
		var d Deck
		err := json.Unmarshal([]byte(serialized), &d)

		return d, errors.Trace(err)
	}

	// Not found in cache
	d, err := r.getDeckFromDB(ctx, id)
	if err != nil {
		return d, errors.Trace(err)
	}

	return r.doCache(ctx, d)
}

func (r *redisRepository) DeleteDeck(ctx context.Context, id uuid.UUID) error {
	if err := r.pgRepo.DeleteDeck(ctx, id); err != nil {
		return errors.Trace(err)
	}

	if err := r.delete(ctx, r.getDeckCacheKey(id)); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (r *redisRepository) GetDecks(ctx context.Context) ([]Deck, error) {
	return r.pgRepo.GetDecks(ctx)
}

func (r *redisRepository) GetDeckCards(ctx context.Context, id uuid.UUID) ([]Card, error) {
	return r.pgRepo.GetDeckCards(ctx, id)
}

func (r *redisRepository) GetCardAnswers(ctx context.Context, id uuid.UUID) ([]Answer, error) {
	return r.pgRepo.GetCardAnswers(ctx, id)
}

func (r *redisRepository) StoreDeck(ctx context.Context, d Deck) error {
	if err := r.pgRepo.StoreDeck(ctx, d); err != nil {
		return errors.Trace(err)
	}

	if _, err := r.doCache(ctx, d); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (r *redisRepository) getDeckFromDB(ctx context.Context, id uuid.UUID) (Deck, error) {
	d, err := r.pgRepo.GetDeck(ctx, id)
	if err != nil {
		return Deck{}, errors.Trace(err)
	}

	cards, err := r.pgRepo.GetDeckCards(ctx, id)
	if err != nil {
		return Deck{}, errors.Trace(err)
	}

	for _, c := range cards {
		answers, err := r.pgRepo.GetCardAnswers(ctx, c.ID)
		if err != nil {
			return Deck{}, errors.Trace(err)
		}

		c.PossibleAnswers = answers
		d.Cards = append(d.Cards, c)

	}

	return d, nil
}

func (r *redisRepository) doCache(ctx context.Context, d Deck) (Deck, error) {
	serialized, err := json.Marshal(d)
	if err != nil {
		return d, errors.Trace(err)
	}

	err = r.cache.Do(
		ctx,
		radix.Cmd(nil, "SET", r.getDeckCacheKey(d.ID), string(serialized)),
	)

	return d, errors.Trace(err)
}

func (r *redisRepository) delete(ctx context.Context, key string) error {
	err := r.cache.Do(ctx, radix.Cmd(nil, "DEL", key))

	return errors.Trace(err)
}

type pgRepository struct {
	db *sql.DB
}

func NewPGRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) DeleteDeck(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(`UPDATE decks SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (r *pgRepository) GetDecks(ctx context.Context) ([]Deck, error) {
	rows, err := r.db.Query(`
		SELECT 
			id,
			author_id,
			title,
			"description",
			is_public
		FROM decks
		WHERE deleted_at IS NULL
		ORDER BY created_at`,
	)
	if err != nil {
		return []Deck{}, err
	}

	var (
		d   Deck
		out []Deck
	)

	for rows.Next() {
		if err := rows.Scan(&d.ID, &d.AuthorID, &d.Title, &d.Description, &d.Public); err != nil {
			return []Deck{}, errors.Trace(err)
		}

		out = append(out, d)
	}

	return out, nil
}

func (r *pgRepository) GetDeck(ctx context.Context, id uuid.UUID) (Deck, error) {
	var d Deck
	row := r.db.QueryRow(`
		SELECT 
			id,
			author_id,
			title,
			description,
			is_public
		FROM decks
		WHERE 
			id = $1 
			AND deleted_at IS NULL`,
		id,
	)
	if err := row.Scan(&d.ID, &d.AuthorID, &d.Title, &d.Description, &d.Public); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Deck{}, ErrDeckNotFound
		}

		return Deck{}, errors.Trace(err)
	}

	return d, nil
}

func (r *pgRepository) StoreDeck(ctx context.Context, d Deck) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Trace(err)
	}

	defer tx.Rollback()

	_, err = tx.Exec(`
		INSERT INTO decks (
			id,
			author_id,
			title,
			"description",
			is_public,
			created_at
		) VALUES ($1, $2, $3, $4, $5, NOW());`,
		d.ID, d.AuthorID, d.Title, d.Description, d.Public,
	)
	if db.IsConstraintError(err, "decks_pkey") {
		return ErrDeckAlreadyExists
	}

	stmt, err := tx.Prepare(pq.CopyIn("cards", "id", "title"))
	if err != nil {
		return errors.Trace(err)
	}

	for _, card := range d.Cards {
		if _, err := stmt.Exec(card.ID, card.Title); err != nil {
			return errors.Trace(err)
		}
	}
	if _, err = stmt.Exec(); err != nil {
		return errors.Trace(err)
	}

	stmt, err = tx.Prepare(pq.CopyIn("answers", "id", "text", "is_correct"))
	if err != nil {
		return errors.Trace(err)
	}

	for _, card := range d.Cards {
		for _, answer := range card.PossibleAnswers {
			if _, err := stmt.Exec(answer.ID, answer.Text, answer.IsCorrect); err != nil {
				return errors.Trace(err)
			}
		}
	}

	if _, err = stmt.Exec(); err != nil {
		return errors.Trace(err)
	}

	if err = tx.Commit(); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (r *pgRepository) GetDeckCards(ctx context.Context, id uuid.UUID) ([]Card, error) {
	var out []Card
	rows, err := r.db.Query(`
		SELECT
			id,
			title
		FROM cards
		WHERE
			deck_id = $1
			AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return []Card{}, errors.Trace(err)
	}

	for rows.Next() {
		var c Card
		if err := rows.Scan(&c.ID, &c.Title); err != nil {
			return []Card{}, errors.Trace(err)
		}

		out = append(out, c)
	}

	return out, nil
}

func (r *pgRepository) GetCardAnswers(ctx context.Context, id uuid.UUID) ([]Answer, error) {
	rows, err := r.db.Query(`
		SELECT
			id,
			text,
			is_correct
		FROM answers
		WHERE
			card_id = $1
			AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return []Answer{}, errors.Trace(err)
	}

	var out []Answer

	for rows.Next() {
		var a Answer
		if err := rows.Scan(&a.ID, &a.Text, &a.IsCorrect); err != nil {
			return out, errors.Trace(err)
		}

		out = append(out, a)
	}

	return out, nil
}

func (r *redisRepository) getDeckCacheKey(id uuid.UUID) string {
	return fmt.Sprintf("cache:deck:%s", id.String())
}
