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
	accept  bool // it is `false` if the target server is stop
	server  string
	term    uint64
	granted bool
}

type AppendRequest struct {
	term         uint64
	leaderId     string
	prevLogIndex uint64
	prevLogTerm  uint64
	entries      []LogEntry
	leaderCommit uint64
}

type AppendResponse struct {
	accept bool // it is `false` if the target server is stop
	peer   string

	term    uint64
	success bool

	lastLogTerm  uint64
	lastLogIndex uint64
	commitIndex  uint64
}

// message transporting between servers
type Message struct {
	from string
	to   string
	//heartbeat bool
	request      interface{} //VoteRequest or AppendRequest
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
