package enstore

import (
	"bytes"
	"testing"
)

type sessionFile struct {
	*bytes.Reader
	name string
}

func (f sessionFile) Name() string { return f.name }

func sessionIndex(t *testing.T) (*Index, *LocalFileReadWriter, Crypter) {
	t.Helper()
	rw := &LocalFileReadWriter{BasePath: t.TempDir()}
	c, err := NewAESCrypter(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	ix, err := LoadIndex(rw, c, &Config{BlockSize: 4096, ChunkSize: 64, IndexFile: "index"})
	if err != nil {
		t.Fatal(err)
	}
	return ix, rw, c
}

func TestGetFileAddedInTheSameSession(t *testing.T) {
	ix, rw, c := sessionIndex(t)
	for name, data := range map[string][]byte{"small": []byte("tiny"), "spans-blocks": bytes.Repeat([]byte("0123456789"), 1000)} {
		if err := ix.AddFile(sessionFile{bytes.NewReader(data), name}, rw, rw, c); err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if err := ix.GetFile(name, &buf, rw, c); err != nil {
			t.Fatalf("get %s right after adding it: %v", name, err)
		}
		if !bytes.Equal(buf.Bytes(), data) {
			t.Fatalf("%s: content differs", name)
		}
	}
}

func TestDeleteFileAddedInTheSameSession(t *testing.T) {
	ix, rw, c := sessionIndex(t)
	if err := ix.AddFile(sessionFile{bytes.NewReader([]byte("short-lived")), "tmp"}, rw, rw, c); err != nil {
		t.Fatal(err)
	}
	if err := ix.DeleteFile("tmp", rw, rw, c, true); err != nil {
		t.Fatalf("delete right after adding: %v", err)
	}
	if len(ix.ListFiles()) != 0 {
		t.Fatalf("files after delete: %+v", ix.ListFiles())
	}
	if err := ix.GetFile("tmp", &bytes.Buffer{}, rw, c); err == nil {
		t.Fatal("deleted file still readable")
	}
}

func TestAddingADuplicateNameFails(t *testing.T) {
	ix, rw, c := sessionIndex(t)
	if err := ix.AddFile(sessionFile{bytes.NewReader([]byte("first")), "same"}, rw, rw, c); err != nil {
		t.Fatal(err)
	}
	if err := ix.AddFile(sessionFile{bytes.NewReader([]byte("second")), "same"}, rw, rw, c); err == nil {
		t.Fatal("adding a second file with the same name succeeded")
	}
	if files := ix.ListFiles(); len(files) != 1 {
		t.Fatalf("index changed by the failed add: %+v", files)
	}
	var buf bytes.Buffer
	if err := ix.GetFile("same", &buf, rw, c); err != nil || buf.String() != "first" {
		t.Fatalf("original file: %q, %v", buf.String(), err)
	}
}
