package raft

type Config struct {
	peers []string
}

func newConfig(peers []string) *Config {
	return &Config{peers}
}
