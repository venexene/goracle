package rpccontract

import (
	"context"
	"errors"
)

type Code string

const (
	CodeInvalidArgument Code = "INVALID_ARGUMENT"
	CodeNotFound        Code = "NOT_FOUND"
	CodeInternal        Code = "INTERNAL"
)

var (
	ErrInvalid = errors.New("некорректные данные")
	ErrMissing = errors.New("объект не найден")
)

func CodeForError(err error) Code {
	switch {
	case errors.Is(err, ErrInvalid):
		return CodeInvalidArgument
	case errors.Is(err, ErrMissing):
		return CodeNotFound
	default:
		return CodeInternal
	}
}

type Handler func(context.Context, string) (string, error)
type Interceptor func(context.Context, string, Handler) (string, error)

func Chain(handler Handler, interceptors ...Interceptor) Handler {
	for index := len(interceptors) - 1; index >= 0; index-- {
		next := handler
		interceptor := interceptors[index]
		handler = func(ctx context.Context, request string) (string, error) {
			return interceptor(ctx, request, next)
		}
	}
	return handler
}
