package enstore

import (
	"bytes"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type blkSumFile struct {
	*bytes.Reader
	name string
}

func (f blkSumFile) Name() string { return f.name }

type blkSumEnv struct {
	dir string
	rw  *LocalFileReadWriter
	c   Crypter
	cfg *Config
}

func newBlkSumEnv(t *testing.T, dir string) blkSumEnv {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	c, err := NewAESCrypter(key)
	if err != nil {
		t.Fatal(err)
	}
	return blkSumEnv{dir: dir, rw: &LocalFileReadWriter{BasePath: dir}, c: c, cfg: &Config{BlockSize: 1024, ChunkSize: 64, IndexFile: "index"}}
}

func (e blkSumEnv) load(t *testing.T) *Index {
	t.Helper()
	ix, err := LoadIndex(e.rw, e.c, e.cfg)
	if err != nil {
		t.Fatal(err)
	}
	return ix
}

func (e blkSumEnv) get(ix *Index, name string) ([]byte, error) {
	var buf bytes.Buffer
	err := ix.GetFile(name, &buf, e.rw, e.c)
	return buf.Bytes(), err
}

// store adds the files and saves the index; reading happens from a reloaded index.
func (e blkSumEnv) store(t *testing.T, files map[string][]byte) {
	t.Helper()
	ix := e.load(t)
	for name, data := range files {
		if err := ix.AddFile(blkSumFile{bytes.NewReader(data), name}, e.rw, e.rw, e.c); err != nil {
			t.Fatal(err)
		}
	}
	if err := ix.Save(e.rw, e.c); err != nil {
		t.Fatal(err)
	}
}

func blkSumFiles() map[string][]byte {
	big := make([]byte, 3000) // spans several 1 KB blocks
	rand.New(rand.NewSource(1)).Read(big)
	return map[string][]byte{"small.txt": []byte("a small file"), "big.bin": big}
}

func TestBlockChecksumsLetIntactFilesRead(t *testing.T) {
	e := newBlkSumEnv(t, t.TempDir())
	files := blkSumFiles()
	e.store(t, files)
	for round := 0; round < 2; round++ { // and again after another save and reload
		ix := e.load(t)
		for name, want := range files {
			got, err := e.get(ix, name)
			if err != nil || !bytes.Equal(got, want) {
				t.Fatalf("round %d: %s: %d bytes, %v", round, name, len(got), err)
			}
		}
		if err := ix.Save(e.rw, e.c); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCorruptedBlockIsReported(t *testing.T) {
	e := newBlkSumEnv(t, t.TempDir())
	e.store(t, blkSumFiles())
	ix := e.load(t)
	var block string
	for _, f := range ix.ListFiles() {
		if f.Filename == "big.bin" {
			block = f.Blocks[len(f.Blocks)/2].Block
		}
	}
	if block == "" {
		t.Fatal("big.bin has no blocks")
	}
	path := filepath.Join(e.dir, block)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)/2] ^= 0xff // one flipped byte, as disk rot or a bad transfer would leave
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	for round, ix := range []*Index{ix, e.load(t)} {
		_, err := e.get(ix, "big.bin")
		if !errors.Is(err, ErrCorruptedBlock) {
			t.Fatalf("round %d: reading a file in a corrupted block: %v, want ErrCorruptedBlock", round, err)
		}
		if !strings.Contains(err.Error(), "corrupted block: "+block) {
			t.Fatalf("round %d: error %q doesn't name the block %s", round, err, block)
		}
	}
	if got, err := e.get(e.load(t), "small.txt"); err != nil || string(got) != "a small file" {
		// small.txt is never in big.bin's middle block, whichever was added first
		t.Fatalf("an intact file: %q, %v", got, err)
	}
}

// A store written before checksums existed keeps working.
func TestStoreWithoutChecksumsStillReads(t *testing.T) {
	dir := t.TempDir()
	entries, err := os.ReadDir("testdata/legacy-store")
	if err != nil {
		t.Fatal(err)
	}
	for _, en := range entries {
		b, err := os.ReadFile(filepath.Join("testdata/legacy-store", en.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, en.Name()), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	e := newBlkSumEnv(t, dir)
	ix := e.load(t)
	want := map[string][]byte{"a.txt": bytes.Repeat([]byte("legacy-a "), 40)}
	b := make([]byte, 2500)
	rand.New(rand.NewSource(42)).Read(b)
	want["b.bin"] = b
	for name, w := range want {
		got, err := e.get(ix, name)
		if err != nil || !bytes.Equal(got, w) {
			t.Fatalf("legacy %s: %d bytes, %v", name, len(got), err)
		}
	}
}
