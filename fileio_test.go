package enstore

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// LocalFileReadWriter must keep handling BasePath with and without a trailing
// slash, and an empty BasePath (the working directory), for all three methods.
func TestLocalFileReadWriterBasePaths(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if err := os.Mkdir(filepath.Join(root, "store"), 0o755); err != nil {
		t.Fatal(err)
	}
	for i, base := range []string{"", "store", "store/", filepath.Join(root, "store"), filepath.Join(root, "store") + "/"} {
		rw := &LocalFileReadWriter{BasePath: base}
		name := fmt.Sprintf("blk-%d", i)
		if rw.Exists(name) {
			t.Fatalf("BasePath %q: %s exists before it was written", base, name)
		}
		if err := rw.Write(name, []byte("data:"+base)); err != nil {
			t.Fatalf("BasePath %q: write: %v", base, err)
		}
		if !rw.Exists(name) {
			t.Fatalf("BasePath %q: %s doesn't exist after writing", base, name)
		}
		got, err := rw.Read(name)
		if err != nil || string(got) != "data:"+base {
			t.Fatalf("BasePath %q: read %q, %v", base, got, err)
		}
		dir := root
		if base != "" {
			dir = filepath.Join(root, "store")
		}
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("BasePath %q: file not where expected: %v", base, err)
		}
	}
}
