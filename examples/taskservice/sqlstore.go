package taskservice

import (
	"context"
	"database/sql"
	"fmt"
)

const Schema = `CREATE TABLE IF NOT EXISTS tasks (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title TEXT NOT NULL
)`

type SQLStore struct {
	DB *sql.DB
}

func (s *SQLStore) Create(ctx context.Context, title string) (Task, error) {
	if s == nil || s.DB == nil {
		return Task{}, fmt.Errorf("taskservice: база данных не настроена")
	}

	const query = `INSERT INTO tasks (title) VALUES ($1) RETURNING id, title`
	var task Task
	if err := s.DB.QueryRowContext(ctx, query, title).Scan(&task.ID, &task.Title); err != nil {
		return Task{}, fmt.Errorf("create task: %w", err)
	}
	return task, nil
}
