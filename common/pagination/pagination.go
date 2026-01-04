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

func NewOlderstFistPagination(opts ...PaginationOpts) Pagination {
	p := Pagination{}

	for _, opt := range opts {
		opt(&p)
	}

	p.Kind = PaginationKindOldestFirst

	return p
}

type PaginationOpts func(p *Pagination)

func WithBefore(c Cursor) PaginationOpts {
	return func(p *Pagination) {
		p.Before = c
	}
}

func WithAfter(c Cursor) PaginationOpts {
	return func(p *Pagination) {
		p.After = c
	}
}

func WithFirst(i int) PaginationOpts {
	return func(p *Pagination) {
		p.First = i
	}
}

func WithLast(i int) PaginationOpts {
	return func(p *Pagination) {
		p.Last = i
	}
}

type Pagination struct {
	Before Cursor
	After  Cursor
	First  int
	Last   int
	Kind   PaginationKind
}

type PaginationKind int

const (
	PaginationKindNewestFirst PaginationKind = iota // TODO: Alternative names. KindASC, KindDESC> FirstFirst, FirstLast
	PaginationKindOldestFirst
)

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
	if p.Kind == PaginationKindOldestFirst {
		if p.IsForward() {
			return "ASC"
		}
		return "DESC"
	}

	if p.IsForward() {
		return "DESC"
	}

	return "ASC"
}

func (p *Pagination) Comparator() string {
	if p.Kind == PaginationKindOldestFirst {
		if p.IsForward() {
			return ">"
		}
		return "<"
	}

	if p.IsForward() {
		return "<"
	}

	return ">"
}

func (p *Pagination) IsForward() bool {
	return p.Last == 0
}
