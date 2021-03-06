package enstore

const (
	// ConfigfilePath is the default config file path
	ConfigfilePath string = "config.json"
	// DefaultBlockSize is the default block size (5 MB)
	DefaultBlockSize int = 5242880 // 5 MB
	// DefaultChunkSize is the default chunk size (512 bytes)
	DefaultChunkSize int = 512 // 512 bytes
	// DefaultIndexfile is the default index file path
	DefaultIndexfile string = "index"
)

// Config is the basic configuration for enstore
type Config struct {
	BlockSize int
	ChunkSize int
	IndexFile string
}

// NewDefaultConfig returns a pointer to a new Config with default values
func NewDefaultConfig() *Config {
	return &Config{
		BlockSize: DefaultBlockSize,
		ChunkSize: DefaultChunkSize,
		IndexFile: DefaultIndexfile,
	}
}
