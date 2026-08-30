package language

import (
	"context"
	"errors"
	"math"
	"slices"
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

func TestReverseRunes(t *testing.T) {
	if got := ReverseRunes("Go🙂"); got != "🙂oG" {
		t.Fatalf("reverse = %q", got)
	}
}

func TestEvenValues(t *testing.T) {
	if got := EvenValues(map[string]int{"odd": 1, "even": 2}); len(got) != 1 || got["even"] != 2 {
		t.Fatalf("values = %v", got)
	}
}

func TestLoopClosures(t *testing.T) {
	counters := Counters(3)
	if counters[0]() != 0 || counters[2]() != 2 {
		t.Fatal("loop values were captured incorrectly")
	}
}

func TestStringBytesAndRunes(t *testing.T) {
	text := "Go🙂"
	if len(text) != 6 || len([]rune(text)) != 3 {
		t.Fatalf("bytes=%d runes=%d", len(text), len([]rune(text)))
	}
}

func TestAccumulatorClosure(t *testing.T) {
	left, right := Accumulator(), Accumulator()
	if left(2) != 2 || left(3) != 5 || right(4) != 4 {
		t.Fatal("closures must keep independent state")
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

func TestArrayCopyAndSliceSharing(t *testing.T) {
	array := [3]int{1, 2, 3}
	copyOfArray := array
	copyOfArray[0] = 9
	if array[0] != 1 {
		t.Fatal("array assignment must copy values")
	}
	slice := array[:]
	slice[0] = 7
	if array[0] != 7 {
		t.Fatal("slice must refer to its backing array")
	}
}

func TestMapLookupDistinguishesMissingKey(t *testing.T) {
	values := map[string]int{"zero": 0}
	if value, ok := values["zero"]; !ok || value != 0 {
		t.Fatalf("existing zero: value=%d ok=%v", value, ok)
	}
	if value, ok := values["missing"]; ok || value != 0 {
		t.Fatalf("missing: value=%d ok=%v", value, ok)
	}
}

func TestDeferUsesLIFOOrder(t *testing.T) {
	var order []int
	func() {
		defer func() { order = append(order, 1) }()
		defer func() { order = append(order, 2) }()
		defer func() { order = append(order, 3) }()
	}()
	if !slices.Equal(order, []int{3, 2, 1}) {
		t.Fatalf("order = %v", order)
	}
}

func TestJoinedErrorsKeepBothCauses(t *testing.T) {
	left := errors.New("left")
	right := errors.New("right")
	err := errors.Join(left, right)
	if !errors.Is(err, left) || !errors.Is(err, right) {
		t.Fatalf("joined error = %v", err)
	}
}

func TestClosedChannelReturnsZeroAndFalse(t *testing.T) {
	values := make(chan int, 1)
	values <- 7
	close(values)
	if value, ok := <-values; !ok || value != 7 {
		t.Fatalf("buffered value=%d ok=%v", value, ok)
	}
	if value, ok := <-values; ok || value != 0 {
		t.Fatalf("closed channel value=%d ok=%v", value, ok)
	}
}

func TestGenericClonePreservesNamedSlice(t *testing.T) {
	type numbers []int
	source := numbers{1, 2, 3}
	clone := Clone(source)
	if !slices.Equal(clone, source) {
		t.Fatalf("clone = %v", clone)
	}
}
