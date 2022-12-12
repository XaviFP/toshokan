package user

import (
	"context"
	"database/sql"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"

	"github.com/XaviFP/toshokan/common/config"
	"github.com/XaviFP/toshokan/common/db"
)

func TestUserRepository_CreateUser(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(h.db)

	t.Run("success", func(t *testing.T) {
		req := CreateUserRequest{Username: "Aunt May", Password: "XXX", Bio: "", Nick: "Auntie"}
		expectedUser, err := repo.Create(context.Background(), req)
		assert.NoError(t, err)

		var out User
		var password []byte
		row := h.db.QueryRow(`
			SELECT id, username, nick, bio, password
			FROM users
			WHERE username = $1 AND deleted_at IS NULL`,
			req.Username,
		)
		err = row.Scan(&out.ID, &out.Username, &out.Nick, &out.Bio, &password)
		assert.NoError(t, err)
		assert.Equal(t, expectedUser, out)

		err = bcrypt.CompareHashAndPassword(password, []byte("XXX"))
		assert.NoError(t, err)
	})

	t.Run("failure", func(t *testing.T) {
		_, err := repo.Create(context.Background(), CreateUserRequest{Username: "Uncle Ben"})
		assert.ErrorIs(t, err, ErrUserAlreadyExists)
	})
}

func TestUserRepository_GetUSerByID(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(h.db)

	t.Run("success", func(t *testing.T) {
		id := uuid.MustParse("4e37a600-c29e-4d0f-af44-66f2cd8cc1c9")
		u, err := repo.GetUserByID(context.Background(), id)
		assert.NoError(t, err)
		assert.Equal(t, id, u.ID)
	})

	t.Run("failure", func(t *testing.T) {
		id := uuid.MustParse("f60b2fa3-4b79-488f-9341-46123dd7332b")
		u, err := repo.GetUserByID(context.Background(), id)
		assert.Error(t, err)
		assert.Equal(t, uuid.UUID{}, u.ID)
	})
}

func TestUserRepository_GetUSerByUsername(t *testing.T) {
	h := newTestHarness(t)
	repo := NewPGRepository(h.db)

	t.Run("success", func(t *testing.T) {
		username := "Uncle Ben"
		u, err := repo.GetUserByUsername(context.Background(), username)
		assert.NoError(t, err)
		assert.Equal(t, username, u.Username)
	})

	t.Run("failure", func(t *testing.T) {
		username := "Aunt May"
		u, err := repo.GetUserByUsername(context.Background(), username)
		assert.Error(t, err)
		assert.Equal(t, User{}, u)
	})
}

type testHarness struct {
	db *sql.DB
}

func newTestHarness(t *testing.T) testHarness {
	db, err := db.InitDB(config.DBConfig{User: "toshokan", Password: "t.o.s.h.o.k.a.n.", Name: "test_user", Host: "localhost", Port: "5432"})
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
			users (id, username, nick, password, bio, created_at)
		VALUES(
			'4e37a600-c29e-4d0f-af44-66f2cd8cc1c9', 
			'Uncle Ben',
			'Benny',
			'$2y$10$FfS13P1/T94IBUkbWasd6.siVGANuS0THIDerH9gs9C6Ybj5g/gou', 
			'Great power comes with great responsability.', 
			NOW()
		);`,
	)

	return errors.Trace(err)
}
