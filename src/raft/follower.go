package raft

import (
	"log"
)

func (s *Server) followerLoop() {

	reset := false
	timeoutChan := afterBetween(DefaultElectionTimeout, DefaultElectionTimeout*2)

	for s.GetState() == Follower {
		if reset {
			log.Printf("[INFO][%s][followerLoop] reset timeout\n", s.name)
			timeoutChan = afterBetween(DefaultElectionTimeout, DefaultElectionTimeout*2)
		}

		select {
		case <-s.stopChan:
			s.SetState(Stopped)
			return
		case msg := <-s.voteChan:
			req := msg.request.(*VoteRequest)
			resp := s.handleVoteRequest(req)
			reset = resp.granted //todo
			msg.responseChan <- resp
		case msg := <-s.appendChan:
			req := msg.request.(*AppendRequest)
			resp := s.handleAppendRequest(req)
			reset = resp.success
			msg.responseChan <- resp
		case resp := <-s.appendRespChan:
			log.Printf("[INFO][%s][followerLoop] resp=%v, error=append response message\n", s.name, resp)
		case <-timeoutChan:
			log.Printf("[INFO][%s][followerLoop] convert to Candidate!\n", s.name)
			s.SetState(Candidate)
			return
		}
	}
}
