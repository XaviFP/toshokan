package graph

import (
	"context"
	"sync"
	"time"

	"github.com/juju/errors"
)

var ErrNoResult = errors.New("No result returned")

type DataLoader interface {
	Load(ctx context.Context, id string) (any, error)
}

type BatchFn func(ctx context.Context, ids []string) (map[string]Result, error)

func NewDataLoader(batchFn BatchFn, cacheTTL, batchEvery time.Duration) DataLoader {
	loader := loader{
		cacheTTL:   cacheTTL,
		batchEvery: batchEvery,
		batchFn:    batchFn,
		cache:      make(map[string]*ttlResult),
		buffer:     make(map[string]chan Result),
	}

	loader.Start()

	return &loader
}

type ttlResult struct {
	result Result
	ttl    time.Time
	id     string
}

type Result struct {
	Value any
	Error error
}

type loader struct {
	queue      []string
	cache      map[string]*ttlResult
	buffer     map[string]chan Result
	m          sync.Mutex
	cacheTTL   time.Duration
	batchEvery time.Duration
	batchFn    BatchFn
}

func (l *loader) Load(ctx context.Context, id string) (any, error) {
	c, ok := l.cache[id]
	if ok {
		return c.result.Value, c.result.Error
	}

	ch := l.add(id)

	out := <-ch

	delete(l.buffer, id)

	return out.Value, out.Error
}

func (l *loader) add(id string) chan Result {
	l.m.Lock()
	defer l.m.Unlock()
	l.queue = append(l.queue, id)
	ch := make(chan Result)
	l.buffer[id] = ch

	return ch
}

func (l *loader) batch(ctx context.Context) {
	l.m.Lock()
	defer l.m.Unlock()

	results, err := l.batchFn(ctx, l.queue)
	if err != nil {
		for _, id := range l.queue {
			if ch, ok := l.buffer[id]; ok {
				ch <- Result{Error: err}
			}
		}

		return
	}

	// Traverse `l.queue` instead of `results`
	// Can't trust batchFn to return a result for each of the ids
	for _, id := range l.queue {
		res, ok := results[id]
		if !ok {
			res = Result{Error: ErrNoResult}
		} else {
			l.cache[id] = &ttlResult{
				id:     id,
				result: res,
				ttl:    time.Now().Add(l.cacheTTL),
			}
		}

		if ch, ok := l.buffer[id]; ok {
			ch <- res
		}
	}

	l.queue = []string{}
}

func (l *loader) Start() {
	batchTicker := time.NewTicker(l.batchEvery)
	expireTicker := time.NewTicker(l.cacheTTL)
	go func() {
		for {
			select {
			case <-batchTicker.C:
				if len(l.queue) > 0 {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
					l.batch(ctx)
					cancel()
				}
			case <-expireTicker.C:
				l.expire()
			}
		}
	}()
}

func (l *loader) expire() {
	l.m.Lock()
	defer l.m.Unlock()

	now := time.Now()
	for _, res := range l.cache {
		if res.ttl.After(now) {
			delete(l.cache, res.id)
		}
	}
}
