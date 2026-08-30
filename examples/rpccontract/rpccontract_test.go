package rpccontract

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestCodeForErrorDoesNotExposeInternalFailure(t *testing.T) {
	if got := CodeForError(ErrInvalid); got != CodeInvalidArgument {
		t.Fatalf("invalid code = %s", got)
	}
	if got := CodeForError(errors.New("password=secret")); got != CodeInternal {
		t.Fatalf("internal code = %s", got)
	}
}

func TestInterceptorChainOrder(t *testing.T) {
	var order []string
	first := func(ctx context.Context, request string, next Handler) (string, error) {
		order = append(order, "first:before")
		response, err := next(ctx, request)
		order = append(order, "first:after")
		return response, err
	}
	second := func(ctx context.Context, request string, next Handler) (string, error) {
		order = append(order, "second:before")
		response, err := next(ctx, request)
		order = append(order, "second:after")
		return response, err
	}
	handler := Chain(func(context.Context, string) (string, error) {
		order = append(order, "handler")
		return "ok", nil
	}, first, second)
	if _, err := handler(t.Context(), "request"); err != nil {
		t.Fatal(err)
	}
	want := []string{"first:before", "second:before", "handler", "second:after", "first:after"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order=%v want=%v", order, want)
	}
}
