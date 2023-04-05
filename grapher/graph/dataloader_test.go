package graph

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDataloader(t *testing.T) {
	batchCalls := 0
	loader := NewDataLoader(
		newTestBatchFn(&batchCalls),
		time.Second,
		time.Millisecond,
	)

	result, err := loader.Load(context.Background(), "test")
	assert.Nil(t, err)
	assert.Equal(t, "test", result)

	result, err = loader.Load(context.Background(), "test")
	assert.Nil(t, err)
	assert.Equal(t, "test", result)

	// Cache did not expire yet so the second Load result came from the cache
	assert.Equal(t, 1, batchCalls)

	time.Sleep(time.Second)

	result, err = loader.Load(context.Background(), "test")
	assert.Nil(t, err)
	assert.Equal(t, "test", result)

	// Cache expired once and Load triggered another batch call
	assert.Equal(t, 2, batchCalls)
}

func newTestBatchFn(counter *int) BatchFn {
	return func(ctx context.Context, ids []string) (map[string]Result, error) {
		out := make(map[string]Result, len(ids))

		*counter++
		for _, id := range ids {
			out[id] = Result{Value: id}
		}

		return out, nil
	}
}
