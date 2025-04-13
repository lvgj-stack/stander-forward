package selector

import (
	"time"
)

var _description = `
strategy (string, default=round)
指定选择策略。

round - 轮询
rand - 随机
fifo - 自上而下，主备模式
hash - 基于特定Hash值(客户端IP或目标地址)
maxFails (int, default=1)
指定最大失败次数，当失败次数超过此设定值时，此对象会被标记为失败(Fail)状态，失败状态的对象不会被选择使用。
failTimeout (duration, default=10s)
指定失败状态的超时时长，当一个对象被标记为失败后，在此设定的时间间隔内不会被选择使用，超过此设定时间间隔后，会再次参与选择。
`

func WithStrategy(strategy string) Option {
	return func(s *Selector) {
		s.strategy = strategy
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(s *Selector) {
		s.timeout = timeout
	}
}

func WithMaxFail(maxFail int64) Option {
	return func(s *Selector) {
		s.maxFail = maxFail
	}
}

type Option func(*Selector)

type Selector struct {
	strategy string
	timeout  time.Duration
	maxFail  int64
}

func NewSelector(ops ...Option) *Selector {
	s := newDefaultSelector()
	for _, op := range ops {
		op(s)
	}
	return s
}

func newDefaultSelector() *Selector {
	return &Selector{
		strategy: "fifo",
		timeout:  10 * time.Second,
		maxFail:  3,
	}
}
