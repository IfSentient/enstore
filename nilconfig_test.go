package enstore

import (
	"bytes"
	"testing"
)

type nilCfgFile struct {
	*bytes.Reader
	name string
}

func (f nilCfgFile) Name() string { return f.name }

// With no config, LoadIndex uses the defaults (NewDefaultConfig) rather than
// panicking, and the index it returns works.
func TestLoadIndexWithoutConfig(t *testing.T) {
	dir := t.TempDir()
	rw := &LocalFileReadWriter{BasePath: dir}
	c, err := NewAESCrypter(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	ix, err := LoadIndex(rw, c, nil)
	if err != nil || ix == nil {
		t.Fatalf("LoadIndex with a nil config: %v, %v", ix, err)
	}
	data := []byte("hello, defaults")
	if err := ix.AddFile(nilCfgFile{bytes.NewReader(data), "a.txt"}, rw, rw, c); err != nil {
		t.Fatal(err)
	}
	if err := ix.Save(rw, c); err != nil {
		t.Fatal(err)
	}
	if !rw.Exists(DefaultIndexfile) {
		t.Fatalf("index not saved as %q", DefaultIndexfile)
	}
	again, err := LoadIndex(rw, c, nil)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := again.GetFile("a.txt", &buf, rw, c); err != nil || buf.String() != string(data) {
		t.Fatalf("read back %q, %v", buf.String(), err)
	}
	files := again.ListFiles()
	if len(files) != 1 || len(files[0].Blocks) != 1 {
		t.Fatalf("files %+v", files)
	}
}
