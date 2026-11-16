package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A missing file is reported plainly, not with a panic: the message
// "file not found: <name>" on stderr and exit status 1.
func TestCLIReportsAMissingFile(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "enstore")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	for _, flag := range []string{"-get-file", "-delete-file"} {
		dir := t.TempDir()
		// An explicit config, so the store is this directory.
		if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"StoreDir": "."}`), 0o644); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(bin, "-key", "secret", flag, "missing.txt")
		cmd.Dir = dir
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		err := cmd.Run()
		var exit *exec.ExitError
		if !errorsAs(err, &exit) || exit.ExitCode() != 1 {
			t.Errorf("%s missing.txt: %v, want exit status 1", flag, err)
		}
		if got := stderr.String(); !strings.Contains(got, "file not found: missing.txt") || strings.Contains(got, "panic") {
			t.Errorf("%s missing.txt printed %q", flag, got)
		}
	}
}

func errorsAs(err error, target **exec.ExitError) bool {
	e, ok := err.(*exec.ExitError)
	if ok {
		*target = e
	}
	return ok
}
