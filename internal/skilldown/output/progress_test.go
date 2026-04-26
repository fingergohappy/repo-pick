package output

import (
	"bytes"
	"errors"
	"testing"
)

func TestProgressPrinterWritesProgressLines(t *testing.T) {
	var out bytes.Buffer
	printer := NewProgressPrinter(&out)

	printer.Start("clone repo")
	printer.Success("clone repo")
	printer.Failure("install skill", errors.New("target exists"))

	want := "[..] clone repo\n[ok] clone repo\n[!!] install skill: target exists\n"
	if got := out.String(); got != want {
		t.Fatalf("progress output = %q, want %q", got, want)
	}
}

func TestProgressPrinterIgnoresNilWriter(t *testing.T) {
	printer := NewProgressPrinter(nil)

	printer.Start("clone repo")
	printer.Success("clone repo")
	printer.Failure("clone repo", errors.New("failed"))
}
