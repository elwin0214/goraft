package raft

type Header struct {
	from string
	to   string
}

type VoteRequest struct {
	term         uint64
	candidateId  string
	lastLogTerm  uint64
	lastLogIndex uint64
}

type VoteResponse struct {
	term    uint64
	granted bool
}

type AppendRequest struct {
	term         uint64
	leaderId     string
	prevLogIndex uint64
	prevLogTerm  uint64
	entries      []*LogEntry
	leaderCommit uint64
}

type AppendResponse struct {
	peer string

	term    uint64
	success bool

	lastLogTerm  uint64
	lastLogIndex uint64
	commitIndex  uint64
}

type Message struct {
	from string
	to   string
	//heartbeat bool
	request      interface{}
	responseChan chan interface{}
}

type ElectMessage struct {
	Server string
	Term   uint64
}

type CommitMessage struct {
	Server      string
	Term        uint64
	CommitIndex uint64
}
