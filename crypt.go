package enstore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

// AESCrypter is a struct that can encrypt and decypt bytes using AES and a common key
type AESCrypter struct {
	key []byte // TODO: should we store the cipher here instead?
}

// NewAESCrypter creates a new AESCrypter with the provided key
func NewAESCrypter(key []byte) (*AESCrypter, error) {
	_, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &AESCrypter{key}, nil
}

// Encrypt encrypts the bytes passed to it
func (a *AESCrypter) Encrypt(bytes []byte) ([]byte, error) {
	// Get the cipher using the key
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return nil, err
	}

	// CipherText length is payload length + AES blocksize (for the IV)
	cipherText := make([]byte, aes.BlockSize+len(bytes))

	// Generated the IV
	iv := cipherText[:aes.BlockSize]

	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// Return an encrypted stream
	stream := cipher.NewCFBEncrypter(block, iv)

	// Encrypt bytes from plaintext to ciphertext (keeping IV at the font)
	stream.XORKeyStream(cipherText[aes.BlockSize:], bytes)

	return cipherText, nil
}

// Decrypt decrypts the bytes passed to it
func (a *AESCrypter) Decrypt(bytes []byte) ([]byte, error) {
	// Get the cipher using the key
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return nil, err
	}

	// The length of bytes to decrupt must be at least the AES blocksize (as it is preceeded by the IV)
	if len(bytes) < aes.BlockSize {
		return nil, errors.New("text is too short")
	}

	// Get the IV from the beginning
	iv := bytes[:aes.BlockSize]

	// The remainder if the actual ciphertext
	ciphertext := bytes[aes.BlockSize:]

	// Get the stream from the cipher and IV
	stream := cipher.NewCFBDecrypter(block, iv)

	// Decrypt the cyphertext
	stream.XORKeyStream(ciphertext, ciphertext)

	return ciphertext, nil
}

// Encrypter returns an io.Reader which, when read, returns the encrypted bytes of the `source` io.Reader
func (a *AESCrypter) Encrypter(source io.Reader) (io.Reader, error) {
	//Make the cipher
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return nil, err
	}

	// Create the IV
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// Get the stream
	stream := cipher.NewCFBEncrypter(block, iv)

	// Return the encrypter
	return &AESEncryptionReader{
		Source: source,
		Stream: stream,
		IV:     iv,
	}, nil
}

// Decrypter returns an io.Reader which, when read, returns the decrypted bytes of the `source` io.Reader
func (a *AESCrypter) Decrypter(source io.Reader) (io.Reader, error) {
	// Make the cipher
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return nil, err
	}

	// Read the IV from the source
	iv := make([]byte, aes.BlockSize)
	_, err = source.Read(iv)
	if err != nil {
		return nil, err
	}

	// Get the stream
	stream := cipher.NewCFBDecrypter(block, iv)

	// Return the decrypter
	return &AESDecryptionReader{
		Source: source,
		Stream: stream,
	}, nil
}

// AESEncyptionReader is an io.Reader which provides the encrypted version of a source io.Reader when read
type AESEncryptionReader struct {
	Source io.Reader
	Stream cipher.Stream
	IV     []byte
	cursor int
}

// Read returns the encrypted bytes of the source io.Reader, starting with the IV
func (e *AESEncryptionReader) Read(p []byte) (int, error) {
	offset := 0

	// Check if we're still in the IV block at the beginning of the read
	// If we are, we need to finish returning that before we begin reading from `Source`
	if e.cursor < len(e.IV) {
		part := len(e.IV) - e.cursor
		if len(p) <= part {
			copy(p, e.IV[e.cursor:e.cursor+len(p)])
			e.cursor += len(p)
			return len(p), nil
		}
		copy(p[:part], e.IV[e.cursor:])
		e.cursor += part
		offset = part
	}

	// Read from `Source`, and encrypt those bytes before returning them
	n, readErr := e.Source.Read(p[offset:])
	if n > 0 {
		e.Stream.XORKeyStream(p[offset:offset+n], p[offset:offset+n])
		return n + offset, readErr
	}
	return 0, io.EOF
}

// AESDecryptionReader is an io.Reader which provides the decrypted version of a source io.Reader when read
// The source io.Reader should be positioned at the start of the ciphertext when initialized
type AESDecryptionReader struct {
	Source io.Reader
	Stream cipher.Stream
}

// Read returns the decrypted bytes of the source io.Reader
func (d *AESDecryptionReader) Read(p []byte) (int, error) {
	n, readErr := d.Source.Read(p)
	if n > 0 {
		d.Stream.XORKeyStream(p[:n], p[:n])
		return n, readErr
	}
	return 0, io.EOF
}
