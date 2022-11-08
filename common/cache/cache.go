package cache

import (
	"context"

	"github.com/juju/errors"
	"github.com/mediocregopher/radix/v4"
)

var ErrNoValueForKey = errors.New("cache: no value found for the given key")

type Cache interface {
	SetEx(ctx context.Context, key, value string, seconds uint) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
}

type cache struct {
	redis radix.Client
}

func NewCache(redis radix.Client) Cache {
	return &cache{redis: redis}
}

func (c *cache) SetEx(ctx context.Context, key, value string, seconds uint) error {
	err := c.redis.Do(
		ctx,
		radix.FlatCmd(nil, "SETEX", key, seconds, value),
	)

	return errors.Trace(err)
}

func (c *cache) Get(ctx context.Context, key string) (string, error) {
	var out string

	mb := radix.Maybe{Rcv: &out}
	if err := c.redis.Do(ctx, radix.Cmd(&mb, "GET", key)); err != nil {
		return "", errors.Trace(err)
	}

	if mb.Null {
		return "", ErrNoValueForKey
	}

	return out, nil
}

func (c *cache) Delete(ctx context.Context, key string) error {
	err := c.redis.Do(ctx, radix.Cmd(nil, "DEL", key))

	return errors.Trace(err)
}
