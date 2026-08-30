package language

import (
	"context"
	"errors"
	"math"
	"testing"
)

type closeFunc func() error

func (f closeFunc) Close() error { return f() }

type typedError struct{}

func (*typedError) Error() string { return "typed" }

func TestAddUint64(t *testing.T) {
	if _, err := AddUint64(math.MaxUint64, 1); err == nil {
		t.Fatal("overflow must be rejected")
	}
	if got, err := AddUint64(20, 22); err != nil || got != 42 {
		t.Fatalf("got %d, err %v", got, err)
	}
}

func TestCloneDoesNotShareElements(t *testing.T) {
	source := []int{1, 2, 3}
	clone := Clone(source)
	clone[0] = 9
	if source[0] != 1 {
		t.Fatal("clone unexpectedly shares backing array")
	}
}

func TestCountAndSafeEqual(t *testing.T) {
	if Count([]string{"a", "b", "a"})["a"] != 2 {
		t.Fatal("unexpected count")
	}
	if SafeEqual([]int{1}, []int{1}) {
		t.Fatal("slices are not comparable")
	}
	if !SafeEqual(UserID(7), UserID(7)) {
		t.Fatal("defined comparable values must compare")
	}
}

func TestMergeCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	left, right := make(chan int, 1), make(chan int, 1)
	left <- 1
	right <- 2
	close(left)
	close(right)
	var sum int
	for value := range Merge(ctx, left, right) {
		sum += value
	}
	cancel()
	if sum != 3 {
		t.Fatalf("sum = %d", sum)
	}
}

func TestCancelCause(t *testing.T) {
	want := errors.New("invalid")
	ctx := CauseAfterValidation(t.Context(), want)
	if !errors.Is(context.Cause(ctx), want) || !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("err=%v cause=%v", ctx.Err(), context.Cause(ctx))
	}
}

func TestStringsMapsAndClosures(t *testing.T) {
	if got := ReverseRunes("Go🙂"); got != "🙂oG" {
		t.Fatalf("reverse = %q", got)
	}
	if got := EvenValues(map[string]int{"odd": 1, "even": 2}); len(got) != 1 || got["even"] != 2 {
		t.Fatalf("values = %v", got)
	}
	counters := Counters(3)
	if counters[0]() != 0 || counters[2]() != 2 {
		t.Fatal("loop values were captured incorrectly")
	}
}

func TestCloseAllAndTypedNil(t *testing.T) {
	want := errors.New("close failed")
	if err := CloseAll(closeFunc(func() error { return want })); !errors.Is(err, want) {
		t.Fatalf("close error = %v", err)
	}
	var concrete *typedError
	if IsNilError(concrete) {
		t.Fatal("an interface containing a typed nil is not nil")
	}
}
