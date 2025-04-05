package config

import "sync"

// NOTE: For full list of supported configs
// check out redis.conf
type Config struct {
	Dir        string
	DBFilename string
	mu         sync.RWMutex
}

func New() *Config {
	return &Config{
		Dir:        "/var/lib/redis", // Default Redis Dir
		DBFilename: "dump.rdb",       // Default Redis DB Filename
	}
}

func (c *Config) Get(arg string) string {
	switch arg {
	case "dir":
		return c.Dir
	case "dbfilename":
		return c.DBFilename
	default:
		return ""
	}
}
