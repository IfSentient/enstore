package enstore

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
)

type memTestFile struct {
	*bytes.Reader
	name string
}

func (f memTestFile) Name() string { return f.name }

func TestMemoryStoreBacksAnIndex(t *testing.T) {
	store := NewMemoryStore()
	c, err := NewAESCrypter(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	cfg := &Config{BlockSize: 2048, ChunkSize: 64, IndexFile: "index"}
	ix, err := LoadIndex(store, c, cfg)
	if err != nil {
		t.Fatal(err)
	}
	data := bytes.Repeat([]byte("enstore "), 1000) // spans several blocks
	if err := ix.AddFile(memTestFile{bytes.NewReader(data), "big"}, store, store, c); err != nil {
		t.Fatal(err)
	}
	if err := ix.Save(store, c); err != nil {
		t.Fatal(err)
	}
	if !store.Exists("index") {
		t.Fatal("index not in the store")
	}
	again, err := LoadIndex(store, c, cfg)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := again.GetFile("big", &buf, store, c); err != nil || !bytes.Equal(buf.Bytes(), data) {
		t.Fatalf("read back %d bytes, %v", buf.Len(), err)
	}
}

func TestMemoryStoreBasics(t *testing.T) {
	store := NewMemoryStore()
	if store.Exists("x") {
		t.Fatal("empty store has x")
	}
	if _, err := store.Read("x"); err == nil {
		t.Fatal("reading a missing block succeeded")
	}
	in := []byte("abc")
	if err := store.Write("x", in); err != nil {
		t.Fatal(err)
	}
	in[0] = 'z' // the store keeps its own copy
	out, err := store.Read("x")
	if err != nil || string(out) != "abc" {
		t.Fatalf("read %q, %v", out, err)
	}
	out[1] = 'z' // and hands out copies
	if again, _ := store.Read("x"); string(again) != "abc" {
		t.Fatalf("store changed through a returned slice: %q", again)
	}
}

func TestMemoryStoreConcurrentUse(t *testing.T) {
	store := NewMemoryStore()
	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				name := fmt.Sprintf("b%d-%d", w, i%10)
				_ = store.Write(name, []byte(name))
				_ = store.Exists(name)
				_, _ = store.Read(name)
			}
		}(w)
	}
	wg.Wait()
	if !store.Exists("b7-9") {
		t.Fatal("missing a written block")
	}
}
