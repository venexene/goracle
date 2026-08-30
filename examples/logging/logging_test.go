package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestStableEventSchema(t *testing.T) {
	var output bytes.Buffer
	RequestCompleted(t.Context(), NewJSONLogger(&output, "users", "test"), "/users/{id}", 200)
	var event map[string]any
	if err := json.Unmarshal(output.Bytes(), &event); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"service", "environment", "event", "route", "status", "level", "msg"} {
		if _, ok := event[key]; !ok {
			t.Errorf("missing key %q", key)
		}
	}
	if event["route"] != "/users/{id}" {
		t.Fatalf("raw route: %v", event["route"])
	}
}

func TestSensitiveAttributesAreRedacted(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{ReplaceAttr: Redact}))
	logger.Info("вход", "token", "secret", "user_id", 42)
	if bytes.Contains(output.Bytes(), []byte("secret")) || !bytes.Contains(output.Bytes(), []byte("[скрыто]")) {
		t.Fatalf("output = %s", output.String())
	}
}
