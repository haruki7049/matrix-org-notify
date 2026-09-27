package main

import (
	"bytes"
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestRunNoMessage(t *testing.T) {
	err := run(nil, strings.NewReader(""), &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected an error when no message is provided")
	}
}

func TestRunHelp(t *testing.T) {
	for _, name := range []string{"-h", "--help"} {
		var out bytes.Buffer
		err := run([]string{name}, strings.NewReader(""), &out)
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("err for %q = %v, want flag.ErrHelp", name, err)
		}
		if !strings.Contains(out.String(), "Usage:") {
			t.Errorf("output for %q = %q, want it to contain usage text", name, out.String())
		}
	}
}
