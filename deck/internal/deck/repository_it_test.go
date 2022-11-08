package deck

import (
	"context"
	"database/sql"
	"testing"

	"github.com/XaviFP/toshokan/common/config"
	"github.com/XaviFP/toshokan/common/db"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestRepository_StoreDeck(t *testing.T) {
	h := newTestHarness(t)

	repo := NewPGRepository(h.db)

	id := uuid.MustParse("ebcfffa0-a96f-450b-a0f3-a2e47263855d")
	d := Deck{ID: id, Title: "Go Learning", Description: "Polish your Go skills"}

	t.Run("success", func(t *testing.T) {
		err := repo.StoreDeck(context.Background(), d)
		assert.NoError(t, err)

		var out Deck
		row := h.db.QueryRow(`SELECT id, author_id, title, description FROM decks WHERE id = $1 AND deleted_at IS NULL`, id)
		err = row.Scan(&out.ID, &out.AuthorID, &out.Title, &out.Description)
		assert.NoError(t, err)
		assert.Equal(t, d, out)
	})

	t.Run("failure", func(t *testing.T) {
		err := repo.StoreDeck(context.Background(), d)
		assert.ErrorIs(t, err, ErrDeckAlreadyExists)
	})

}

func TestRepository_GetDeck(t *testing.T) {
	h := newTestHarness(t)

	repo := NewPGRepository(h.db)

	t.Run("success", func(t *testing.T) {
		id := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

		d, err := repo.GetDeck(context.Background(), id)
		assert.NoError(t, err)
		assert.Equal(t, Deck{
			ID:          uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72"),
			AuthorID:    uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9"),
			Title:       "Programming languages",
			Description: "Compiled or interpreted?",
		}, d)
	})

	t.Run("failure", func(t *testing.T) {
		id := uuid.MustParse("ebcfffa0-a96f-450b-a0f3-a2e47263855d")

		d, err := repo.GetDeck(context.Background(), id)
		assert.ErrorIs(t, err, ErrDeckNotFound)
		assert.Empty(t, d)
	})
}

func TestRepository_GetDecks(t *testing.T) {
	h := newTestHarness(t)

	repo := NewPGRepository(h.db)

	t.Run("success", func(t *testing.T) {
		expected := []Deck{
			{
				ID:          uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72"),
				AuthorID:    uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9"),
				Title:       "Programming languages",
				Description: "Compiled or interpreted?",
			},
		}

		actual, err := repo.GetDecks(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("empty", func(t *testing.T) {
		_, err := h.db.Exec(`UPDATE decks SET deleted_at = NOW()`)
		assert.NoError(t, err)

		decks, err := repo.GetDecks(context.Background())
		assert.NoError(t, err)
		assert.Empty(t, decks)
	})
}

func TestRepository_DeleteDeck(t *testing.T) {
	h := newTestHarness(t)

	repo := NewPGRepository(h.db)
	deckID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

	deckExists := func(id uuid.UUID) bool {
		var exists bool
		row := h.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM decks WHERE id = $1 AND deleted_at IS NULL)`, id)
		err := row.Scan(&exists)
		assert.NoError(t, err)

		return exists
	}

	assert.True(t, deckExists(deckID))

	err := repo.DeleteDeck(context.Background(), deckID)
	assert.NoError(t, err)
	assert.False(t, deckExists(deckID))
}

func TestRepository_GetDeckCards(t *testing.T) {
	h := newTestHarness(t)

	repo := NewPGRepository(h.db)
	id := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

	t.Run("success", func(t *testing.T) {
		expected := []Card{
			{
				ID:    uuid.MustParse("72bdff92-5bc8-4e1d-9217-d0b23e22ff33"),
				Title: "Golang",
			},
			{
				ID:    uuid.MustParse("c924f7e0-efd8-4c2d-9c43-8eafb7102ebc"),
				Title: "Rust",
			},
			{
				ID:    uuid.MustParse("d42a90dd-818c-4eed-8e9f-9e8af1a654f4"),
				Title: "Lua",
			},
		}

		actual, err := repo.GetDeckCards(context.Background(), id)
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("empty", func(t *testing.T) {
		_, err := h.db.Exec(`UPDATE cards SET deleted_at = NOW() WHERE deck_id = $1`, id)
		assert.NoError(t, err)

		cards, err := repo.GetDeckCards(context.Background(), id)
		assert.NoError(t, err)
		assert.Empty(t, cards)
	})
}

func TestRepository_GetCardAnswers(t *testing.T) {
	h := newTestHarness(t)

	repo := NewPGRepository(h.db)
	id := uuid.MustParse("72bdff92-5bc8-4e1d-9217-d0b23e22ff33")

	t.Run("success", func(t *testing.T) {
		expected := []Answer{
			{
				ID:        uuid.MustParse("7e6926da-82b2-4ae8-99b4-1b803ebf1877"),
				Text:      "Compiled",
				IsCorrect: true,
			},
			{
				ID:        uuid.MustParse("dfcb1c81-f590-486e-9b7e-a44f0c436933"),
				Text:      "Interpreted",
				IsCorrect: false,
			},
		}

		actual, err := repo.GetCardAnswers(context.Background(), id)
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("empty", func(t *testing.T) {
		_, err := h.db.Exec(`UPDATE answers SET deleted_at = NOW() WHERE card_id = $1`, id)
		assert.NoError(t, err)

		answers, err := repo.GetCardAnswers(context.Background(), id)
		assert.NoError(t, err)
		assert.Empty(t, answers)
	})
}

type testHarness struct {
	db *sql.DB
}

func newTestHarness(t *testing.T) testHarness {
	db, err := db.InitDB(config.DBConfig{User: "toshokan", Password: "t.o.s.h.o.k.a.n.", Name: "test", Host: "localhost", Port: "5432"})
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		t.Fatal(err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://../../cmd/migrate/migrations", "toshokan", driver)
	if err != nil {
		t.Fatal(err)
	}

	err = m.Down()
	if err != nil && err != migrate.ErrNoChange {
		t.Fatal(err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		t.Fatal(err)
	}

	if err := populateTestDB(db); err != nil {
		t.Fatal(err)
	}

	return testHarness{db: db}
}

func populateTestDB(pg *sql.DB) error {
	_, err := pg.Exec(`
		INSERT INTO
			decks (id, author_id, title, description, created_at)
		VALUES (
			'fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72',
			'4e37a600-c29e-4d0f-af44-66f2cd8cc1c9',
			'Programming languages',
			'Compiled or interpreted?',
			NOW()
		);

		INSERT INTO
			cards (deck_id, id, title)
		VALUES (
			'fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72',
			'72bdff92-5bc8-4e1d-9217-d0b23e22ff33',
			'Golang'
		),
		(
			'fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72',
			'c924f7e0-efd8-4c2d-9c43-8eafb7102ebc',
			'Rust'
		),
		(
			'fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72',
			'd42a90dd-818c-4eed-8e9f-9e8af1a654f4',
			'Lua'
		);

		INSERT INTO
			answers (card_id, id, text, is_correct)
		VALUES (
			'72bdff92-5bc8-4e1d-9217-d0b23e22ff33',
			'7e6926da-82b2-4ae8-99b4-1b803ebf1877',
			'Compiled',
			true
		),
		(
			'72bdff92-5bc8-4e1d-9217-d0b23e22ff33',
			'dfcb1c81-f590-486e-9b7e-a44f0c436933',
			'Interpreted',
			false
		),
		(
			'c924f7e0-efd8-4c2d-9c43-8eafb7102ebc',
			'06be1892-4765-4f60-9d47-1489419dc316',
			'Compiled',
			true
		),
		(
			'c924f7e0-efd8-4c2d-9c43-8eafb7102ebc',
			'9403ad3e-45e6-4b23-8f63-b751de8576cc',
			'Interpreted',
			false
		),
		(
			'd42a90dd-818c-4eed-8e9f-9e8af1a654f4',
			'3b1bbdb3-b84a-4f59-8f02-2a21586cf6ca',
			'Compiled',
			false
		),
		(
			'd42a90dd-818c-4eed-8e9f-9e8af1a654f4',
			'd23d0201-55f3-40da-8718-853a6cea419d',
			'Interpreted',
			true
		);`,
	)

	return errors.Trace(err)
}
