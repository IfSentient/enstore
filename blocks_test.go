package enstore

import (
	"crypto/aes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockBlockReader struct {
	ReadFunc func(string) ([]byte, error)
}

func (br *mockBlockReader) Read(blockname string) ([]byte, error) {
	if br.ReadFunc != nil {
		return br.ReadFunc(blockname)
	}
	return nil, nil
}

type mockBlockWriter struct {
	WriteFunc func(string, []byte) error
}

func (bw *mockBlockWriter) Write(blockname string, bytes []byte) error {
	if bw.WriteFunc != nil {
		return bw.WriteFunc(blockname, bytes)
	}
	return nil
}

func TestBlockUpdate(t *testing.T) {
	tests := []struct {
		testname             string
		block                *Block
		startByte            int
		bytes                []byte
		newBytes             []byte
		expectedBytesWritten int
		expectedError        error
	}{
		{"Start past end of block", &Block{Bytes: []byte{1, 2, 3}}, 3, []byte{5}, []byte{1, 2, 3}, 0, errors.New("start position is outside block")},
		{"Start before start of block", &Block{Bytes: []byte{1, 2, 3}}, -1, []byte{5}, []byte{1, 2, 3}, 0, errors.New("start position is outside block")},
		{"Only 1 byte written", &Block{Bytes: []byte{1, 2, 3}}, 2, []byte{5, 6}, []byte{1, 2, 5}, 1, nil},
		{"All bytes written", &Block{Bytes: []byte{1, 2, 3}}, 0, []byte{5, 6}, []byte{5, 6, 3}, 2, nil},
		{"Overwrite all bytes", &Block{Bytes: []byte{1, 2, 3}}, 0, []byte{5, 6, 7, 8}, []byte{5, 6, 7}, 3, nil},
	}

	for _, test := range tests {
		written, err := test.block.Update(test.startByte, test.bytes)
		assert.Equal(t, test.expectedBytesWritten, written)
		assert.Equal(t, test.expectedError, err)
		assert.Equal(t, test.newBytes, test.block.Bytes)
	}
}

func TestReadBlock(t *testing.T) {
	blockNameReadError := "badReadBlock"
	readError := errors.New("I AM ERROR")

	tests := []struct {
		testname      string
		blockname     string
		key           []byte
		expectedBlock *Block
		expectedError error
	}{
		{"Read error", blockNameReadError, good16ByteKey, nil, readError},
		{"Decrypt error", "b10ck4", []byte{1, 2, 3}, nil, aes.KeySizeError(3)},
		{"Success", "b10ck4", good16ByteKey, &Block{Filename: "b10ck4", Bytes: []byte("b10ck4")}, nil},
	}

	reader := mockBlockReader{
		ReadFunc: func(block string) ([]byte, error) {
			if block == blockNameReadError {
				return nil, readError
			}
			bytes, err := aesEncrypt([]byte(block), good16ByteKey)
			if err != nil {
				return []byte{}, nil
			}
			return bytes, nil
		},
	}

	for _, test := range tests {
		t.Run(test.testname, func(t *testing.T) {
			block, err := ReadBlock(test.blockname, test.key, &reader)
			assert.Equal(t, test.expectedError, err)
			if test.expectedBlock != nil {
				assert.Equal(t, test.expectedBlock.Filename, block.Filename)
				assert.Equal(t, test.expectedBlock.Bytes, block.Bytes)
			}
		})
	}
}

func TestWriteBlock(t *testing.T) {
	blockNameWriteError := "badWriteBlock"
	writeError := errors.New("I AM ERROR")

	tests := []struct {
		testname      string
		blockname     string
		bytes         []byte
		key           []byte
		expectedError error
	}{
		{"Bad key size", blockNameWriteError, []byte{1, 2, 3}, []byte{1, 2, 3}, aes.KeySizeError(3)},
		{"Write error", blockNameWriteError, []byte{1, 2, 3}, good16ByteKey, writeError},
		{"Success", "b10ck4", []byte{1, 2, 3}, good16ByteKey, nil},
	}

	var bytesWritten []byte
	writer := mockBlockWriter{
		WriteFunc: func(block string, bytes []byte) error {
			if block == blockNameWriteError {
				return writeError
			}
			bytesWritten = bytes
			return nil
		},
	}

	for _, test := range tests {
		t.Run(test.testname, func(t *testing.T) {
			err := WriteBlock(&Block{Filename: test.blockname, Bytes: test.bytes}, test.key, &writer)
			assert.Equal(t, test.expectedError, err)
			if test.expectedError == nil {
				assert.Equal(t, aes.BlockSize+len(test.bytes), len(bytesWritten))
			}
		})
	}
}
