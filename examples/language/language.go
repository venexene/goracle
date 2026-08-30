package language

import (
	"context"
	"errors"
	"io"
	"math"
	"reflect"
)

type UserID int64

func AddUint64(a, b uint64) (uint64, error) {
	if math.MaxUint64-a < b {
		return 0, errors.New("переполнение")
	}
	return a + b, nil
}

func Clone[T any](values []T) []T { return append([]T(nil), values...) }

func Count[T comparable](values []T) map[T]int {
	counts := make(map[T]int, len(values))
	for _, value := range values {
		counts[value]++
	}
	return counts
}

func SafeEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == b
	}
	ta, tb := reflect.TypeOf(a), reflect.TypeOf(b)
	return ta == tb && ta.Comparable() && a == b
}

func Merge[T any](ctx context.Context, inputs ...<-chan T) <-chan T {
	out := make(chan T)
	done := make(chan struct{}, len(inputs))
	for _, input := range inputs {
		go func() {
			defer func() { done <- struct{}{} }()
			for value := range input {
				select {
				case out <- value:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() {
		for range inputs {
			<-done
		}
		close(out)
	}()
	return out
}

func CauseAfterValidation(parent context.Context, validationErr error) context.Context {
	if validationErr == nil {
		return parent
	}
	ctx, cancel := context.WithCancelCause(parent)
	cancel(validationErr)
	return ctx
}

func ReverseRunes(text string) string {
	runes := []rune(text)
	for left, right := 0, len(runes)-1; left < right; left, right = left+1, right-1 {
		runes[left], runes[right] = runes[right], runes[left]
	}
	return string(runes)
}

func EvenValues(values map[string]int) map[string]int {
	result := make(map[string]int)
	for key, value := range values {
		if value%2 == 0 {
			result[key] = value
		}
	}
	return result
}

func Counters(n int) []func() int {
	result := make([]func() int, n)
	for i := range n {
		value := i
		result[i] = func() int { return value }
	}
	return result
}

func Accumulator() func(int) int {
	var total int
	return func(delta int) int {
		total += delta
		return total
	}
}

func CloseAll(closers ...io.Closer) (err error) {
	for i := len(closers) - 1; i >= 0; i-- {
		err = errors.Join(err, closers[i].Close())
	}
	return err
}

func IsNilError(err error) bool { return err == nil }
