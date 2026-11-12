package cli

import (
	"os"
	"testing"
)

func TestClosedStdinIsNotATerminal(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	original := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() { os.Stdin = original })
	if isStdinTerminal() {
		t.Fatal("closed stdin reported as a terminal")
	}
}
