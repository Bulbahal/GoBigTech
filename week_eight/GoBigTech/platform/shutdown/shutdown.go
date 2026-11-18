package shutdown

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestManager_ShutdownOrderAndErrors(t *testing.T) {
	m := New()

	var order []string
	var mu sync.Mutex

	errFirst := errors.New("first closer failed")
	errThird := errors.New("third closer failed")

	m.Add(func(ctx context.Context) error {
		mu.Lock()
		order = append(order, "first")
		mu.Unlock()
		return errFirst
	})

	m.Add(func(ctx context.Context) error {
		mu.Lock()
		order = append(order, "second")
		mu.Unlock()
		return nil
	})

	m.Add(func(ctx context.Context) error {
		mu.Lock()
		order = append(order, "third")
		mu.Unlock()
		return errThird
	})

	err := m.Shutdown(time.Second)
	if !errors.Is(err, errFirst) || !errors.Is(err, errThird) {
		t.Fatalf("expected joined errors, got: %v", err)
	}

	expectedOrder := []string{"third", "second", "first"}
	if !reflect.DeepEqual(order, expectedOrder) {
		t.Errorf("wrong shutdown order: got %v, want %v", order, expectedOrder)
	}
}

func TestManager_Timeout(t *testing.T) {
	m := New()

	m.Add(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	err := m.Shutdown(30 * time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestManager_ConcurrentAdd(t *testing.T) {
	m := New()

	const workers = 20
	var wg sync.WaitGroup
	wg.Add(workers)

	var count int
	var mu sync.Mutex

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			m.Add(func(ctx context.Context) error {
				mu.Lock()
				count++
				mu.Unlock()
				return nil
			})
		}()
	}

	wg.Wait()

	if err := m.Shutdown(time.Second); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != workers {
		t.Errorf("expected %d closers executed, got %d", workers, count)
	}
}
