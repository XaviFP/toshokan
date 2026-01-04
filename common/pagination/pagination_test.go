package pagination

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
