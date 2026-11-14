package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// The CLI works with -key and no config.json in the working directory.
func TestCLIWithoutConfigFile(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "enstore")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	work := t.TempDir()
	if err := os.WriteFile(filepath.Join(work, "note.txt"), []byte("remember the milk"), 0o644); err != nil {
		t.Fatal(err)
	}
	add := exec.Command(bin, "-key", "k", "-add-file", "note.txt")
	add.Dir = work
	if out, err := add.CombinedOutput(); err != nil {
		t.Fatalf("add without a config file: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(work, "index")); err != nil {
		t.Fatalf("no index written to the working directory: %v", err)
	}
	get := exec.Command(bin, "-key", "k", "-get-file", "note.txt", "-o", "out.txt")
	get.Dir = work
	if out, err := get.CombinedOutput(); err != nil {
		t.Fatalf("get without a config file: %v\n%s", err, out)
	}
	if got, err := os.ReadFile(filepath.Join(work, "out.txt")); err != nil || string(got) != "remember the milk" {
		t.Fatalf("got %q, %v", got, err)
	}
}
