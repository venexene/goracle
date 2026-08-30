package architecture

import (
	"context"
	"errors"
	"testing"
)

type notifyFunc func(context.Context, string) error

func (f notifyFunc) Notify(ctx context.Context, address string) error {
	return f(ctx, address)
}

func TestConsumerOwnedInterfaceCanBeSubstituted(t *testing.T) {
	var got string
	err := SendWelcome(t.Context(), notifyFunc(func(_ context.Context, address string) error {
		got = address
		return nil
	}), "student@example.test")
	if err != nil || got != "student@example.test" {
		t.Fatalf("address=%q err=%v", got, err)
	}
}

func TestIdempotentCommandAppliesEffectOnce(t *testing.T) {
	commands := &IdempotentCommands{}
	var calls int
	for range 2 {
		if err := commands.Handle("request-1", func() error { calls++; return nil }); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("effect calls = %d", calls)
	}
}

func TestFailedCommandCanBeRetried(t *testing.T) {
	commands := &IdempotentCommands{}
	want := errors.New("temporary")
	if err := commands.Handle("request-1", func() error { return want }); !errors.Is(err, want) {
		t.Fatalf("error = %v", err)
	}
	var calls int
	if err := commands.Handle("request-1", func() error { calls++; return nil }); err != nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}
