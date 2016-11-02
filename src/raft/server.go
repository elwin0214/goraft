package raft

import (
	"sync"
	"time"
)

const (
	Stopped     = "stopped"
	Initialized = "initialized"
	Follower    = "follower"
	Candidate   = "candidate"
	Leader      = "leader"
)

const (
	DefaultHeartbeatInterval = 1000 * time.Millisecond
	DefaultElectionTimeout   = 3000 * time.Millisecond
)

type Peer struct {
	nextIndex  uint64
	matchIndex uint64
	appendChan chan *Message
}

type Server struct {
	mutex sync.Mutex
	name  string

	voteChan       chan *Message        // receive  requestvote
	appendChan     chan *Message        // receive appendentity
	appendRespChan chan *AppendResponse // receive appendentity response

	stopChan    chan bool
	observeChan chan interface{}

	commitIndex uint64
	applyIndex  uint64

	peers map[string]*Peer

	log   *LogStore
	state string
	raft  *RaftState
	trans Transporter

	config *Config

	logger Logger
}

func NewServer(name string, log *LogStore, raft *RaftState, config *Config, observeChan chan interface{}) *Server {
	server := &Server{name: name, log: log, raft: raft}
	server.logger.Init()
	server.voteChan = make(chan *Message, 1024)
	server.appendChan = make(chan *Message, 1024)
	server.appendRespChan = make(chan *AppendResponse, 1024)

	server.stopChan = make(chan bool, 1)
	server.observeChan = observeChan
	server.peers = make(map[string]*Peer)

	server.commitIndex = 0
	server.applyIndex = 0

	server.config = config
	server.state = Stopped
	for _, p := range server.config.peers {
		server.addPeer(p)
	}
	return server
}

func (s *Server) addPeer(p string) {
	s.peers[p] = &Peer{1, 0, make(chan *Message, 128)}
}

func (s *Server) commit(index uint64) {
	s.raft.CommitIndex = index
	s.log.commit(index)
	if s.raft.LastApplied < index {
		s.raft.LastApplied = index
	}
	s.observeChan <- &CommitMessage{Server: s.name, Term: s.raft.CurrentTerm, CommitIndex: s.raft.CommitIndex}
}

func (s *Server) getCommitIndex() uint64 {
	return s.raft.CommitIndex
}

func (s *Server) GetVoteChan() chan *Message {
	return s.voteChan
}

func (s *Server) GetAppendChan() chan *Message {
	return s.appendChan
}

func (s *Server) GetObserveChan() chan interface{} {
	return s.observeChan
}

func (s *Server) SetTransporter(trans Transporter) {
	s.trans = trans
}

func (s *Server) GetState() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.state
}

func (s *Server) SetState(state string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.state = state
}

func (s *Server) Start() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.state != Stopped {
		s.logger.Error.Printf("%s state is started!", s.name)
		return
	}
	s.state = Follower
	s.logger.Info.Printf("%s state is Follwer!", s.name)
	go s.loop()
}

func (s *Server) Stop() {
	s.stopChan <- true
}

func (s *Server) loop() {
	for s.GetState() != Stopped {
		switch s.state {
		case Follower:
			s.followerLoop()
		case Candidate:
			s.candidateLoop()
		case Leader:
			s.leaderLoop()
		}
	}
}

func (s *Server) quorumSize() int {
	return (len(s.peers)/2 + 1)
}

func (s *Server) handleVoteRequest(request *VoteRequest) *VoteResponse {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if request.term < s.raft.CurrentTerm {
		s.logger.Info.Printf("[%s][handleVoteRequest] current term = %d, reject term = %d\n", s.name, s.raft.CurrentTerm, request.term)
		return &VoteResponse{s.raft.CurrentTerm, false}
	}

	if request.term > s.raft.CurrentTerm {
		s.raft.CurrentTerm = request.term
		s.raft.VotedFor = ""
	}

	if request.term == s.raft.CurrentTerm {
		if "" != s.raft.VotedFor && request.candidateId != s.raft.VotedFor {
			return &VoteResponse{s.raft.CurrentTerm, false}
		}
	}

	lastLogIndex, lastLogTerm := s.log.getLast()
	if (lastLogTerm == request.lastLogTerm && lastLogIndex > request.lastLogIndex) || lastLogTerm > request.lastLogTerm {
		return &VoteResponse{s.raft.CurrentTerm, false}
	}
	s.raft.VotedFor = request.candidateId
	s.logger.Info.Printf("[%s][handleVoteRequest] current term = %d, vote for = %s\n", s.name, s.raft.CurrentTerm, request.candidateId)
	return &VoteResponse{s.raft.CurrentTerm, true}
}

func (s *Server) processVoteResponse(resp *VoteResponse) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if resp.term == s.raft.CurrentTerm && resp.granted {
		return true
	}

	if resp.term > s.raft.CurrentTerm {
		s.raft.CurrentTerm = resp.term
	}

	return false
}

func (s *Server) handleAppendRequest(request *AppendRequest) *AppendResponse {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	lastLogTerm, lastLogIndex := s.log.getLast()
	commitIndex := s.raft.CommitIndex
	if request.term < s.raft.CurrentTerm {
		s.logger.Info.Printf("[%s][handleAppendRequest]  response = false, term = %d, requset.term = %d\n", s.name, s.raft.CurrentTerm, request.term)
		return &AppendResponse{s.name, s.raft.CurrentTerm, false, lastLogTerm, lastLogIndex, commitIndex}
	}

	if request.term > s.raft.CurrentTerm {
		s.raft.CurrentTerm = request.term
		s.raft.LeaderId = request.leaderId
		s.raft.VotedFor = ""
		s.state = Follower
	} else {
		if s.state == Candidate {
			s.raft.LeaderId = request.leaderId
			s.raft.VotedFor = ""
			s.state = Follower
		} else if s.state == Leader {
			s.logger.Error.Printf("[%s][handleAppendRequest] another leader with same term[%d] was found!", s.name, request.term)
		}
	}

	if request.prevLogIndex == lastLogIndex && request.prevLogTerm == lastLogTerm {
		if request.leaderCommit > 0 {
			s.commit(request.leaderCommit)
		}
		s.logger.Info.Printf("[%s][handleAppendRequest] response = true, match = true, term = %d, lastLog = [%d,%d], commit = %d\n", s.name, s.raft.CurrentTerm, lastLogTerm, lastLogIndex, request.leaderCommit)
		return &AppendResponse{s.name, s.raft.CurrentTerm, true, lastLogTerm, lastLogIndex, request.leaderCommit}
	}

	result := s.log.truncate(request.prevLogTerm, request.prevLogIndex)
	lastLogTerm, lastLogIndex = s.log.getLast()

	if !result {
		s.logger.Info.Printf("[%s][handleAppendRequest] response = false, truncate = false, term = %d, lastLog =[%d,%d]\n", s.name, s.raft.CurrentTerm, lastLogTerm, lastLogIndex)
		return &AppendResponse{s.name, s.raft.CurrentTerm, false, lastLogTerm, lastLogIndex, commitIndex}
	}
	if len(request.entries) > 0 {
		s.log.appendList(request.entries)
		lastLogTerm, lastLogIndex = s.log.getLast()
	}

	if request.leaderCommit > s.getCommitIndex() {
		//	s.log.commit(min(request.leaderCommit, lastLogIndex))
		toCommitIndex := min(request.leaderCommit, lastLogIndex)
		if toCommitIndex > 0 {
			s.commit(toCommitIndex)
		}
	}
	commitIndex = s.raft.CommitIndex
	s.logger.Info.Printf("[%s][handleAppendRequest] response = true, term = %d, lastLog = [%d,%d], append_entries = %d, commitIndex = %d\n", s.name, s.raft.CurrentTerm, lastLogTerm, lastLogIndex, len(request.entries), commitIndex)
	return &AppendResponse{s.name, s.raft.CurrentTerm, true, lastLogTerm, lastLogIndex, commitIndex}
}
