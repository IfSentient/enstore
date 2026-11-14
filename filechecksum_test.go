package enstore

import (
	"bytes"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

type fileSumFile struct {
	*bytes.Reader
	name string
}

func (f fileSumFile) Name() string { return f.name }

func fileSumEnv(t *testing.T, dir string, strict bool) (*LocalFileReadWriter, Crypter, *Config) {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	c, err := NewAESCrypter(key)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &Config{BlockSize: 1024, ChunkSize: 64, IndexFile: "index"}
	cfg.StrictChecksums = strict
	return &LocalFileReadWriter{BasePath: dir}, c, cfg
}

// fileSumStored adds name with data to a new store in dir and saves the index.
func fileSumStored(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	rw, c, cfg := fileSumEnv(t, dir, false)
	ix, err := LoadIndex(rw, c, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := ix.AddFile(fileSumFile{bytes.NewReader(data), name}, rw, rw, c); err != nil {
		t.Fatal(err)
	}
	if err := ix.Save(rw, c); err != nil {
		t.Fatal(err)
	}
}

func fileSumGet(t *testing.T, dir, name string, strict bool) ([]byte, error) {
	t.Helper()
	rw, c, cfg := fileSumEnv(t, dir, strict)
	ix, err := LoadIndex(rw, c, cfg)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	err = ix.GetFile(name, &buf, rw, c)
	return buf.Bytes(), err
}

func TestFileChecksumRoundTrip(t *testing.T) {
	data := make([]byte, 2600) // several blocks
	rand.New(rand.NewSource(3)).Read(data)
	dir := t.TempDir()
	fileSumStored(t, dir, "doc.bin", data)
	for _, strict := range []bool{false, true} {
		got, err := fileSumGet(t, dir, "doc.bin", strict)
		if err != nil || !bytes.Equal(got, data) {
			t.Fatalf("strict=%v: %d bytes, %v", strict, len(got), err)
		}
	}
}

// corruptFileData changes one byte of the file's plaintext inside its block,
// re-encrypting the block through the library, as a bug in block handling
// would; the block itself stays well-formed.
func corruptFileData(t *testing.T, dir, name string) {
	t.Helper()
	rw, c, cfg := fileSumEnv(t, dir, false)
	ix, err := LoadIndex(rw, c, cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range ix.ListFiles() {
		if f.Filename != name {
			continue
		}
		loc := f.Blocks[0]
		block, err := ReadBlock(loc.Block, c, rw)
		if err != nil {
			t.Fatal(err)
		}
		block.Bytes[loc.StartByte+(loc.EndByte-loc.StartByte)/2] ^= 0x01
		if err := WriteBlock(block, c, rw); err != nil {
			t.Fatal(err)
		}
		return
	}
	t.Fatalf("%s not in the index", name)
}

func TestFileChecksumMismatchWarnsAndReturnsTheFile(t *testing.T) {
	data := bytes.Repeat([]byte("checksum me "), 25)
	dir := t.TempDir()
	fileSumStored(t, dir, "note.txt", data)
	corruptFileData(t, dir, "note.txt")
	got, err := fileSumGet(t, dir, "note.txt", false)
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("default mode: %v, want ErrChecksumMismatch", err)
	}
	if len(got) != len(data) || bytes.Equal(got, data) {
		t.Fatalf("default mode should still return the (corrupted) file: %d bytes, unchanged=%v", len(got), bytes.Equal(got, data))
	}
}

func TestStrictModeReturnsNothingOnMismatch(t *testing.T) {
	data := bytes.Repeat([]byte("strict "), 300) // several blocks
	dir := t.TempDir()
	fileSumStored(t, dir, "big.txt", data)
	corruptFileData(t, dir, "big.txt")
	got, err := fileSumGet(t, dir, "big.txt", true)
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("strict mode: %v, want ErrChecksumMismatch", err)
	}
	if len(got) != 0 {
		t.Fatalf("strict mode wrote %d bytes of a corrupted file", len(got))
	}
}

// Files stored before checksums existed are returned without validation.
func TestFilesWithoutChecksumsStillRead(t *testing.T) {
	dir := t.TempDir()
	entries, err := os.ReadDir("testdata/legacy-store")
	if err != nil {
		t.Fatal(err)
	}
	for _, en := range entries {
		b, _ := os.ReadFile(filepath.Join("testdata/legacy-store", en.Name()))
		if err := os.WriteFile(filepath.Join(dir, en.Name()), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	b := make([]byte, 2500)
	rand.New(rand.NewSource(42)).Read(b)
	for _, strict := range []bool{false, true} {
		got, err := fileSumGet(t, dir, "b.bin", strict)
		if err != nil || !bytes.Equal(got, b) {
			t.Fatalf("strict=%v legacy b.bin: %d bytes, %v", strict, len(got), err)
		}
	}
}
