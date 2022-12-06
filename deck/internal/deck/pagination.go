package deck

import (
	"time"

	"github.com/google/uuid"

	"github.com/XaviFP/toshokan/common/pagination"
)

type Cursor struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

type PopularDeckEdge struct {
	DeckID uuid.UUID
	Cursor pagination.Cursor
}

type PopularDecksConnection struct {
	Edges    []PopularDeckEdge
	PageInfo pagination.PageInfo
}
