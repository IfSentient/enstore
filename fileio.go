package main

import (
	"io/ioutil"
	"os"
)

type LocalFileReadWriter struct {
}

func (lfrw *LocalFileReadWriter) Read(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}

func (lfrw *LocalFileReadWriter) Write(filename string, bytes []byte) error {
	return ioutil.WriteFile(filename, bytes, 0644)
}

func (lfrw *LocalFileReadWriter) Exists(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}
	return true
}
