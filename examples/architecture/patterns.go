package architecture

import (
	"context"
	"sync"
)

type Notifier interface {
	Notify(context.Context, string) error
}

func SendWelcome(ctx context.Context, notifier Notifier, address string) error {
	return notifier.Notify(ctx, address)
}

type IdempotentCommands struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func (c *IdempotentCommands) Handle(key string, effect func() error) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.seen == nil {
		c.seen = make(map[string]struct{})
	}
	if _, ok := c.seen[key]; ok {
		return nil
	}
	if err := effect(); err != nil {
		return err
	}
	c.seen[key] = struct{}{}
	return nil
}
