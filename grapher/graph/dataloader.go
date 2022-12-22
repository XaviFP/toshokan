package graph

import (
	"context"
	"sync"
	"time"
)

type DataLoader interface {
	Load(ctx context.Context, id string) (any, error)
}

func NewDataLoader(batchFn func(ctx context.Context, ids []string) map[string]any, cacheTTL, batchEvery time.Duration) DataLoader {
	loader := loader{
		cacheTTL:   cacheTTL,
		batchEvery: batchEvery,
		batchFn:    batchFn,
		elements:   make(map[string]*ttlElement),
		buffer:     make(map[string]chan any),
	}

	loader.Start()

	return &loader
}

type ttlElement struct {
	element any
	ttl     time.Time
	id      string
}

type loader struct {
	queue      []string
	elements   map[string]*ttlElement
	buffer     map[string]chan any
	m          sync.Mutex
	cacheTTL   time.Duration
	batchEvery time.Duration
	batchFn    func(ctx context.Context, ids []string) map[string]any
}

func (l *loader) Load(ctx context.Context, id string) (any, error) {
	d, ok := l.elements[id]
	if ok {
		return d.element, nil
	}

	ch := l.add(id)

	out := <-ch

	delete(l.buffer, id)

	return out, nil
}

func (l *loader) add(id string) chan any {
	l.m.Lock()
	defer l.m.Unlock()
	l.queue = append(l.queue, id)
	ch := make(chan any)
	l.buffer[id] = ch

	return ch
}

func (l *loader) batch(ctx context.Context) {
	l.m.Lock()
	defer l.m.Unlock()

	for id, e := range l.batchFn(ctx, l.queue) {
		l.elements[id] = &ttlElement{
			id:      id,
			element: e,
			ttl:     time.Now().Add(l.cacheTTL),
		}
		if ch, ok := l.buffer[id]; ok {
			ch <- e
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
	for _, e := range l.elements {
		if e.ttl.After(now) {
			delete(l.elements, e.id)
		}
	}
}
