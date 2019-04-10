# enstore

EnStore is a library for placing files into encrypted blocks, which can be written to an arbitrary destination, and being able to list and retrieve said files without having to pull down an entire encrypted volume (only blocks which contain the file being retrieved will be read and decrypted). Adding new files will also only pull and push the necessary blocks. The only place files are ever decrypted is in-memory (unless the decrypted file is written out to a file by the user).

## Usage
EnStore is a library, but does expose a command-line interface to use as a simple tool. See usage as a library, or as a CLI.

### Library Usage
TODO
### CLI
The CLI can be compiled from source using
```bash
$ go build -o enstore
```
go must be installed to build. Pre-built binaries may be forthcoming.

TODO