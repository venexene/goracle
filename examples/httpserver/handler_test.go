package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGreeting(t *testing.T) {
	tests := []struct {
		name, body string
		want       int
	}{
		{"valid", `{"name":"Гофер"}`, http.StatusOK},
		{"unknown field", `{"name":"Гофер","admin":true}`, http.StatusBadRequest},
		{"too large", `{"name":"очень длинное имя"}`, http.StatusBadRequest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/greeting", strings.NewReader(test.body))
			res := httptest.NewRecorder()
			Greeting(24).ServeHTTP(res, req)
			if res.Code != test.want {
				t.Fatalf("status = %d, want %d", res.Code, test.want)
			}
		})
	}
}

func TestGreetingRejectsMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/greeting", nil)
	res := httptest.NewRecorder()
	Greeting(24).ServeHTTP(res, req)
	if res.Code != http.StatusMethodNotAllowed || res.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("status=%d allow=%q", res.Code, res.Header().Get("Allow"))
	}
}
