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
	key []byte
}

// NewAESCrypter creates a new AESCrypter with the provided key
func NewAESCrypter(key []byte) (*AESCrypter, error) {
	_, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &AESCrypter{key}, nil
}

// Encrypt encrypts
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
