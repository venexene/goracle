package taskservice

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type blockedListener struct {
	closed chan struct{}
	once   sync.Once
}

func newBlockedListener() *blockedListener {
	return &blockedListener{closed: make(chan struct{})}
}

func (l *blockedListener) Accept() (net.Conn, error) {
	<-l.closed
	return nil, net.ErrClosed
}

func (l *blockedListener) Close() error {
	l.once.Do(func() { close(l.closed) })
	return nil
}

func (l *blockedListener) Addr() net.Addr { return testAddress("test") }

type testAddress string

func (a testAddress) Network() string { return string(a) }
func (a testAddress) String() string  { return string(a) }

func TestCreateTask(t *testing.T) {
	store := &MemoryStore{}
	logs := &bytes.Buffer{}
	handler := Handler(NewService(store), slog.New(slog.NewJSONHandler(logs, nil)), 128)
	request := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(`{"title":"изучить контекст"}`))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var task Task
	if err := json.NewDecoder(response.Body).Decode(&task); err != nil {
		t.Fatal(err)
	}
	if task.ID != 1 || task.Title != "изучить контекст" || len(store.Tasks()) != 1 {
		t.Fatalf("task = %+v, stored = %+v", task, store.Tasks())
	}
	if !bytes.Contains(logs.Bytes(), []byte(`"task_id":1`)) {
		t.Fatalf("log does not contain task id: %s", logs.String())
	}
}

func TestHandlerRejectsInvalidInput(t *testing.T) {
	handler := Handler(NewService(&MemoryStore{}), slog.New(slog.NewTextHandler(io.Discard, nil)), 16)
	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "empty title", body: `{"title":""}`, want: http.StatusUnprocessableEntity},
		{name: "unknown field", body: `{"title":"x","extra":true}`, want: http.StatusBadRequest},
		{name: "too large", body: `{"title":"a very long title"}`, want: http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewBufferString(test.body))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.want {
				t.Fatalf("status = %d, want %d", response.Code, test.want)
			}
		})
	}
}

func TestHandlerRejectsUnsupportedMethod(t *testing.T) {
	handler := Handler(NewService(&MemoryStore{}), slog.New(slog.NewTextHandler(io.Discard, nil)), 16)
	request := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
	if allow := response.Header().Get("Allow"); allow != http.MethodPost {
		t.Fatalf("Allow = %q, want %q", allow, http.MethodPost)
	}
}

func TestServeStopsAfterCancellation(t *testing.T) {
	listener := newBlockedListener()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, listener, http.NewServeMux()) }()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop")
	}
}
