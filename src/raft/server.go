package raft

import (
	"sync"
	"time"
)

const (
	Follower  = "follower"
	Candidate = "candidate"
	Leader    = "leader"
)

const (
	DefaultHeartbeatInterval = 1000 * time.Millisecond
	DefaultElectionTimeout   = 3000 * time.Millisecond
	DefaultSendTimeout       = 10000 * time.Millisecond
)

type Peer struct {
	mutex      sync.Mutex //   prevent the goroute for log replication from executing concurrently
	nextIndex  uint64
	matchIndex uint64
}

type Store interface {
	apply(entry LogEntry)
	reply(entry LogEntry)
}

type Server struct {
	mutex sync.Mutex
	name  string

	voteChan       chan *Message        // receive  requestvote
	appendChan     chan *Message        // receive appendentity
	appendRespChan chan *AppendResponse // receive appendentity response

	stopChan     chan bool
	electionChan chan *ElectMessage // receive election message

	peers map[string]*Peer

	trans  Transporter
	config *Config

	// protected by mutex
	role      string
	leaderId  string
	raft      RaftState
	log       *LogStore
	persister *Persister
	stop      bool
	// for request
	pendingRequest map[uint64]Request
	store          Store

	// for testing
	testing         bool
	electionTimeout int64

	logger Logger
}

// func (p *Peer) sync(s *Server) {
// 	select {
// 	case <-p.syncChan:
// 		req := s.getAppendRequest(p.name)
// 		resp := s.trans.sendAppend(s.name, p.name, req)
// 		s.handleAppendResponse(resp)
// 	case <-p.stopChan:
// 		return
// 	}
// }

func newServer(name string, log *LogStore, config *Config, electionChan chan *ElectMessage) *Server {
	persister := &Persister{}
	server := &Server{name: name, log: log, persister: persister, role: Follower, leaderId: "", stop: true, testing: false}
	server.logger.Init()
	server.voteChan = make(chan *Message, 1024)
	server.appendChan = make(chan *Message, 1024)
	server.appendRespChan = make(chan *AppendResponse, 1024)

	server.stopChan = make(chan bool, 1)
	server.electionChan = electionChan
	server.peers = make(map[string]*Peer)

	server.config = config
	for _, p := range server.config.peers {
		server.addPeer(p)
	}
	server.raft = newRaftState()
	server.log.setPersister(persister)

	server.pendingRequest = make(map[uint64]Request, 1024)
	return server
}

func (s *Server) addPeer(p string) {
	peer := new(Peer)
	peer.nextIndex = 1
	peer.matchIndex = 0
	s.peers[p] = peer
}

func (s *Server) commit(index uint64) {
	if s.raft.CommitIndex >= index {
		return
	}
	s.raft.CommitIndex = index
	s.persist()
	//s.electionChan <- &CommitMessage{Server: s.name, Term: s.raft.CurrentTerm, CommitIndex: s.raft.CommitIndex}
	if s.raft.LastApplied >= index {
		return
	}
	beforeIndex := s.raft.LastApplied
	s.logger.Debug.Println(s.log.entries)
	s.logger.Debug.Printf("[%s][commit] beforeIndex = %d, index = %d\n", s.name, beforeIndex, index)
	entries := s.log.getRange(beforeIndex+1, index)
	if len(entries) == 0 {
		return
	}
	for _, entry := range entries {
		if nil != s.store {
			s.store.apply(entry)
			s.store.reply(entry)
		}
		if _, ok := s.pendingRequest[entry.Index]; ok {
			s.pendingRequest[entry.Index].reponseChan <- true
		}
	}
	s.raft.LastApplied = index
	s.persist()
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

func (s *Server) SetTransporter(trans Transporter) {
	s.trans = trans
}

func (s *Server) GetRole() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.role
}

func (s *Server) SetRole(role string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.role = role
	if role == Leader {
		s.leaderId = s.name
	}
}

func (s *Server) SetStop(stop bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.stop = stop
}

func (s *Server) GetState() (string, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.role, s.stop
}
func (s *Server) GetRaftState() RaftState {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.raft
}
func (s *Server) recover() {
	s.log.recover()
	s.raft = s.persister.readRaft()
}

func (s *Server) persist() {
	s.log.persist()
	s.persister.writeRaft(s.raft)
}

func (s *Server) Start() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if !s.stop {
		s.logger.Error.Printf("%s is started! %s ", s.name, s.role)
		return
	}
	s.stop = false
	s.recover()
	s.logger.Info.Printf("[%s][Start] role = %s", s.name, s.role)
	go s.loop()
}

func (s *Server) Stop() {
	s.stopChan <- true
}

func (s *Server) loop() {
	for true {
		role, stop := s.GetState()
		if stop {
			return
		}
		switch role {
		case Follower:
			s.logger.Debug.Printf("[%s][loop] Follower role = %s %s", s.name, s.role, role)
			s.followerLoop()
		case Candidate:
			s.logger.Debug.Printf("[%s][loop] Candidate role = %s", s.name, s.role)
			s.candidateLoop()
		case Leader:
			s.logger.Debug.Printf("[%s][loop] Leader role = %s", s.name, s.role)
			s.leaderLoop()
		default:
			s.logger.Error.Printf("[%s][loop] unknown role = %s", s.name, s.role)

		}
	}
}

func (s *Server) quorumSize() int {
	return (len(s.peers)/2 + 1)
}

func (s *Server) handleVoteRequest(request *VoteRequest) *VoteResponse {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.stop {
		return &VoteResponse{false, s.name, s.raft.CurrentTerm, false}
	}
	if request.term < s.raft.CurrentTerm {
		s.logger.Info.Printf("[%s][handleVoteRequest] current term = %d, reject term = %d\n", s.name, s.raft.CurrentTerm, request.term)
		return &VoteResponse{true, s.name, s.raft.CurrentTerm, false}
	}

	if request.term == s.raft.CurrentTerm {
		if "" != s.raft.VotedFor && request.candidateId != s.raft.VotedFor {
			return &VoteResponse{true, s.name, s.raft.CurrentTerm, false}
		}
	}
	toPersist := false
	if request.term > s.raft.CurrentTerm {
		s.raft.CurrentTerm = request.term
		s.raft.VotedFor = ""
		toPersist = true
	}

	lastLogTerm, lastLogIndex := s.log.getLast()
	if (lastLogTerm == request.lastLogTerm && lastLogIndex > request.lastLogIndex) || lastLogTerm > request.lastLogTerm {
		if toPersist {
			s.persist()
		}
		s.logger.Info.Printf("[%s][handleVoteRequest] current term = %d, last = [%d, %d], req.last = [%d, %d] vote = false \n", s.name, s.raft.CurrentTerm, lastLogTerm, lastLogIndex, request.lastLogTerm, request.lastLogIndex)
		return &VoteResponse{true, s.name, s.raft.CurrentTerm, false}
	}
	s.raft.VotedFor = request.candidateId
	s.persist()
	s.logger.Info.Printf("[%s][handleVoteRequest] current term = %d, vote for = %s\n", s.name, s.raft.CurrentTerm, request.candidateId)
	return &VoteResponse{true, s.name, s.raft.CurrentTerm, true}
}

func (s *Server) processVoteResponse(resp *VoteResponse) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if resp.term == s.raft.CurrentTerm && resp.granted {
		return true
	}

	if resp.term > s.raft.CurrentTerm {
		s.raft.CurrentTerm = resp.term
		s.persist()
	}

	return false
}

func (s *Server) handleAppendRequest(request *AppendRequest) *AppendResponse {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	lastLogTerm, lastLogIndex := s.log.getLast()
	commitIndex := s.raft.CommitIndex
	if s.stop {
		return &AppendResponse{false, s.name, s.raft.CurrentTerm, false, lastLogTerm, lastLogIndex, commitIndex}

	}
	if request.term < s.raft.CurrentTerm {
		s.logger.Warn.Printf("[%s][handleAppendRequest] response = false, term = %d, requset.term = %d\n", s.name, s.raft.CurrentTerm, request.term)
		return &AppendResponse{true, s.name, s.raft.CurrentTerm, false, lastLogTerm, lastLogIndex, commitIndex}
	}
	toPersist := false
	if request.term > s.raft.CurrentTerm {
		s.raft.CurrentTerm = request.term
		s.raft.VotedFor = ""
		toPersist = true
		s.leaderId = request.leaderId
		s.role = Follower

	} else {
		if s.role == Candidate {
			s.leaderId = request.leaderId
			s.raft.VotedFor = ""
			toPersist = true
			s.role = Follower
		} else if s.role == Leader {
			s.logger.Error.Printf("[%s][handleAppendRequest] it's impossible that another leader with same term[%d] was found!", s.name, request.term)

		}
	}

	s.logger.Trace.Printf("[%s][handleAppendRequest] prevLogTerm = %d, prevLogIndex = %d \n", s.name, request.prevLogTerm, request.prevLogIndex)
	s.logger.Trace.Println(request)

	result, size := s.log.appendFromMatched(request.prevLogTerm, request.prevLogIndex, request.entries)
	lastLogTerm, lastLogIndex = s.log.getLast()

	if !result {
		if toPersist {
			s.persist()
		}
		s.logger.Error.Printf("[%s][handleAppendRequest] response = false, truncate = false, term = %d, lastLog =[%d,%d]\n", s.name, s.raft.CurrentTerm, lastLogTerm, lastLogIndex)
		return &AppendResponse{true, s.name, s.raft.CurrentTerm, false, lastLogTerm, lastLogIndex, commitIndex}
	}

	if request.leaderCommit > s.getCommitIndex() {
		toCommitIndex := min(request.leaderCommit, lastLogIndex)
		if toCommitIndex > 0 {
			s.commit(toCommitIndex)
		}
	}
	if toPersist {
		s.persist()
	}
	commitIndex = s.raft.CommitIndex
	s.logger.Info.Printf("[%s][handleAppendRequest] response = true, term = %d, lastLog = [%d,%d], append_entries = %d, commitIndex = %d\n", s.name, s.raft.CurrentTerm, lastLogTerm, lastLogIndex, size, commitIndex)
	return &AppendResponse{true, s.name, s.raft.CurrentTerm, true, lastLogTerm, lastLogIndex, commitIndex}
}
