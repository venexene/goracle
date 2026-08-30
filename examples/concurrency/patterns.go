package concurrency

import "context"

func Map[T, R any](ctx context.Context, input <-chan T, transform func(T) R) <-chan R {
	output := make(chan R)
	go func() {
		defer close(output)
		for value := range input {
			select {
			case output <- transform(value):
			case <-ctx.Done():
				return
			}
		}
	}()
	return output
}

func RunLimited[T any](ctx context.Context, limit int, jobs []T, work func(context.Context, T) error) error {
	if limit < 1 {
		limit = 1
	}
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	semaphore := make(chan struct{}, limit)
	done := make(chan struct{}, len(jobs))
	for _, job := range jobs {
		select {
		case semaphore <- struct{}{}:
		case <-ctx.Done():
			return context.Cause(ctx)
		}
		go func() {
			defer func() { <-semaphore; done <- struct{}{} }()
			if err := work(ctx, job); err != nil {
				cancel(err)
			}
		}()
	}
	for range jobs {
		<-done
	}
	return context.Cause(ctx)
}
