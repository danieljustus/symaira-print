package main

import (
	"io"
	"os"
	"testing"
)

// captureStdout redirects os.Stdout for the duration of fn and returns
// everything written to it. The commands under test print via fmt.Println
// directly to os.Stdout rather than cmd.OutOrStdout(), so cobra's own output
// buffering does not capture it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	return string(out)
}
