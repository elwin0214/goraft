package raft

import (
	"sort"
	"time"
)

func (s *Server) leaderLoop() {
	ticker := time.Tick(DefaultHeartbeatInterval)
	for true {
		role, stop := s.GetState()
		if stop || role != Leader {
			return
		}
		s.logger.Debug.Printf("[%s][leaderLoop] goto a loop", s.name)
		select {
		case <-s.stopChan:
			s.SetStop(true)
			return
		case msg := <-s.voteChan:
			req := msg.request.(*VoteRequest)
			resp := s.handleVoteRequest(req)
			msg.responseChan <- resp

		case msg := <-s.appendChan:
			req := msg.request.(*AppendRequest)
			resp := s.handleAppendRequest(req)
			msg.responseChan <- resp

		case resp := <-s.appendRespChan:
			s.handleAppendResponse(resp)
		case <-ticker:
			s.heartBeat()

		}

	}
}

func (s *Server) handleAppendResponse(response *AppendResponse) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.role != Leader {
		return false
	}
	if !response.accept {
		return false
	}
	p := response.peer
	if response.term > s.raft.CurrentTerm {
		s.raft.CurrentTerm = response.term
		s.role = Follower
		s.leaderId = ""
		s.persist()
		return false
	}

	if response.success {
		s.peers[p].matchIndex = response.lastLogIndex
		s.peers[p].nextIndex = response.lastLogIndex + 1
		s.logger.Info.Printf("[%s][handleAppendResponse] append = true, peer = %s, matchIndex = %d, nextIndex = %d\n", s.name, p, s.peers[p].matchIndex, s.peers[p].nextIndex)

		indexs := make(uint64Slice, 0, len(s.peers))

		for p, peer := range s.peers {
			if s.name == p {
				continue
			}
			indexs = append(indexs, peer.matchIndex)
		}
		_, lastLogIndex := s.log.getLast()
		indexs = append(indexs, lastLogIndex)
		sort.Sort(indexs)
		toCommitIndex := indexs[s.quorumSize()-1]
		s.logger.Debug.Println(indexs)
		entry := s.log.getEntry(toCommitIndex)
		s.logger.Debug.Printf("[%s][handleAppendResponse] toCommitIndex=%d\n", s.name, toCommitIndex)

		if toCommitIndex > s.raft.CommitIndex && (nil != entry && s.raft.CurrentTerm == entry.Term) {
			s.commit(toCommitIndex)
			s.logger.Info.Printf("[%s][handleAppendResponse] toCommitIndex = %d\n", s.name, toCommitIndex)
		}

		return true
	} else {
		if s.peers[p].nextIndex > 1 {
			s.peers[p].nextIndex--
		}
		s.peers[p].matchIndex = 0
		s.logger.Info.Printf("[%s][handleAppendResponse] append = false, peer = %s, matchIndex = %d, nextIndex = %d\n", s.name, p, s.peers[p].matchIndex, s.peers[p].nextIndex)
		s.sync(p)
		return false
	}
}

func (s *Server) sendCommand(command *Command) bool {

	if s.GetRole() != Leader {
		return false
	}
	s.mutex.Lock()
	_, lastLogIndex := s.log.getLast()

	entry := LogEntry{lastLogIndex + 1, s.raft.CurrentTerm, command}
	s.log.append(entry)
	for to, _ := range s.peers {
		if to == s.name {
			continue
		}
		s.sync(to)
	}
	responseChan := make(chan bool, 1)
	s.pendingRequest[entry.Index] = Request{command: command, reponseChan: responseChan}
	s.mutex.Unlock()
	s.logger.Trace.Printf("[%s][sendCommand] entry.index = %d\n", s.name, entry.Index)
	timer := time.After(DefaultSendTimeout)
	select {
	case <-timer:
		return false
	case result := <-responseChan:
		return result
	}
}

func (s *Server) getAppendRequest(to string) *AppendRequest {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	entries := s.log.get(s.peers[to].nextIndex)
	entry := s.log.getEntry(s.peers[to].nextIndex - 1)
	s.logger.Trace.Printf("[%s][getAppendRequest] commitIndex = %d, peer = %s, nextIndex = %d\n", s.name, s.raft.CommitIndex, to, s.peers[to].nextIndex)
	prevLogIndex := uint64(0)
	prevLogTerm := uint64(0)
	if nil != entry {
		prevLogIndex = entry.Index
		prevLogTerm = entry.Term
	}
	return &AppendRequest{s.raft.CurrentTerm, s.leaderId, prevLogIndex, prevLogTerm, entries, s.raft.CommitIndex}
}

func (s *Server) sync(to string) {
	s.logger.Trace.Printf("[%s][sync], peer = %s\n", s.name, to)
	go func(s *Server, from string, to string) {
		s.peers[to].mutex.Lock()
		defer s.peers[to].mutex.Unlock()
		if s.role != Leader {
			return
		}
		request := s.getAppendRequest(to)
		response := s.trans.sendAppend(from, to, request)
		if response.accept {
			s.handleAppendResponse(response)
		}
	}(s, s.name, to)
}

func (s *Server) heartBeat() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.logger.Debug.Printf("[%s][heartBeat] ", s.name)
	for to, _ := range s.peers {
		if to == s.name {
			continue
		}
		s.sync(to)
	}
}
