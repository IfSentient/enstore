package enstore

import (
	"crypto/aes"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	good16ByteKey []byte = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	good24ByteKey []byte = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}
	good32ByteKey []byte = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
)

func TestNewAESCrypter(t *testing.T) {
	tests := []struct {
		name          string
		key           []byte
		expected      *AESCrypter
		expectedError error
	}{
		{"Bad key size", []byte{1, 2, 3}, nil, aes.KeySizeError(3)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a, e := NewAESCrypter(test.key)
			assert.Equal(t, test.expected, a)
			assert.Equal(t, test.expectedError, e)
		})
	}
}

func TestAesEncrypt(t *testing.T) {
	tests := []struct {
		testname      string
		toEncrypt     []byte
		key           []byte
		expectedError error
	}{
		{"16-byte key", []byte{0}, good16ByteKey, nil},
		{"24-byte key", []byte{0}, good24ByteKey, nil},
		{"32-byte key", []byte{0}, good32ByteKey, nil},
	}

	for _, test := range tests {
		t.Run(test.testname, func(t *testing.T) {
			c, err := NewAESCrypter(test.key)
			assert.Nil(t, err)

			bytes, err := c.Encrypt(test.toEncrypt)
			if test.expectedError == nil && err == nil {
				assert.Equal(t, len(test.toEncrypt)+aes.BlockSize, len(bytes))
				assert.NotEqual(t, test.toEncrypt, bytes[aes.BlockSize:])
			}
			assert.Equal(t, test.expectedError, err)
		})
	}
}

func TestAesDecrypt(t *testing.T) {
	raw := []byte{9, 8, 7, 6, 5, 4, 3, 2, 1}
	c16, _ := NewAESCrypter(good16ByteKey)
	c24, _ := NewAESCrypter(good24ByteKey)
	c32, _ := NewAESCrypter(good32ByteKey)
	encrypted16, _ := c16.Encrypt(raw)
	encrypted24, _ := c24.Encrypt(raw)
	encrypted32, _ := c32.Encrypt(raw)

	tests := []struct {
		testname      string
		toDecrypt     []byte
		key           []byte
		expectedBytes []byte
		expectedError error
	}{
		{"16-byte key encrypted", encrypted16, good16ByteKey, raw, nil},
		{"24-byte key encrypted", encrypted24, good24ByteKey, raw, nil},
		{"43-byte key encrypted", encrypted32, good32ByteKey, raw, nil},
	}

	for _, test := range tests {
		t.Run(test.testname, func(t *testing.T) {
			c, err := NewAESCrypter(test.key)
			assert.Nil(t, err)

			bytes, err := c.Decrypt(test.toDecrypt)
			assert.Equal(t, test.expectedBytes, bytes)
			assert.Equal(t, test.expectedError, err)
		})
	}
}
