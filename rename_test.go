package enstore

import (
	"bytes"
	"testing"
)

type renameTestFile struct {
	*bytes.Reader
	name string
}

func (f renameTestFile) Name() string { return f.name }

// An index with two saved files, loaded fresh.
func renameIndex(t *testing.T) (*Index, *LocalFileReadWriter, Crypter, *Config) {
	t.Helper()
	rw := &LocalFileReadWriter{BasePath: t.TempDir()}
	c, err := NewAESCrypter(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	cfg := &Config{BlockSize: 4096, ChunkSize: 64, IndexFile: "index"}
	ix, _ := LoadIndex(rw, c, cfg)
	for name, data := range map[string]string{"a.txt": "alpha", "b.txt": "bravo"} {
		if err := ix.AddFile(renameTestFile{bytes.NewReader([]byte(data)), name}, rw, rw, c); err != nil {
			t.Fatal(err)
		}
	}
	if err := ix.Save(rw, c); err != nil {
		t.Fatal(err)
	}
	ix, err = LoadIndex(rw, c, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return ix, rw, c, cfg
}

func renameGet(t *testing.T, ix *Index, rw *LocalFileReadWriter, c Crypter, name string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	err := ix.GetFile(name, &buf, rw, c)
	return buf.String(), err
}

func TestRenameFile(t *testing.T) {
	ix, rw, c, cfg := renameIndex(t)
	if err := ix.RenameFile("a.txt", "renamed.txt"); err != nil {
		t.Fatal(err)
	}
	if got, err := renameGet(t, ix, rw, c, "renamed.txt"); err != nil || got != "alpha" {
		t.Fatalf("new name: %q, %v", got, err)
	}
	if _, err := renameGet(t, ix, rw, c, "a.txt"); err == nil {
		t.Fatal("old name still readable")
	}
	if err := ix.Save(rw, c); err != nil {
		t.Fatal(err)
	}
	again, _ := LoadIndex(rw, c, cfg)
	if got, err := renameGet(t, again, rw, c, "renamed.txt"); err != nil || got != "alpha" {
		t.Fatalf("after reload: %q, %v", got, err)
	}
	names := map[string]bool{}
	for _, f := range again.ListFiles() {
		names[f.Filename] = true
	}
	if len(names) != 2 || !names["renamed.txt"] || !names["b.txt"] {
		t.Fatalf("files after reload: %v", names)
	}
}

func TestRenameFileErrorsLeaveTheIndexUnchanged(t *testing.T) {
	ix, rw, c, _ := renameIndex(t)
	if err := ix.RenameFile("missing.txt", "x.txt"); err == nil {
		t.Fatal("renaming a missing file succeeded")
	}
	if err := ix.RenameFile("a.txt", "b.txt"); err == nil {
		t.Fatal("renaming onto an existing name succeeded")
	}
	for name, want := range map[string]string{"a.txt": "alpha", "b.txt": "bravo"} {
		if got, err := renameGet(t, ix, rw, c, name); err != nil || got != want {
			t.Fatalf("%s after failed renames: %q, %v", name, got, err)
		}
	}
	if len(ix.ListFiles()) != 2 {
		t.Fatalf("files %+v", ix.ListFiles())
	}
}
