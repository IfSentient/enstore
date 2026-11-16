package main

import (
	"bytes"
	"crypto/md5"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/scrypt"
)

// deriveKey(passphrase, storeDir, indexFile) returns the store's AES key and
// whether it is a legacy store:
//
//   - a store with a "salt" file: scrypt(passphrase, salt, N=32768, r=8, p=1), 32 bytes
//   - a new store (no salt file, no index file): a random 32-byte salt is written
//     to "salt" (mode 0600) first, then as above
//   - a legacy store (an index file but no salt file): the old key, MD5 of the
//     passphrase, with legacy true; no salt file is written

func TestNewStoreGetsASaltAndAnScryptKey(t *testing.T) {
	dir := t.TempDir()
	key, legacy, err := deriveKey([]byte("correct horse"), dir, "index")
	if err != nil || legacy {
		t.Fatalf("new store: legacy %v, err %v", legacy, err)
	}
	salt, err := os.ReadFile(filepath.Join(dir, "salt"))
	if err != nil || len(salt) != 32 {
		t.Fatalf("salt file: %d bytes, %v", len(salt), err)
	}
	if fi, _ := os.Stat(filepath.Join(dir, "salt")); fi.Mode().Perm() != 0o600 {
		t.Errorf("salt file mode %v, want 0600", fi.Mode().Perm())
	}
	want, _ := scrypt.Key([]byte("correct horse"), salt, 32768, 8, 1, 32)
	if !bytes.Equal(key, want) {
		t.Error("key isn't scrypt(passphrase, salt, 32768, 8, 1, 32)")
	}
	again, _, err := deriveKey([]byte("correct horse"), dir, "index")
	if err != nil || !bytes.Equal(again, key) {
		t.Errorf("the same store gave a different key the second time (%v)", err)
	}
	if after, _ := os.ReadFile(filepath.Join(dir, "salt")); !bytes.Equal(after, salt) {
		t.Error("the salt changed on the second use")
	}
}

func TestSamePassphraseDifferentStoresDifferentKeys(t *testing.T) {
	a, _, err := deriveKey([]byte("correct horse"), t.TempDir(), "index")
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := deriveKey([]byte("correct horse"), t.TempDir(), "index")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a, b) {
		t.Error("two stores with the same passphrase got the same key")
	}
}

func TestLegacyStoreKeepsItsMD5Key(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "my-index"), []byte("an index written by the old CLI"), 0o644); err != nil {
		t.Fatal(err)
	}
	key, legacy, err := deriveKey([]byte("correct horse"), dir, "my-index")
	if err != nil || !legacy {
		t.Fatalf("legacy store: legacy %v, err %v", legacy, err)
	}
	sum := md5.Sum([]byte("correct horse"))
	if !bytes.Equal(key, sum[:]) {
		t.Error("a legacy store's key must stay MD5(passphrase)")
	}
	if _, err := os.Stat(filepath.Join(dir, "salt")); !os.IsNotExist(err) {
		t.Error("a salt file was written for a legacy store")
	}
}
