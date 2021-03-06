package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"

	"github.com/IfSentient/enstore"
)

type fileWrapper struct {
	file io.Reader
	size int64
	name string
}

func (f *fileWrapper) Read(b []byte) (int, error) {
	return f.file.Read(b)
}

func (f *fileWrapper) Size() int64 {
	return f.size
}

func (f *fileWrapper) Name() string {
	return f.name
}

func main() {
	keyArg := flag.String("key", "", "key")
	keyFileArg := flag.String("keyfile", "", "key file")
	addFileArg := flag.String("add-file", "", "file to add")
	getFileArg := flag.String("get-file", "", "filename to retrieve")
	delFileArg := flag.String("delete-file", "", "filemame to delete")
	configFileArg := flag.String("config", "", "config file path")
	outputArg := flag.String("o", "", "output file")

	flag.Parse()

	var cfg *enstore.Config
	var keyFile string
	var storeDir string
	if *configFileArg != "" {
		var err error
		cfg, keyFile, storeDir, err = loadConfig(*configFileArg)
		if err != nil {
			panic(err)
		}
	} else {
		// Look in default config file location, "config.json"
		if _, err := os.Stat(enstore.ConfigfilePath); !os.IsNotExist(err) {
			var err error
			cfg, keyFile, storeDir, err = loadConfig(enstore.ConfigfilePath)
			if err != nil {
				panic(err)
			}
		}
	}

	if *keyFileArg != "" {
		keyFile = *keyFileArg
	}

	var initialKey []byte
	if *keyArg != "" {
		initialKey = []byte(*keyArg)
	} else if keyFile != "" {
		var err error
		initialKey, err = ioutil.ReadFile(keyFile)
		if err != nil {
			panic(err)
		}
	} else {
		panic("either -key or -keyfile is required or KeyFile must be specified in the config")
	}

	hasher := md5.New()
	hasher.Write([]byte(initialKey))
	key := hasher.Sum(nil)

	// Make the AES crypter from the key
	crypter, err := enstore.NewAESCrypter(key)
	if err != nil {
		panic(err)
	}

	// File I/O
	blockInterfacer := enstore.LocalFileReadWriter{
		BasePath: storeDir,
	}

	// Index
	index, err := enstore.LoadIndex(&blockInterfacer, crypter, cfg)
	if err != nil {
		panic(err)
	}

	if *addFileArg != "" {
		finfo, err := os.Stat(*addFileArg)
		if err != nil {
			panic(err)
		}
		file, err := os.OpenFile(*addFileArg, os.O_RDONLY, 0644)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		err = index.AddFile(&fileWrapper{file, finfo.Size(), *addFileArg}, &blockInterfacer, &blockInterfacer, crypter)
		if err != nil {
			panic(err)
		}

		err = index.Save(&blockInterfacer, crypter)
		if err != nil {
			panic(err)
		}
		return
	}

	if *getFileArg != "" {
		var writer io.Writer
		if *outputArg != "" {
			file, err := os.OpenFile(*outputArg, os.O_WRONLY|os.O_CREATE, 0644)
			if err != nil {
				panic(err)
			}
			defer file.Close()
			writer = file
		} else {
			buffer := &bytes.Buffer{}
			defer fmt.Println(buffer.Bytes())
			writer = buffer
		}

		err := index.GetFile(*getFileArg, writer, &blockInterfacer, crypter)
		if err != nil {
			panic(err)
		}
		return
	}

	if *delFileArg != "" {
		err := index.DeleteFile(*delFileArg, &blockInterfacer, &blockInterfacer, crypter, true)
		if err != nil {
			panic(err)
		}
		err = index.Save(&blockInterfacer, crypter)
		if err != nil {
			panic(err)
		}
		return
	}

	fmt.Println("Files:")
	files := index.ListFiles()
	sort.Slice(files, func(i, j int) bool {
		return files[i].Filename < files[j].Filename
	})
	for _, file := range files {
		fmt.Println("\t" + file.Filename)
	}

}

type cliConfig struct {
	KeyFile  string
	StoreDir string
}

// LoadConfig attempts to load a JSON file at a path into a new default Config
func loadConfig(path string) (*enstore.Config, string, string, error) {
	cfg := enstore.NewDefaultConfig()
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, "", "", err
	}
	err = json.Unmarshal(bytes, cfg)
	if err != nil {
		return nil, "", "", err
	}
	ccfg := &cliConfig{}
	err = json.Unmarshal(bytes, ccfg)
	if err != nil {
		return nil, "", "", err
	}

	return cfg, ccfg.KeyFile, ccfg.StoreDir, nil
}
