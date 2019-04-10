package main

import (
	"encoding/json"
	"errors"
	"sort"
)

// IndexReader exposes BlockReader method(s), and a method to check the existence of a file
type IndexReader interface {
	BlockReader
	Exists(filename string) bool
}

// IndexWriter mirrors BlockWriter
type IndexWriter interface {
	BlockWriter
}

// File contains a file's name and contents
type File struct {
	Filename string
	Contents []byte
}

// FileMetadata contains file metadata in the Index
type FileMetadata struct {
	Filename string
	Size     int64
	Blocks   []BlockLocation
}

// BlockLocation describes a section of bytes on a block
type BlockLocation struct {
	Block     string
	StartByte int64
	EndByte   int64
}

type indexJson struct {
	Files      []FileMetadata
	Blocks     map[string]BlockMetadata
	StartBlock string
}

// Index keeps track of all files and blocks and where all files exist across each block.
// It has methods for adding, removing, getting, and listing files on the blocks.
type Index struct {
	files           []FileMetadata
	blocks          map[string]BlockMetadata
	startBlock      string
	fileMap         map[string]*FileMetadata
	blockAllocation map[string][]BlockLocation
	config          *Config
}

// LoadIndex will attempt to load an existing index file and decrypt its store. If no file exists,
// it will create a new one with the supplied key.
func LoadIndex(reader IndexReader, key []byte, cfg *Config) (*Index, error) {
	if !reader.Exists(cfg.IndexFile) {
		return NewIndex(cfg), nil
	}

	var tempIndex indexJson
	data, err := reader.Read(cfg.IndexFile)
	if err != nil {
		return nil, err
	}
	decrypted, err := decrypt(data, key)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(decrypted, &tempIndex); err != nil {
		return nil, err
	}

	// Initialize the index with the values loaded from JSON
	index := Index{
		files:           tempIndex.Files,
		blocks:          tempIndex.Blocks,
		startBlock:      tempIndex.StartBlock,
		fileMap:         make(map[string]*FileMetadata),
		blockAllocation: make(map[string][]BlockLocation),
		config:          cfg,
	}

	// Build out the block allocation and file maps from the loaded data
	for i := 0; i < len(index.files); i++ {
		file := index.files[i]
		index.fileMap[file.Filename] = &index.files[i]
		for _, loc := range file.Blocks {
			locations, ok := index.blockAllocation[loc.Block]
			if !ok {
				locations = make([]BlockLocation, 0)
			}
			locations = append(locations, loc)
			sort.Slice(locations, func(i, j int) bool {
				return locations[i].StartByte < locations[j].StartByte
			})
			index.blockAllocation[loc.Block] = locations
		}
	}

	return &index, nil
}

// NewIndex returns an empty index
func NewIndex(cfg *Config) *Index {
	return &Index{
		files:           make([]FileMetadata, 0),
		blocks:          make(map[string]BlockMetadata),
		startBlock:      "",
		fileMap:         make(map[string]*FileMetadata),
		blockAllocation: make(map[string][]BlockLocation),
		config:          cfg,
	}
}

// Save will save the index encrypted with the supplied key, using the IndexWriter to write the file
func (ix *Index) Save(writer IndexWriter, key []byte) error {
	jsonIndex := indexJson{
		Files:      ix.files,
		Blocks:     ix.blocks,
		StartBlock: ix.startBlock,
	}

	jsonBytes, err := json.Marshal(jsonIndex)
	if err != nil {
		return err
	}
	data, err := encrypt(jsonBytes, key)
	if err != nil {
		return err
	}

	return writer.Write(ix.config.IndexFile, data)
}

// ListFiles returns a slice of all the files in the index.
// This slice is a copy of the internal store, so manipulations can be performed on it
func (ix *Index) ListFiles() []FileMetadata {
	files := make([]FileMetadata, len(ix.files))
	copy(files, ix.files)
	return files
}

// GetFile will read all blocks a file in the index is stored on, and assemble and return the unencrypted file
func (ix *Index) GetFile(filename string, reader BlockReader, key []byte) (*File, error) {
	fileMeta, ok := ix.fileMap[filename]
	if !ok {
		return nil, errors.New("file does not exist in the index")
	}

	bytes := make([]byte, 0)
	for _, loc := range fileMeta.Blocks {
		block, err := ReadBlock(loc.Block, key, reader)
		if err != nil {
			return nil, err
		}
		bytes = append(bytes, block.Bytes[loc.StartByte:loc.EndByte]...)
	}

	return &File{
		Filename: fileMeta.Filename,
		Contents: bytes,
	}, nil
}

// AddFile will add a file to the index and write it to any blocks with space, creating new blocks as necessary
func (ix *Index) AddFile(file *File, reader BlockReader, writer BlockWriter, key []byte) error {
	blockLocations := make([]BlockLocation, 0)
	newBlocks := make(map[string]bool, 0)
	fileSize := int64(len(file.Contents))
	remainingSize := fileSize

	// Iterate through the blocks looking for open chunks
	var block *BlockMetadata
	if ix.startBlock == "" {
		block, _ = ix.nextBlock("")
		ix.startBlock = block.Filename
		newBlocks[block.Filename] = true
	} else {
		b := ix.blocks[ix.startBlock]
		block = &b
	}
	for remainingSize > 0 {
		locs, spaceFound := ix.findSpaceInBlock(block, int(remainingSize))
		remainingSize -= int64(spaceFound)
		blockLocations = append(blockLocations, locs...)
		ix.addBlockAllocations(block.Filename, blockLocations)

		if remainingSize > 0 {
			var isNew bool
			block, isNew = ix.nextBlock(block.Filename)
			if isNew {
				newBlocks[block.Filename] = true
			}
		}
	}

	cursor := 0
	for _, loc := range blockLocations {
		var block *Block
		var err error
		if newBlocks[loc.Block] {
			block, err = NewBlock(loc.Block, ix.blocks[loc.Block].Size)
		} else {
			block, err = ReadBlock(loc.Block, key, reader)
		}
		if err != nil {
			return err
		}

		written, err := block.Update(int(loc.StartByte), file.Contents[cursor:cursor+int(loc.EndByte-loc.StartByte)])
		if err != nil {
			return err
		}

		err = WriteBlock(block, key, writer)
		if err != nil {
			return err
		}

		cursor += written
	}

	ix.files = append(ix.files, FileMetadata{
		Filename: file.Filename,
		Size:     int64(len(file.Contents)),
		Blocks:   blockLocations,
	})

	return nil
}

func (ix *Index) addBlockAllocations(block string, newAllocations []BlockLocation) {
	bAlloc := append(ix.blockAllocation[block], newAllocations...)
	sort.Slice(bAlloc, func(i, j int) bool {
		return bAlloc[i].StartByte < bAlloc[j].StartByte
	})
	ix.blockAllocation[block] = bAlloc
}

func (ix *Index) findSpaceInBlock(block *BlockMetadata, space int) ([]BlockLocation, int) {
	remainingSize := int64(space)
	blockLocations := make([]BlockLocation, 0)

	allocations, ok := ix.blockAllocation[block.Filename]
	// If there are no allocations on this block, we can use as much of it as we need
	if !ok || len(allocations) == 0 {
		writeLength := remainingSize
		if writeLength > block.Size {
			writeLength = block.Size
		}
		remainingSize -= writeLength
		allocation := BlockLocation{
			Block:     block.Filename,
			StartByte: 0,
			EndByte:   writeLength,
		}
		blockLocations = append(blockLocations, allocation)
		newAllocation := make([]BlockLocation, 0)
		newAllocation = append(newAllocation, allocation)
		ix.blockAllocation[block.Filename] = newAllocation
	} else {
		// Otherwise, look for gaps in the allocations of at least CHUNKSIZE,
		// and create allocations for bits of the file there
		last := int64(0)
		newAllocations := allocations
		for _, allocation := range allocations {
			if allocation.StartByte-last >= int64(ix.config.ChunkSize) {
				// We can add a chunk
				chunkSize := allocation.StartByte - last
				if chunkSize > remainingSize {
					chunkSize = remainingSize
					if chunkSize == 0 {
						return blockLocations, space
					}
				}
				newAllocation := BlockLocation{
					Block:     block.Filename,
					StartByte: last,
					EndByte:   last + chunkSize,
				}
				blockLocations = append(blockLocations, newAllocation)
				newAllocations = append(newAllocations, newAllocation)
				remainingSize -= chunkSize
			}
			last = allocation.EndByte
		}

		// If we've found all the space we need, we can stop here
		if remainingSize == 0 {
			return blockLocations, space
		}

		// Check the end of the block for space
		if block.Size-last >= int64(ix.config.ChunkSize) {
			// We can add a chunk
			chunkSize := block.Size - last
			if chunkSize > remainingSize {
				chunkSize = remainingSize
				if chunkSize == 0 {
					return blockLocations, space
				}
			}
			newAllocation := BlockLocation{
				Block:     block.Filename,
				StartByte: last,
				EndByte:   last + chunkSize,
			}
			blockLocations = append(blockLocations, newAllocation)
			newAllocations = append(newAllocations, newAllocation)
			remainingSize -= chunkSize
		}
	}

	return blockLocations, space - int(remainingSize)
}

func (ix *Index) DeleteFile(filename string, reader BlockReader, writer BlockWriter, key []byte, zeroOut bool) error {
	fileMeta, ok := ix.fileMap[filename]
	if !ok {
		return errors.New("file does not exist in the index")
	}

	var block *Block
	var err error
	if zeroOut {
		block, err = ReadBlock(fileMeta.Blocks[0].Block, key, reader)
		if err != nil {
			return err
		}
	}
	for _, allocation := range fileMeta.Blocks {
		if zeroOut {
			if block.Filename != allocation.Block {
				err = WriteBlock(block, key, writer)
				if err != nil {
					return err
				}
				block, err = ReadBlock(allocation.Block, key, reader)
				if err != nil {
					return err
				}
			}

			bytes := make([]byte, allocation.EndByte-allocation.StartByte)
			block.Update(int(allocation.StartByte), bytes)
		}

		allocations := ix.blockAllocation[allocation.Block]
		for idx, all := range allocations {
			if all.Block == allocation.Block && all.StartByte == allocation.StartByte && all.EndByte == allocation.EndByte {
				ix.blockAllocation[allocation.Block] = append(allocations[:idx], allocations[idx+1:]...)
				break
			}
		}
	}
	if zeroOut {
		err = WriteBlock(block, key, writer)
		if err != nil {
			return err
		}
	}

	delete(ix.fileMap, fileMeta.Filename)
	for i, f := range ix.files {
		if f.Filename == fileMeta.Filename {
			ix.files = append(ix.files[:i], ix.files[i+1:]...)
			break
		}
	}

	return nil
}

func (ix *Index) nextBlock(curBlock string) (*BlockMetadata, bool) {
	if curBlock == "" {
		newBlock := BlockMetadata{
			Filename: getNextBlockName(""),
			Size:     int64(ix.config.BlockSize),
			Next:     "",
		}
		ix.blocks[newBlock.Filename] = newBlock
		return &newBlock, true
	}
	curBlockMeta := ix.blocks[curBlock]
	if curBlockMeta.Next != "" {
		nextBlock := ix.blocks[curBlockMeta.Next]
		return &nextBlock, false
	}

	curBlockMeta.Next = getNextBlockName(curBlock)
	ix.blocks[curBlock] = curBlockMeta
	newBlock := BlockMetadata{
		Filename: curBlockMeta.Next,
		Size:     int64(ix.config.BlockSize),
		Next:     "",
	}
	ix.blocks[newBlock.Filename] = newBlock
	return &newBlock, true
}
