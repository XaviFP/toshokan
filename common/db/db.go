package db

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/XaviFP/toshokan/common/config"
)

func InitDB(c config.DBConfig) (*sql.DB, error) {
	psqlconn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable", c.User, c.Password, c.Name, c.Host, c.Port)
	var err error
	db, err := sql.Open("postgres", psqlconn)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return db, err
}

func IsConstraintError(err error, constraintName string) bool {
	if err == nil {
		return false
	}

	pgErr, ok := errors.Cause(err).(*pq.Error)
	if !ok {
		return false
	}

	return pgErr.Constraint == constraintName
}

type Argumenter struct {
	values []any
}

func (a *Argumenter) Add(v any) string {
	a.values = append(a.values, v)

	return fmt.Sprintf("$%d", len(a.values))
}

func (a *Argumenter) Values() []any {
	return a.values
}
