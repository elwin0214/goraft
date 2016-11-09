package raft

// command sent by client
type Command struct {
	Op    string
	Key   string
	Value string
}

type Request struct {
	command     interface{}
	reponseChan chan bool
}
