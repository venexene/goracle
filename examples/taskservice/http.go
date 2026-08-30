package taskservice

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
)

type createRequest struct {
	Title string `json:"title"`
}

func Handler(service *Service, logger *slog.Logger, maxBody int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "метод не поддерживается", http.StatusMethodNotAllowed)
			return
		}

		body := http.MaxBytesReader(w, r.Body, maxBody)
		defer body.Close()
		decoder := json.NewDecoder(body)
		decoder.DisallowUnknownFields()
		var input createRequest
		if err := decoder.Decode(&input); err != nil {
			http.Error(w, "некорректный запрос", http.StatusBadRequest)
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			http.Error(w, "ожидался один объект", http.StatusBadRequest)
			return
		}

		task, err := service.Create(r.Context(), input.Title)
		if err != nil {
			if errors.Is(err, ErrInvalidTitle) {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
			logger.ErrorContext(r.Context(), "не удалось создать задачу", "error", err)
			http.Error(w, "внутренняя ошибка", http.StatusInternalServerError)
			return
		}

		logger.InfoContext(r.Context(), "задача создана", "task_id", task.ID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(task)
	})
}
