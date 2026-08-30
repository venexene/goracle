package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type request struct {
	Name string `json:"name"`
}

func Greeting(maxBody int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "метод не поддерживается", http.StatusMethodNotAllowed)
			return
		}
		body := http.MaxBytesReader(w, r.Body, maxBody)
		defer body.Close()
		var input request
		decoder := json.NewDecoder(body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil || input.Name == "" {
			http.Error(w, "некорректный запрос", http.StatusBadRequest)
			return
		}
		if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
			http.Error(w, "ожидался один объект", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Привет, " + input.Name})
	})
}
