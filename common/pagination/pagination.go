package pagination

import (
	"encoding/base64"
	"encoding/json"

	"github.com/juju/errors"
)

type Cursor string

func (c Cursor) String() string {
	return string(c)
}

func (c Cursor) IsEmpty() bool {
	return c == ""
}

func ToCursor(v any) (Cursor, error) {
	out, err := json.Marshal(v)
	if err != nil {
		return Cursor(""), errors.Trace(err)
	}

	encoded := base64.StdEncoding.EncodeToString(out)

	return Cursor(encoded), nil
}

func FromCursor(c Cursor, out any) error {
	decoded, err := base64.StdEncoding.DecodeString(c.String())
	if err != nil {
		return errors.Trace(err)
	}

	err = json.Unmarshal(decoded, out)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

type PageInfo struct {
	HasPreviousPage bool
	HasNextPage     bool
	StartCursor     Cursor
	EndCursor       Cursor
}

type Pagination struct {
	Before Cursor
	After  Cursor
	First  int
	Last   int
}

// Limit returns the pagination's limit.
// By default, 1 is returned if no limit was specified.
func (p *Pagination) Limit() int {
	var limit int
	if p.IsForward() {
		limit = p.First
	} else {
		limit = p.Last
	}

	if limit == 0 {
		return 1
	}

	return limit
}

func (p *Pagination) Cursor() Cursor {
	if p.IsForward() {
		return p.After
	}

	return p.Before
}

func (p *Pagination) OrderBy() string {
	if p.IsForward() {
		return "DESC"
	}

	return "ASC"
}

func (p *Pagination) Comparator() string {
	if p.IsForward() {
		return "<"
	}

	return ">"
}

func (p *Pagination) IsForward() bool {
	return p.Last == 0
}
