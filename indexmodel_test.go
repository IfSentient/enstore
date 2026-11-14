package enstore

import (
	"bytes"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"testing"
)

// modelStore is an in-memory block store for the model test.
type modelStore struct {
	mu sync.Mutex
	m  map[string][]byte
}

func (s *modelStore) Read(n string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.m[n]
	if !ok {
		return nil, fmt.Errorf("no block %s", n)
	}
	return append([]byte(nil), b...), nil
}

func (s *modelStore) Write(n string, b []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[n] = append([]byte(nil), b...)
	return nil
}

func (s *modelStore) Exists(n string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.m[n]
	return ok
}

func (s *modelStore) blocks() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for k := range s.m {
		if k != "index" {
			n++
		}
	}
	return n
}

type modelFile struct {
	*bytes.Reader
	name string
}

func (f modelFile) Name() string { return f.name }

func modelEnv() (*modelStore, Crypter, *Config) {
	c, _ := NewAESCrypter(make([]byte, 32))
	return &modelStore{m: map[string][]byte{}}, c, &Config{BlockSize: 1024, ChunkSize: 64, IndexFile: "index"}
}

// checkIndex compares the index with the model: every file reads back
// exactly, nothing else is listed, and no two files share bytes of a block.
func checkIndex(t *testing.T, where string, ix *Index, st *modelStore, c Crypter, model map[string][]byte) {
	t.Helper()
	listed := map[string]bool{}
	type span struct {
		start, end int64
		file       string
	}
	byBlock := map[string][]span{}
	for _, f := range ix.ListFiles() {
		if listed[f.Filename] {
			t.Fatalf("%s: %s listed twice", where, f.Filename)
		}
		listed[f.Filename] = true
		for _, l := range f.Blocks {
			byBlock[l.Block] = append(byBlock[l.Block], span{l.StartByte, l.EndByte, f.Filename})
		}
	}
	if len(listed) != len(model) {
		t.Fatalf("%s: index lists %d files, want %d", where, len(listed), len(model))
	}
	for name, want := range model {
		var buf bytes.Buffer
		if err := ix.GetFile(name, &buf, st, c); err != nil {
			t.Fatalf("%s: get %s: %v", where, name, err)
		}
		if !bytes.Equal(buf.Bytes(), want) {
			t.Fatalf("%s: %s reads back different content", where, name)
		}
	}
	for b, spans := range byBlock {
		sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
		for i := 1; i < len(spans); i++ {
			if spans[i].start < spans[i-1].end {
				t.Fatalf("%s: in block %s, %s [%d,%d) overlaps %s [%d,%d)", where, b[:8], spans[i].file, spans[i].start, spans[i].end,
					spans[i-1].file, spans[i-1].start, spans[i-1].end)
			}
		}
	}
}

// Seeded sequences of adds, deletes and lookups within one session (and,
// for some, with saves and reloads in between) must always agree with a
// plain map of names to contents.
func TestIndexAgreesWithAModel(t *testing.T) {
	for seed := int64(1); seed <= 40; seed++ {
		reloadEvery := []int{0, 7}[seed%2] // 0: never reload, all in one session
		r := rand.New(rand.NewSource(seed))
		st, c, cfg := modelEnv()
		ix, err := LoadIndex(st, c, cfg)
		if err != nil {
			t.Fatal(err)
		}
		model := map[string][]byte{}
		for op := 1; op <= 150; op++ {
			name := fmt.Sprintf("f%02d", r.Intn(15))
			where := fmt.Sprintf("seed %d op %d", seed, op)
			if _, exists := model[name]; exists && r.Intn(3) == 0 {
				if err := ix.DeleteFile(name, st, st, c, r.Intn(2) == 0); err != nil {
					t.Fatalf("%s: delete %s: %v", where, name, err)
				}
				delete(model, name)
			} else if !exists {
				data := make([]byte, 1+r.Intn(2800))
				r.Read(data)
				if err := ix.AddFile(modelFile{bytes.NewReader(data), name}, st, st, c); err != nil {
					t.Fatalf("%s: add %s: %v", where, name, err)
				}
				model[name] = data
			}
			if reloadEvery > 0 && op%reloadEvery == 0 {
				if err := ix.Save(st, c); err != nil {
					t.Fatal(err)
				}
				if ix, err = LoadIndex(st, c, cfg); err != nil {
					t.Fatal(err)
				}
			}
			checkIndex(t, where, ix, st, c, model)
		}
	}
}

// Space freed by a delete is used again: deleting a file and adding another
// of the same size, in the same session, creates no new blocks.
func TestFreedSpaceIsReused(t *testing.T) {
	st, c, cfg := modelEnv()
	ix, _ := LoadIndex(st, c, cfg)
	data := []byte(strings.Repeat("x", 2500))
	for i := 0; i < 3; i++ {
		if err := ix.AddFile(modelFile{bytes.NewReader(data), fmt.Sprintf("keep%d", i)}, st, st, c); err != nil {
			t.Fatal(err)
		}
	}
	if err := ix.AddFile(modelFile{bytes.NewReader(data), "temp"}, st, st, c); err != nil {
		t.Fatal(err)
	}
	before := st.blocks()
	for i := 0; i < 5; i++ {
		if err := ix.DeleteFile("temp", st, st, c, false); err != nil {
			t.Fatalf("round %d: delete: %v", i, err)
		}
		if err := ix.AddFile(modelFile{bytes.NewReader(data), "temp"}, st, st, c); err != nil {
			t.Fatalf("round %d: add: %v", i, err)
		}
	}
	if after := st.blocks(); after != before {
		t.Fatalf("blocks grew from %d to %d while the stored size stayed the same", before, after)
	}
	checkIndex(t, "after reuse", ix, st, c, map[string][]byte{"keep0": data, "keep1": data, "keep2": data, "temp": data})
}
