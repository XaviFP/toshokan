package deck

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/XaviFP/toshokan/common/config"
	"github.com/XaviFP/toshokan/common/db"
	"github.com/XaviFP/toshokan/common/pagination"

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

func TestRepository_StoreCard(t *testing.T) {
	h := newTestHarness(t)

	repo := NewPGRepository(h.db)

	t.Run("success", func(t *testing.T) {
		dID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

		cards, err := repo.GetDeckCards(context.Background(), dID)
		assert.NoError(t, err)

		nCards := len(cards)

		card := Card{
			ID:    uuid.MustParse("bc8a13b3-c257-497f-9e80-02e9a50a2fbe"),
			Title: "Which is faster?",
			PossibleAnswers: []Answer{
				{
					ID:        uuid.MustParse("a0f3a2e4-7263-855d-ebcf-fffa0a96f450"),
					Text:      "Compiled",
					IsCorrect: true,
				},
				{
					ID:        uuid.MustParse("a0f3a2e4-7263-855d-ebcf-fffa0a96f451"),
					Text:      "Interpreted",
					IsCorrect: false,
				},
			},
			Explanation: "Compiled languages are faster",
		}
		err = repo.StoreCard(context.Background(), card, dID)
		assert.NoError(t, err)

		actualCards, err := repo.GetDeckCards(context.Background(), dID)
		assert.NoError(t, err)

		assert.Equal(t, nCards+1, len(actualCards))

		err = repo.StoreCard(context.Background(), card, dID)
		assert.ErrorIs(t, err, ErrCardAlreadyExists)

	})

	t.Run("failure", func(t *testing.T) {
		dID := uuid.MustParse("ebcfffa0-a96f-450b-a0f3-a2e47263855d")

		card := Card{
			ID:    uuid.MustParse("19cc0f18-70d6-49d2-85d4-ef36b9f4579c"),
			Title: "Which is faster?",
			PossibleAnswers: []Answer{
				{
					ID:        uuid.MustParse("a0f3a2e4-7263-855d-ebcf-fffa0a96f450"),
					Text:      "Compiled",
					IsCorrect: true,
				},
				{
					ID:        uuid.MustParse("a0f3a2e4-7263-855d-ebcf-fffa0a96f451"),
					Text:      "Interpreted",
					IsCorrect: false,
				},
			},
			Explanation: "Compiled languages are faster",
		}

		err := repo.StoreCard(context.Background(), card, dID)
		//d, err := repo.GetDeck(context.Background(), id)
		assert.ErrorIs(t, err, ErrDeckNotFound)
		cards, _ := repo.GetCards(context.Background(), []uuid.UUID{card.ID})
		assert.Empty(t, cards)
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
			Public:      true,
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
		expected := map[uuid.UUID]Deck{
			uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72"): {
				ID:          uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72"),
				AuthorID:    uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9"),
				Title:       "Programming languages",
				Description: "Compiled or interpreted?",
				Public:      true,
			},
			uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c"): {
				ID:          uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c"),
				AuthorID:    uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9"),
				Title:       "Greek Mythology",
				Description: "Bits of Greek Mythology",
				Public:      true,
			},
		}

		actual, err := repo.GetDecks(context.Background(), []uuid.UUID{
			uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72"),
			uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c"),
		})
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("empty", func(t *testing.T) {
		decks, err := repo.GetDecks(context.Background(), []uuid.UUID{})
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
				ID:          uuid.MustParse("72bdff92-5bc8-4e1d-9217-d0b23e22ff33"),
				Title:       "Golang",
				Explanation: "Go code is compiled directly to machine code, not interpreted at runtime.",
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

func TestRepository_GetPopularDecks(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(h.db)
	cursors := []Cursor{
		{
			ID:        uuid.MustParse("f79aea77-9aa0-4a84-b4c8-d000a27d2c52"),
			CreatedAt: time.Date(2000, time.January, 5, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        uuid.MustParse("6363e2c6-d89e-4610-92e8-1e1d2fea49ec"),
			CreatedAt: time.Date(2000, time.January, 4, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        uuid.MustParse("60766223-ff9f-4871-a497-f765c05a0c5e"),
			CreatedAt: time.Date(2000, time.January, 3, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        uuid.MustParse("334ddbf8-1acc-405b-86d8-49f0d1ca636c"),
			CreatedAt: time.Date(2000, time.January, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			ID:        uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72"),
			CreatedAt: time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	userID := uuid.MustParse("fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72")

	t.Run("forward_pagination", func(t *testing.T) {
		conn, err := repo.GetPopularDecks(context.Background(), userID, pagination.Pagination{First: 2})
		assert.NoError(t, err)

		assert.Equal(t, PopularDecksConnection{
			Edges: []PopularDeckEdge{
				{
					DeckID: cursors[0].ID,
					Cursor: mustToCursor(t, cursors[0]),
				},
				{
					DeckID: cursors[1].ID,
					Cursor: mustToCursor(t, cursors[1]),
				},
			},
			PageInfo: pagination.PageInfo{
				HasNextPage: true,
				StartCursor: mustToCursor(t, cursors[0]),
				EndCursor:   mustToCursor(t, cursors[1]),
			},
		}, conn)

		conn, err = repo.GetPopularDecks(context.Background(), userID, pagination.Pagination{First: 2, After: conn.PageInfo.EndCursor})
		assert.NoError(t, err)

		assert.Equal(t, PopularDecksConnection{
			Edges: []PopularDeckEdge{
				{
					DeckID: cursors[2].ID,
					Cursor: mustToCursor(t, cursors[2]),
				},
				{
					DeckID: cursors[3].ID,
					Cursor: mustToCursor(t, cursors[3]),
				},
			},
			PageInfo: pagination.PageInfo{
				HasNextPage: true,
				StartCursor: mustToCursor(t, cursors[2]),
				EndCursor:   mustToCursor(t, cursors[3]),
			},
		}, conn)

		conn, err = repo.GetPopularDecks(context.Background(), userID, pagination.Pagination{First: 2, After: conn.PageInfo.EndCursor})
		assert.NoError(t, err)

		assert.Equal(t, PopularDecksConnection{
			Edges: []PopularDeckEdge{
				{
					DeckID: cursors[4].ID,
					Cursor: mustToCursor(t, cursors[4]),
				},
			},
			PageInfo: pagination.PageInfo{
				HasNextPage: false,
			},
		}, conn)
	})

	t.Run("backward_pagination", func(t *testing.T) {
		conn, err := repo.GetPopularDecks(context.Background(), userID, pagination.Pagination{Last: 2})
		assert.NoError(t, err)

		assert.Equal(t, PopularDecksConnection{
			Edges: []PopularDeckEdge{
				{
					DeckID: cursors[3].ID,
					Cursor: mustToCursor(t, cursors[3]),
				},
				{
					DeckID: cursors[4].ID,
					Cursor: mustToCursor(t, cursors[4]),
				},
			},
			PageInfo: pagination.PageInfo{
				HasPreviousPage: true,
				StartCursor:     mustToCursor(t, cursors[3]),
				EndCursor:       mustToCursor(t, cursors[4]),
			},
		}, conn)

		conn, err = repo.GetPopularDecks(context.Background(), userID, pagination.Pagination{Last: 2, Before: conn.PageInfo.StartCursor})
		assert.NoError(t, err)

		assert.Equal(t, PopularDecksConnection{
			Edges: []PopularDeckEdge{
				{
					DeckID: cursors[1].ID,
					Cursor: mustToCursor(t, cursors[1]),
				},
				{
					DeckID: cursors[2].ID,
					Cursor: mustToCursor(t, cursors[2]),
				},
			},
			PageInfo: pagination.PageInfo{
				HasPreviousPage: true,
				StartCursor:     mustToCursor(t, cursors[1]),
				EndCursor:       mustToCursor(t, cursors[2]),
			},
		}, conn)

		conn, err = repo.GetPopularDecks(context.Background(), userID, pagination.Pagination{Last: 2, Before: conn.PageInfo.StartCursor})
		assert.NoError(t, err)

		assert.Equal(t, PopularDecksConnection{
			Edges: []PopularDeckEdge{
				{
					DeckID: cursors[0].ID,
					Cursor: mustToCursor(t, cursors[0]),
				},
			},
			PageInfo: pagination.PageInfo{
				HasPreviousPage: false,
			},
		}, conn)
	})

	t.Run("ErrInvalidCursor", func(t *testing.T) {
		_, err := repo.GetPopularDecks(context.Background(), userID, pagination.Pagination{After: pagination.Cursor("This Cursor is not valid")})
		assert.ErrorIs(t, err, ErrInvalidCursor)
	})
}

func TestPGRepository_GetCards(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(h.db)

	id := uuid.MustParse("72bdff92-5bc8-4e1d-9217-d0b23e22ff33")

	t.Run("success", func(t *testing.T) {
		expected := map[uuid.UUID]Card{
			id: {
				ID:          id,
				Title:       "Golang",
				Explanation: "Go code is compiled directly to machine code, not interpreted at runtime.",
				PossibleAnswers: []Answer{
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
				},
			},
		}

		actual, err := repo.GetCards(context.Background(), []uuid.UUID{id})
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("empty", func(t *testing.T) {
		cards, err := repo.GetCards(context.Background(), []uuid.UUID{})
		assert.NoError(t, err)
		assert.Empty(t, cards)
	})

	t.Run("not_exists", func(t *testing.T) {
		id := uuid.MustParse("dfcb1c81-f590-486e-9b7e-a44f0c436933")
		cards, err := repo.GetCards(context.Background(), []uuid.UUID{id})
		assert.NoError(t, err)
		assert.Empty(t, cards)
	})
}

type testHarness struct {
	db *sql.DB
}

func newTestHarness(t *testing.T) testHarness {
	db, err := db.InitDB(config.DBConfig{User: "toshokan", Password: "t.o.s.h.o.k.a.n.", Name: "test_deck", Host: "localhost", Port: "5432"})
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
			decks (id, author_id, title, description, created_at, is_public)
		VALUES (
			'fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72',
			'4e37a600-c29e-4d0f-af44-66f2cd8cc1c9',
			'Programming languages',
			'Compiled or interpreted?',
			'2000-01-01',
			true
		),
		(
			'334ddbf8-1acc-405b-86d8-49f0d1ca636c',
			'4e37a600-c29e-4d0f-af44-66f2cd8cc1c9',
			'Greek Mythology',
			'Bits of Greek Mythology',
			'2000-01-02',
			true
		),
		(
			'60766223-ff9f-4871-a497-f765c05a0c5e',
			'4e37a600-c29e-4d0f-af44-66f2cd8cc1c9',
			'Biology 101',
			'The Biology Beginners Course',
			'2000-01-03',
			true
		),
		(
			'6363e2c6-d89e-4610-92e8-1e1d2fea49ec',
			'4e37a600-c29e-4d0f-af44-66f2cd8cc1c9',
			'Presocratic Philosophy II',
			'Advanced Presocratic Philosophy',
			'2000-01-04',
			true
		),
		(
			'f79aea77-9aa0-4a84-b4c8-d000a27d2c52',
			'4e37a600-c29e-4d0f-af44-66f2cd8cc1c9',
			'Music Theory',
			'From Zero to Hero',
			'2000-01-05',
			true
		);

		INSERT INTO
			cards (deck_id, id, title, explanation)
		VALUES (
			'fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72',
			'72bdff92-5bc8-4e1d-9217-d0b23e22ff33',
			'Golang',
			'Go code is compiled directly to machine code, not interpreted at runtime.'
		),
		(
			'fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72',
			'c924f7e0-efd8-4c2d-9c43-8eafb7102ebc',
			'Rust',
			''
		),
		(
			'fb9ffe2c-ad66-4766-9b7b-46fd5d9acd72',
			'd42a90dd-818c-4eed-8e9f-9e8af1a654f4',
			'Lua',
			''
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

func mustToCursor(t *testing.T, v any) pagination.Cursor {
	out, err := pagination.ToCursor(v)
	assert.NoError(t, err)

	return out
}
