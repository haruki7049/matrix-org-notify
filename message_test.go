package main

import (
	"strings"
	"testing"
)

func TestReadMessage(t *testing.T) {
	t.Run("from args", func(t *testing.T) {
		got, err := readMessage([]string{"hello", "world"}, strings.NewReader(""))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "hello world" {
			t.Fatalf("got %q, want %q", got, "hello world")
		}
	})

	t.Run("from stdin", func(t *testing.T) {
		got, err := readMessage(nil, strings.NewReader("piped message\n"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "piped message" {
			t.Fatalf("got %q, want %q", got, "piped message")
		}
	})
}
