package raft

type RaftState struct {
	VotedFor    string
	CurrentTerm uint64

	CommitIndex uint64
	LastApplied uint64
	LeaderId    string
}

func NewRaftState() *RaftState {
	return &RaftState{"", 0, 0, 0, ""}
}
