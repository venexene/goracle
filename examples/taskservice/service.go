package taskservice

import (
	"context"
	"errors"
	"strings"
	"sync"
)

var ErrInvalidTitle = errors.New("название задачи обязательно")

type Task struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

type Store interface {
	Create(context.Context, string) (Task, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service { return &Service{store: store} }

func (s *Service) Create(ctx context.Context, title string) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, ErrInvalidTitle
	}
	return s.store.Create(ctx, title)
}

type MemoryStore struct {
	mu     sync.Mutex
	nextID int64
	tasks  []Task
}

func (s *MemoryStore) Create(ctx context.Context, title string) (Task, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	task := Task{ID: s.nextID, Title: title}
	s.tasks = append(s.tasks, task)
	return task, nil
}

func (s *MemoryStore) Tasks() []Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Task(nil), s.tasks...)
}
