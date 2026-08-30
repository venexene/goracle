package network

import (
	"bytes"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	var stream bytes.Buffer
	if err := WriteFrame(&stream, []byte("данные"), 64); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFrame(&stream, 64)
	if err != nil || string(got) != "данные" {
		t.Fatalf("got %q, err %v", got, err)
	}
}

func TestReadFrameChecksLimitBeforeAllocation(t *testing.T) {
	stream := bytes.NewBuffer([]byte{0, 0, 1, 0})
	if _, err := ReadFrame(stream, 16); err == nil {
		t.Fatal("oversized frame must be rejected")
	}
}
