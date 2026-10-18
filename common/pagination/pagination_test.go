package pagination

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPagination(t *testing.T) {
	t.Run("order_by", func(t *testing.T) {
		tests := []struct {
			name       string
			pagination Pagination
			expected   string
		}{
			{
				name: "forward_pagination_newest_first",
				pagination: Pagination{
					First: 10,
				},
				expected: "DESC",
			},
			{
				name: "backward_pagination_newest_first",
				pagination: Pagination{
					Last: 10,
				},
				expected: "ASC",
			},
			{
				name: "forward_pagination_oldest_first",
				pagination: Pagination{
					First: 10,
					Kind:  PaginationKindOldestFirst,
				},
				expected: "ASC",
			},
			{
				name: "backward_pagination_oldest_first",
				pagination: Pagination{
					Last: 10,
					Kind: PaginationKindOldestFirst,
				},
				expected: "DESC",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.pagination.OrderBy()
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("comparator", func(t *testing.T) {
		tests := []struct {
			name       string
			pagination Pagination
			expected   string
		}{
			{
				name: "forward_pagination_newest_first",
				pagination: Pagination{
					First: 10,
				},
				expected: "<",
			},
			{
				name: "backward_pagination_newest_first",
				pagination: Pagination{
					Last: 10,
				},
				expected: ">",
			},
			{
				name: "forward_pagination_oldest_first",
				pagination: Pagination{
					First: 10,
					Kind:  PaginationKindOldestFirst,
				},
				expected: ">",
			},
			{
				name: "backward_pagination_oldest_first",
				pagination: Pagination{
					Last: 10,
					Kind: PaginationKindOldestFirst,
				},
				expected: "<",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := tt.pagination.Comparator()
				assert.Equal(t, tt.expected, result)
			})
		}
	})
}

func TestFromCursor(t *testing.T) {
	type cursorData struct {
		Order int `json:"order"`
	}

	t.Run("valid", func(t *testing.T) {
		cursor, err := ToCursor(cursorData{Order: 3})
		require.NoError(t, err)

		var actual cursorData
		require.NoError(t, FromCursor(cursor, &actual))
		assert.Equal(t, cursorData{Order: 3}, actual)
	})

	t.Run("invalid_base64", func(t *testing.T) {
		var actual cursorData
		err := FromCursor(Cursor("Ây\xf3\x81"), &actual)
		assert.ErrorIs(t, err, ErrInvalidCursor)
	})

	t.Run("invalid_json", func(t *testing.T) {
		var actual cursorData
		err := FromCursor(Cursor(base64.StdEncoding.EncodeToString([]byte("not json"))), &actual)
		assert.ErrorIs(t, err, ErrInvalidCursor)
	})
}
