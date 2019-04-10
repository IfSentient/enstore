package main

import (
	"encoding/json"
	"io/ioutil"
)

const (
	CONFIGFILE_PATH   string = "config.json"
	DEFAULT_BLOCKSIZE int    = 5242880 // 5 MB
	DEFAULT_CHUNKSIZE int    = 512     // 512 bytes
	DEFAULT_INDEXFILE string = "index"
)

type Config struct {
	BlockSize int
	ChunkSize int
	IndexFile string
	KeyFile   string
}

func NewDefaultConfig() *Config {
	return &Config{
		BlockSize: DEFAULT_BLOCKSIZE,
		ChunkSize: DEFAULT_CHUNKSIZE,
		IndexFile: DEFAULT_INDEXFILE,
		KeyFile:   "",
	}
}

func LoadConfig(path string) (*Config, error) {
	cfg := NewDefaultConfig()
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytes, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
