package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	"github.com/IfSentient/enstore/connectors"
)

type AllReadWriter interface {
	IndexReader
	IndexWriter
}

func main() {
	keyArg := flag.String("key", "", "key")
	keyFileArg := flag.String("keyfile", "", "key file")
	addFileArg := flag.String("add-file", "", "file to add")
	getFileArg := flag.String("get-file", "", "filename to retrieve")
	delFileArg := flag.String("delete-file", "", "filemame to delete")
	configFileArg := flag.String("config", "", "config file path")
	s3ConfigFileArg := flag.String("s3-config", "", "s3 config file path")
	outputArg := flag.String("o", "", "output file")

	flag.Parse()

	cfg := NewDefaultConfig()
	if *configFileArg != "" {
		var err error
		cfg, err = LoadConfig(*configFileArg)
		if err != nil {
			panic(err)
		}
	} else {
		// Look in default config file location, "config.json"
		if _, err := os.Stat(ConfigfilePath); !os.IsNotExist(err) {
			var err error
			cfg, err = LoadConfig(ConfigfilePath)
			if err != nil {
				panic(err)
			}
		}
	}

	var connector AllReadWriter
	if *s3ConfigFileArg != "" {
		s3cfg, err := connectors.LoadS3Config(*s3ConfigFileArg)
		if err != nil {
			panic(err)
		}
		connector, err = connectors.NewS3Connector(*s3cfg)
		if err != nil {
			panic(err)
		}
	} else {
		connector = &connectors.LocalFileReadWriter{}
	}

	var initialKey []byte
	if *keyFileArg != "" {
		var err error
		initialKey, err = ioutil.ReadFile(*keyFileArg)
		if err != nil {
			panic(err)
		}
	} else if *keyArg != "" {
		initialKey = []byte(*keyArg)
	} else if cfg.KeyFile != "" {
		var err error
		initialKey, err = ioutil.ReadFile(cfg.KeyFile)
		if err != nil {
			panic(err)
		}
	} else {
		panic("either -key or -keyfile is required or KeyFile must be specified in the config")
	}

	hasher := md5.New()
	hasher.Write([]byte(initialKey))
	key := hasher.Sum(nil)

	// File I/O
	//blockInterfacer := connectors.LocalFileReadWriter{}

	// Index
	index, err := LoadIndex(connector, key, cfg)
	if err != nil {
		panic(err)
	}

	if *addFileArg != "" {
		bytes, err := ioutil.ReadFile(*addFileArg)
		if err != nil {
			panic(err)
		}

		err = index.AddFile(&File{*addFileArg, bytes}, connector, connector, key)
		if err != nil {
			panic(err)
		}
	}

	if *getFileArg != "" {
		file, err := index.GetFile(*getFileArg, connector, key)
		if err != nil {
			panic(err)
		}
		if *outputArg != "" {
			err = ioutil.WriteFile(*outputArg, file.Contents, 0644)
		} else {
			fmt.Println(string(file.Contents))
		}
	}

	if *delFileArg != "" {
		err := index.DeleteFile(*delFileArg, connector, connector, key, true)
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("Files:")
	files := index.ListFiles()
	sort.Slice(files, func(i, j int) bool {
		return files[i].Filename < files[j].Filename
	})
	for _, file := range files {
		fmt.Println("\t" + file.Filename)
	}

	err = index.Save(connector, key)
	if err != nil {
		panic(err)
	}

}
