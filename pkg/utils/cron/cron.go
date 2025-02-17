package cron

import (
	"context"
	"time"
)

type Cron struct {
	Interval time.Duration
	StopC    chan struct{}
}

func New(interval time.Duration) *Cron {
	return &Cron{
		Interval: interval,
		StopC:    make(chan struct{}),
	}
}

func (c *Cron) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	tk := time.NewTicker(c.Interval)
	defer tk.Stop()
	for {
		select {
		case <-tk.C:
			if err := fn(ctx); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		case <-c.StopC:
			return nil
		}
	}
}
