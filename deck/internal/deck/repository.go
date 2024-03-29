package deck

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/XaviFP/toshokan/common/db"
	"github.com/XaviFP/toshokan/common/pagination"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/juju/errors"
	"github.com/lib/pq"
	"github.com/mediocregopher/radix/v4"
)

var (
	ErrCards             = errors.New("deck: one or more cards are not valid")
	ErrCardInvalid       = errors.New("deck: invalid card")
	ErrCardAlreadyExists = errors.New("deck: card already exists")
	ErrDeckNotFound      = errors.New("deck: deck not found")
	ErrNoTitle           = errors.New("deck: title is missing")
	ErrNoDescription     = errors.New("deck: description is missing")
	ErrNoAnswersProvided = errors.New("deck: no answers provided")
	ErrNoCorrectAnswer   = errors.New("deck: at least one answer must be correct")
	ErrNoTextAnswer      = errors.New("deck: all answers must have a non-empty text")
	ErrDeckAlreadyExists = errors.New("deck: deck already exists")
	ErrDeckInvalid       = errors.New("deck: invalid deck")
	ErrInvalidCursor     = errors.New("deck: invalid cusror")
)

type Repository interface {
	DeleteDeck(ctx context.Context, id uuid.UUID) error
	GetDecks(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Deck, error)
	GetDeck(ctx context.Context, id uuid.UUID) (Deck, error)
	GetDeckCards(ctx context.Context, id uuid.UUID) ([]Card, error)
	GetCardAnswers(ctx context.Context, id uuid.UUID) ([]Answer, error)
	StoreDeck(ctx context.Context, d Deck) error
	StoreCard(ctx context.Context, card Card, deckID uuid.UUID) error
	GetPopularDecks(ctx context.Context, userID uuid.UUID, p pagination.Pagination) (PopularDecksConnection, error)

	GetCards(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Card, error)
}

type redisRepository struct {
	cache  radix.Client
	pgRepo Repository
}

func (r *redisRepository) StoreCard(ctx context.Context, c Card, dID uuid.UUID) error {
	if err := r.pgRepo.StoreCard(ctx, c, dID); err != nil {
		return errors.Trace(err)
	}

	r.delete(ctx, dID.String())

	// Update cache TODO
	if _, err := r.GetDeck(ctx, dID); err == nil {
		return errors.Trace(err)
	}

	return nil
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

func (r *redisRepository) GetDecks(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Deck, error) {
	return r.pgRepo.GetDecks(ctx, ids)
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

func (r *redisRepository) GetPopularDecks(ctx context.Context, userID uuid.UUID, p pagination.Pagination) (PopularDecksConnection, error) {
	return r.pgRepo.GetPopularDecks(ctx, userID, p)
}

func (r *redisRepository) GetCards(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Card, error) {
	return r.pgRepo.GetCards(ctx, ids)
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

func (r *pgRepository) GetDecks(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Deck, error) {
	out := make(map[uuid.UUID]Deck, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	rows, err := r.db.Query(`
		SELECT 
			id,
			author_id,
			title,
			"description",
			is_public
		FROM decks
		WHERE
			deleted_at IS NULL
			AND id = ANY($1)
		ORDER BY created_at`,
		pq.Array(ids),
	)
	if err != nil {
		return out, errors.Trace(err)
	}

	for rows.Next() {
		var d Deck

		if err := rows.Scan(&d.ID, &d.AuthorID, &d.Title, &d.Description, &d.Public); err != nil {
			return out, errors.Trace(err)
		}

		out[d.ID] = d
	}

	return out, nil
}

func (r *pgRepository) GetPopularDecks(ctx context.Context, userID uuid.UUID, p pagination.Pagination) (PopularDecksConnection, error) {
	var (
		out   PopularDecksConnection
		arger db.Argumenter
	)

	whereConditions := []string{
		"deleted_at IS NULL",
		fmt.Sprintf("(is_public = true OR author_id = %s)", arger.Add(userID)),
	}

	if !p.Cursor().IsEmpty() {
		var cursor Cursor
		if err := pagination.FromCursor(p.Cursor(), &cursor); err != nil {
			return out, ErrInvalidCursor
		}

		whereConditions = append(
			whereConditions,
			fmt.Sprintf(
				"(created_at, id) %s (%s, %s)",
				p.Comparator(),
				arger.Add(cursor.CreatedAt),
				arger.Add(cursor.ID),
			),
		)
	}

	query := fmt.Sprintf(`
		SELECT id, created_at
		FROM decks
		WHERE
			%s
		ORDER BY created_at %s, id %s
		LIMIT %s`,
		strings.Join(whereConditions, " AND "),
		p.OrderBy(),
		p.OrderBy(),
		arger.Add(p.Limit()+1),
	)

	rows, err := r.db.Query(query, arger.Values()...)
	if err != nil {
		return out, errors.Trace(err)
	}

	for rows.Next() {
		var c Cursor

		if err := rows.Scan(&c.ID, &c.CreatedAt); err != nil {
			return out, errors.Trace(err)
		}

		cursor, err := pagination.ToCursor(c)
		if err != nil {
			return out, errors.Trace(err)
		}

		out.Edges = append(out.Edges, PopularDeckEdge{
			DeckID: c.ID,
			Cursor: cursor,
		})
	}

	hasMore := len(out.Edges) > p.Limit()

	pageInfo := pagination.PageInfo{
		HasPreviousPage: hasMore && !p.IsForward(),
		HasNextPage:     hasMore && p.IsForward(),
	}

	if hasMore {
		out.Edges = out.Edges[:len(out.Edges)-1]
	}

	if !p.IsForward() {
		sort.SliceStable(out.Edges, func(i, j int) bool {
			return i > j
		})
	}

	if hasMore {
		pageInfo.StartCursor = out.Edges[0].Cursor
		pageInfo.EndCursor = out.Edges[len(out.Edges)-1].Cursor
	}

	out.PageInfo = pageInfo

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

	stmt, err := tx.Prepare(pq.CopyIn("cards", "id", "deck_id", "title", "explanation"))
	if err != nil {
		return errors.Trace(err)
	}

	for _, card := range d.Cards {
		if _, err := stmt.Exec(card.ID, d.ID, card.Title, card.Explanation); err != nil {
			return errors.Trace(err)
		}
	}
	if _, err = stmt.Exec(); err != nil {
		return errors.Trace(err)
	}

	stmt, err = tx.Prepare(pq.CopyIn("answers", "id", "card_id", "text", "is_correct"))
	if err != nil {
		return errors.Trace(err)
	}

	for _, card := range d.Cards {
		for _, answer := range card.PossibleAnswers {
			if _, err := stmt.Exec(answer.ID, card.ID, answer.Text, answer.IsCorrect); err != nil {
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

func (r *pgRepository) StoreCard(ctx context.Context, c Card, dID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Trace(err)
	}

	defer tx.Rollback()

	_, err = tx.Exec(`INSERT INTO cards (
			id,
			deck_id,
			title,
			explanation,
			created_at
		) VALUES ($1, $2, $3, $4, NOW());`,
		c.ID, dID, c.Title, c.Explanation,
	)
	if err != nil {
		if db.IsConstraintError(err, "cards_pkey") {
			return ErrCardAlreadyExists
		}
		if db.IsConstraintError(err, "cards_deck_id_fkey") {
			return ErrDeckNotFound
		}

		return errors.Trace(err)
	}

	stmt, err := tx.Prepare(pq.CopyIn("answers", "id", "card_id", "text", "is_correct"))
	if err != nil {
		return errors.Trace(err)
	}

	for _, answer := range c.PossibleAnswers {
		if _, err := stmt.Exec(answer.ID, c.ID, answer.Text, answer.IsCorrect); err != nil {
			return errors.Trace(err)
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
			title,
			explanation
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
		if err := rows.Scan(&c.ID, &c.Title, &c.Explanation); err != nil {
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

func (r *pgRepository) GetCards(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Card, error) {
	out := make(map[uuid.UUID]Card, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	rows, err := r.db.Query(`
		SELECT 
			id,
			title,
			explanation
		FROM cards
		WHERE
			deleted_at IS NULL
			AND id = ANY($1)
		ORDER BY created_at`,
		pq.Array(ids),
	)
	if err != nil {
		return out, errors.Trace(err)
	}

	for rows.Next() {
		var c Card
		var explanation sql.NullString

		if err := rows.Scan(&c.ID, &c.Title, &explanation); err != nil {
			return out, errors.Trace(err)
		}

		c.Explanation = explanation.String

		// Temporary extra db calls per card for convinience
		answers, err := r.GetCardAnswers(ctx, c.ID)
		if err != nil {
			return out, errors.Trace(err)
		}
		c.PossibleAnswers = answers

		out[c.ID] = c
	}

	return out, nil
}

func (r *redisRepository) getDeckCacheKey(id uuid.UUID) string {
	return fmt.Sprintf("cache:deck:%s", id.String())
}
