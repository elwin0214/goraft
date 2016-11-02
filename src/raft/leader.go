package raft

import (
	"log"
	"sort"
	"time"
)

func (s *Server) leaderLoop() {
	ticker := time.Tick(DefaultHeartbeatInterval)
	for s.GetState() == Leader {
		log.Printf("[DEBUG][%s][leaderLoop] goto a loop", s.name)
		select {
		case <-s.stopChan:
			s.SetState(Stopped)
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
			//log.Printf("[DEBUG][%s][leaderLoop] response = %v\n", resp)
			s.handleAppendResponse(resp)
		case <-ticker:
			//log.Printf("[DEBUG][%s][leaderLoop] begin to heartBeat", s.name)
			s.heartBeat()
			//log.Printf("[DEBUG][%s][leaderLoop] end to heartBeat", s.name)

		}

	}
}

func (s *Server) handleAppendResponse(response *AppendResponse) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	p := response.peer
	if response.term > s.raft.CurrentTerm {
		s.raft.CurrentTerm = response.term
	}

	if response.success {
		s.peers[p].matchIndex = response.lastLogIndex
		s.peers[p].nextIndex = response.lastLogIndex + 1
		log.Printf("[INFO][%s][handleAppendResponse] append = true, peer = %s, matchIndex = %d, nextIndex = %d\n", s.name, p, s.peers[p].matchIndex, s.peers[p].nextIndex)

		indexs := make(uint64Slice, 0, len(s.peers))

		for p, peer := range s.peers {
			if s.name == p {
				continue
			}
			indexs = append(indexs, peer.matchIndex)
		}
		indexs = append(indexs, s.raft.CommitIndex)
		sort.Sort(indexs)

		commitIndex := indexs[s.quorumSize()-1]
		log.Println(indexs)
		if commitIndex > s.raft.CommitIndex {
			s.Commit(commitIndex)
		}
		log.Printf("[INFO][%s][handleAppendResponse] commit = %d\n", s.name, commitIndex)
		return true
	} else {
		s.peers[p].nextIndex--
		s.peers[p].matchIndex = 0
		log.Printf("[INFO][%s][handleAppendResponse] append = false, peer = %s, matchIndex = %d, nextIndex = %d\n", s.name, p, s.peers[p].matchIndex, s.peers[p].nextIndex)
		s.sync(p)
		return false
	}
}

func (s *Server) sendCommand(command string) {

	if s.GetState() != Leader {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// todo
	_, lastLogIndex := s.log.getLast()
	entry := &LogEntry{lastLogIndex + 1, s.raft.CurrentTerm, command}
	s.log.append(entry)

	for to, _ := range s.peers {
		if to == s.name {
			continue
		}
		s.sync(to)
	}
}

func (s *Server) getAppendRequest(to string) *AppendRequest {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	lastLogTerm, lastLogIndex := s.log.getLast()
	entries := s.log.get(s.peers[to].nextIndex)
	return &AppendRequest{s.raft.CurrentTerm, s.raft.LeaderId, lastLogIndex, lastLogTerm, entries, s.raft.CommitIndex}
}

func (s *Server) sync(to string) {
	log.Printf("[INFO][%s][sync], peer = %s\n", s.name, to)

	go func(from string, to string) {
		request := s.getAppendRequest(to)
		response := s.trans.sendAppend(from, to, request)
		s.handleAppendResponse(response)
	}(s.name, to)
}

func (s *Server) heartBeat() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	//leaderCommit := s.raft.CommitIndex
	for to, _ := range s.peers {
		if to == s.name {
			continue
		}
		s.sync(to)
	}
}
