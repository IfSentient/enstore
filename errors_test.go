package enstore

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// errStore is an in-memory BlockReader, BlockWriter and IndexReader whose
// reads can be made to fail.
type errStore struct {
	data    map[string][]byte
	readErr error
}

func newErrStore() *errStore { return &errStore{data: map[string][]byte{}} }

func (s *errStore) Read(name string) ([]byte, error) {
	if s.readErr != nil {
		return nil, s.readErr
	}
	b, ok := s.data[name]
	if !ok {
		return nil, errors.New("no such block " + name)
	}
	return append([]byte(nil), b...), nil
}

func (s *errStore) Write(name string, b []byte) error {
	s.data[name] = append([]byte(nil), b...)
	return nil
}

func (s *errStore) Exists(name string) bool { _, ok := s.data[name]; return ok }

type memFile struct {
	*bytes.Reader
	name string
}

func (f memFile) Name() string { return f.name }

func newMemFile(name, content string) memFile {
	return memFile{bytes.NewReader([]byte(content)), name}
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func testCrypter(t *testing.T) Crypter {
	t.Helper()
	c, err := NewAESCrypter([]byte("0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// storeWith saves an index holding the given files and loads it back.
func storeWith(t *testing.T, s *errStore, c Crypter, files ...memFile) *Index {
	t.Helper()
	cfg := NewDefaultConfig()
	cfg.BlockSize = 4096
	ix := NewIndex(cfg)
	for _, f := range files {
		if err := ix.AddFile(f, s, s, c); err != nil {
			t.Fatal(err)
		}
	}
	if err := ix.Save(s, c); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadIndex(s, c, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func TestMissingFileIsErrFileNotFound(t *testing.T) {
	s, c := newErrStore(), testCrypter(t)
	ix := storeWith(t, s, c, newMemFile("present.txt", "hello"))
	if err := ix.GetFile("absent.txt", &bytes.Buffer{}, s, c); !errors.Is(err, ErrFileNotFound) {
		t.Errorf("GetFile of a missing file: %v, want ErrFileNotFound", err)
	}
	if err := ix.DeleteFile("absent.txt", s, s, c, true); !errors.Is(err, ErrFileNotFound) {
		t.Errorf("DeleteFile of a missing file: %v, want ErrFileNotFound", err)
	}
	if err := ix.GetFile("absent.txt", &bytes.Buffer{}, s, c); err == nil || !strings.Contains(err.Error(), "absent.txt") {
		t.Errorf("the error should name the file: %v", err)
	}
}

func TestAddingAStoredNameIsErrFileExists(t *testing.T) {
	s, c := newErrStore(), testCrypter(t)
	cfg := NewDefaultConfig()
	cfg.BlockSize = 4096
	ix := NewIndex(cfg)
	if err := ix.AddFile(newMemFile("a.txt", "first"), s, s, c); err != nil {
		t.Fatal(err)
	}
	if err := ix.AddFile(newMemFile("a.txt", "second"), s, s, c); !errors.Is(err, ErrFileExists) {
		t.Errorf("adding a.txt twice in one session: %v, want ErrFileExists", err)
	}
	loaded := storeWith(t, s, c, newMemFile("b.txt", "bee"))
	if err := loaded.AddFile(newMemFile("b.txt", "again"), s, s, c); !errors.Is(err, ErrFileExists) {
		t.Errorf("adding b.txt again after a reload: %v, want ErrFileExists", err)
	}
	if n := len(loaded.ListFiles()); n != 1 {
		t.Errorf("a refused add changed the index: %d files", n)
	}
}

func TestGetFileReturnsTheWritersError(t *testing.T) {
	s, c := newErrStore(), testCrypter(t)
	ix := storeWith(t, s, c, newMemFile("report.txt", "some contents"))
	boom := errors.New("disk full")
	if err := ix.GetFile("report.txt", failingWriter{boom}, s, c); !errors.Is(err, boom) {
		t.Errorf("GetFile with a failing writer: %v, want it to wrap %v", err, boom)
	}
}

func TestBlockReadFailuresKeepTheCauseAndNameTheFile(t *testing.T) {
	s, c := newErrStore(), testCrypter(t)
	ix := storeWith(t, s, c, newMemFile("report.txt", "some contents"))
	broken := errors.New("network unreachable")
	s.readErr = broken
	err := ix.GetFile("report.txt", &bytes.Buffer{}, s, c)
	if !errors.Is(err, broken) {
		t.Errorf("GetFile with a failing reader: %v, want it to wrap %v", err, broken)
	}
	if err == nil || !strings.Contains(err.Error(), "report.txt") {
		t.Errorf("the error should name the file: %v", err)
	}
}
