package database

import (
	"errors"
	"testing"
)

func TestUsersQuery(t *testing.T) {
	got, err := UsersQuery("name", "desc")
	if err != nil || got != "SELECT id, display_name FROM users ORDER BY display_name DESC" {
		t.Fatalf("query=%q err=%v", got, err)
	}
	if _, err := UsersQuery("name; DROP TABLE users", "asc"); err == nil {
		t.Fatal("unknown identifier must be rejected")
	}
}

func TestEscapeLike(t *testing.T) {
	if got := EscapeLike(`50%_\`); got != `50\%\_\\` {
		t.Fatalf("escaped = %q", got)
	}
}

func TestRetryRepeatsWholeOperation(t *testing.T) {
	retryable := errors.New("serialization")
	var calls int
	err := Retry(3, func() error {
		calls++
		if calls < 3 {
			return retryable
		}
		return nil
	}, func(err error) bool { return errors.Is(err, retryable) })
	if err != nil || calls != 3 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}
