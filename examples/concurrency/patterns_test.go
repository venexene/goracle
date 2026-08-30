package concurrency

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

func TestMap(t *testing.T) {
	input := make(chan int, 3)
	for _, value := range []int{1, 2, 3} {
		input <- value
	}
	close(input)
	var got []int
	for value := range Map(t.Context(), input, func(value int) int { return value * value }) {
		got = append(got, value)
	}
	if len(got) != 3 || got[2] != 9 {
		t.Fatalf("got %v", got)
	}
}

func TestRunLimited(t *testing.T) {
	var running, maximum atomic.Int64
	err := RunLimited(t.Context(), 2, []int{1, 2, 3, 4}, func(context.Context, int) error {
		current := running.Add(1)
		defer running.Add(-1)
		for old := maximum.Load(); current > old && !maximum.CompareAndSwap(old, current); old = maximum.Load() {
		}
		return nil
	})
	if err != nil || maximum.Load() > 2 {
		t.Fatalf("err=%v max=%d", err, maximum.Load())
	}
}

func TestRunLimitedReturnsCause(t *testing.T) {
	want := errors.New("failed")
	err := RunLimited(t.Context(), 1, []int{1}, func(context.Context, int) error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("err=%v", err)
	}
}

func TestRunLimitedWaitsForStartedWork(t *testing.T) {
	want := errors.New("failed")
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		done <- RunLimited(t.Context(), 2, []int{0, 1, 2}, func(_ context.Context, job int) error {
			if job == 0 {
				close(started)
				<-release
				return nil
			}
			return want
		})
	}()

	<-started
	select {
	case err := <-done:
		t.Fatalf("RunLimited returned before started work finished: %v", err)
	default:
	}
	close(release)
	if err := <-done; !errors.Is(err, want) {
		t.Fatalf("err=%v", err)
	}
}
