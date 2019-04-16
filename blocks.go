package enstore

import (
	"crypto/rand"
	"errors"
	"fmt"
)

type BlockMetadata struct {
	Filename string
	Size     int64
	Next     string
}

type Block struct {
	Filename string
	Bytes    []byte
}

type BlockReader interface {
	Read(string) ([]byte, error)
}

type BlockWriter interface {
	Write(string, []byte) error
}

func (b *Block) Update(startByte int, newBytes []byte) (int, error) {
	if startByte >= len(b.Bytes) || startByte < 0 {
		return 0, errors.New("start position is outside block")
	}
	for i := 0; i < len(newBytes); i++ {
		if startByte+i >= len(b.Bytes) {
			return i, nil
		}

		b.Bytes[startByte+i] = newBytes[i]
	}
	return len(newBytes), nil
}

func NewBlock(blockName string, blockSize int64) (*Block, error) {
	bytes := make([]byte, blockSize)
	return &Block{blockName, bytes}, nil
}

func ReadBlock(blockName string, key []byte, reader BlockReader) (*Block, error) {
	raw, err := reader.Read(blockName)
	if err != nil {
		return nil, err
	}
	block, err := aesDecrypt(raw, key)
	if err != nil {
		return nil, err
	}
	return &Block{blockName, block}, nil
}

func WriteBlock(block *Block, key []byte, writer BlockWriter) error {
	encrypted, err := aesEncrypt(block.Bytes, key)
	if err != nil {
		return err
	}
	return writer.Write(block.Filename, encrypted)
}

func getNextBlockName(currentBlock string) string {
	// TODO: make this slightly more deterministic based on previous block name
	b := make([]byte, 32)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
