package database

import "testing"

func TestUsersQuery(t *testing.T) {
	got, err := UsersQuery("name", "desc")
	if err != nil || got != "SELECT id, display_name FROM users ORDER BY display_name DESC" {
		t.Fatalf("query=%q err=%v", got, err)
	}
	if _, err := UsersQuery("name; DROP TABLE users", "asc"); err == nil {
		t.Fatal("unknown identifier must be rejected")
	}
}
