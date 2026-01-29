package shutdown

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Logger interface {
	Info(msg string, fields ...any)
	Error(msg string, fields ...any)
}

type Manager struct {
	mu      sync.Mutex
	closers []func(context.Context) error
}

func New() *Manager {
	return &Manager{}
}

func (m *Manager) Add(fn func(context.Context) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closers = append(m.closers, fn)
}

func WaitSignal() os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	return <-ch
}

func (m *Manager) Shutdown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	m.mu.Lock()
	closers := make([]func(context.Context) error, len(m.closers))
	copy(closers, m.closers)
	m.mu.Unlock()

	var errs []error
	for i := len(closers) - 1; i >= 0; i-- {
		if err := closers[i](ctx); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
