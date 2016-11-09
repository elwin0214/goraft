package raft

type RaftState struct {
	VotedFor    string
	CurrentTerm uint64

	CommitIndex uint64
	LastApplied uint64
}

func newRaftState() RaftState {
	return RaftState{"", 0, 0, 0}
}
