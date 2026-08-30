package logging

import (
	"bytes"
	"encoding/json"
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
