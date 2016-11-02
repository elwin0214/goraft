package raft

type Config struct {
	peers []string
}

func NewConfig(peers []string) *Config {
	return &Config{peers}
}
