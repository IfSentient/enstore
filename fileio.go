package enstore

import (
	"fmt"
	"io/ioutil"
	"os"
)

type LocalFileReadWriter struct {
	BasePath string
}

func (lfrw *LocalFileReadWriter) Read(filename string) ([]byte, error) {
	if lfrw.BasePath != "" {
		if lfrw.BasePath[len(lfrw.BasePath)-1] == '/' {
			filename = fmt.Sprintf("%s%s", lfrw.BasePath, filename)
		} else {
			filename = fmt.Sprintf("%s/%s", lfrw.BasePath, filename)
		}
	}
	return ioutil.ReadFile(filename)
}

func (lfrw *LocalFileReadWriter) Write(filename string, bytes []byte) error {
	if lfrw.BasePath != "" {
		if lfrw.BasePath[len(lfrw.BasePath)-1] == '/' {
			filename = fmt.Sprintf("%s%s", lfrw.BasePath, filename)
		} else {
			filename = fmt.Sprintf("%s/%s", lfrw.BasePath, filename)
		}
	}
	return ioutil.WriteFile(filename, bytes, 0644)
}

func (lfrw *LocalFileReadWriter) Exists(filename string) bool {
	if lfrw.BasePath != "" {
		if lfrw.BasePath[len(lfrw.BasePath)-1] == '/' {
			filename = fmt.Sprintf("%s%s", lfrw.BasePath, filename)
		} else {
			filename = fmt.Sprintf("%s/%s", lfrw.BasePath, filename)
		}
	}
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}
	return true
}
