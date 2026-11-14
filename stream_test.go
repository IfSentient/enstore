package enstore

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"
	"testing/iotest"
)

func streamCrypter(t *testing.T) *AESCrypter {
	t.Helper()
	c, err := NewAESCrypter(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// zeroFirst returns (0, nil) on its first read, which the io.Reader contract
// allows, then reads from r.
type zeroFirst struct {
	r    io.Reader
	done bool
}

func (z *zeroFirst) Read(p []byte) (int, error) {
	if !z.done {
		z.done = true
		return 0, nil
	}
	return z.r.Read(p)
}

func streamRoundTrip(t *testing.T, c *AESCrypter, plain []byte, wrapEnc, wrapDec func(io.Reader) io.Reader) {
	t.Helper()
	enc, err := c.Encrypter(wrapEnc(bytes.NewReader(plain)))
	if err != nil {
		t.Fatal(err)
	}
	cipherText, err := io.ReadAll(enc)
	if err != nil {
		t.Fatalf("%d bytes: encrypt: %v", len(plain), err)
	}
	if len(cipherText) != len(plain)+16 {
		t.Fatalf("%d bytes: ciphertext is %d bytes, want IV + plaintext = %d", len(plain), len(cipherText), len(plain)+16)
	}
	dec, err := c.Decrypter(wrapDec(bytes.NewReader(cipherText)))
	if err != nil {
		t.Fatalf("%d bytes: decrypter: %v", len(plain), err)
	}
	got, err := io.ReadAll(dec)
	if err != nil {
		t.Fatalf("%d bytes: decrypt: %v", len(plain), err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("%d bytes: round trip differs", len(plain))
	}
}

func TestStreamingRoundTripsWithAnyReader(t *testing.T) {
	c := streamCrypter(t)
	id := func(r io.Reader) io.Reader { return r }
	wrappers := map[string]func(io.Reader) io.Reader{
		"plain":      id,
		"one byte":   iotest.OneByteReader,
		"half":       iotest.HalfReader,
		"data+err":   iotest.DataErrReader,
		"zero first": func(r io.Reader) io.Reader { return &zeroFirst{r: r} },
	}
	for _, size := range []int{0, 1, 15, 16, 17, 1000, 100000} {
		plain := make([]byte, size)
		rand.Read(plain)
		for name, w := range wrappers {
			t.Run(name, func(t *testing.T) {
				streamRoundTrip(t, c, plain, w, id)
				streamRoundTrip(t, c, plain, id, w)
			})
		}
	}
}

func TestDecrypterFollowsTheReaderContract(t *testing.T) {
	c := streamCrypter(t)
	plain := make([]byte, 5000)
	rand.Read(plain)
	enc, _ := c.Encrypter(bytes.NewReader(plain))
	cipherText, _ := io.ReadAll(enc)
	dec, err := c.Decrypter(iotest.HalfReader(bytes.NewReader(cipherText)))
	if err != nil {
		t.Fatal(err)
	}
	if err := iotest.TestReader(dec, plain); err != nil {
		t.Fatal(err)
	}
}
