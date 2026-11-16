package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildCLI builds this package's binary once per test.
func buildCLI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "enstore")
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

// cli runs the binary in dir with the given config and returns its stdout.
func cli(t *testing.T, bin, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("enstore %s: %v\n%s%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// TestListAndStats adds three files to a store with 1 KB blocks, then checks:
//
//	-list   one line per file, sorted by name: "<name>\t<size in bytes>"
//	-stats  exactly four lines:
//	          files: <number of files>
//	          blocks: <number of block files in the store>
//	          stored bytes: <sum of the files' sizes>
//	          free bytes: <blocks × block size − stored bytes>
func TestListAndStats(t *testing.T) {
	bin := buildCLI(t)
	work := t.TempDir()
	store := filepath.Join(work, "store")
	if err := os.Mkdir(store, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg, _ := json.Marshal(map[string]any{"StoreDir": store, "BlockSize": 1024, "ChunkSize": 64, "IndexFile": "index"})
	if err := os.WriteFile(filepath.Join(work, "config.json"), cfg, 0o644); err != nil {
		t.Fatal(err)
	}
	sizes := map[string]int{"charlie.txt": 900, "alpha.txt": 100, "bravo.bin": 1500}
	for _, name := range []string{"charlie.txt", "alpha.txt", "bravo.bin"} {
		if err := os.WriteFile(filepath.Join(work, name), bytes.Repeat([]byte{'x'}, sizes[name]), 0o644); err != nil {
			t.Fatal(err)
		}
		cli(t, bin, work, "-config", "config.json", "-key", "secret", "-add-file", name)
	}

	want := "alpha.txt\t100\nbravo.bin\t1500\ncharlie.txt\t900\n"
	if got := cli(t, bin, work, "-config", "config.json", "-key", "secret", "-list"); got != want {
		t.Errorf("-list printed:\n%q\nwant:\n%q", got, want)
	}

	entries, err := os.ReadDir(store)
	if err != nil {
		t.Fatal(err)
	}
	blocks := 0
	for _, e := range entries {
		if e.Name() != "index" {
			blocks++
		}
	}
	stored := 100 + 1500 + 900
	want = fmt.Sprintf("files: 3\nblocks: %d\nstored bytes: %d\nfree bytes: %d\n", blocks, stored, blocks*1024-stored)
	if got := cli(t, bin, work, "-config", "config.json", "-key", "secret", "-stats"); got != want {
		t.Errorf("-stats printed:\n%q\nwant:\n%q", got, want)
	}
}
