package raft

import (
	"time"
)

func (s *Server) followerLoop() {

	reset := false
	var timeoutChan <-chan time.Time
	if s.testing {
		timeoutChan = time.After(time.Duration(s.electionTimeout))
	} else {
		timeoutChan = afterBetween(DefaultElectionTimeout, DefaultElectionTimeout*2)
	}

	for true {
		role, stop := s.GetState()
		if stop || role != Follower {
			return
		}
		if reset {
			s.logger.Info.Printf("[%s][followerLoop] reset timeout\n", s.name)
			if s.testing {
				timeoutChan = time.After(time.Duration(s.electionTimeout))
			} else {
				timeoutChan = afterBetween(DefaultElectionTimeout, DefaultElectionTimeout*2)
			}
		}

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
			reset = resp.success
			msg.responseChan <- resp
		case resp := <-s.appendRespChan:
			s.logger.Error.Printf("[%s][followerLoop] resp=%v, error=append response message\n", s.name, resp)
		case <-timeoutChan:
			s.logger.Info.Printf("[%s][followerLoop] convert to Candidate!\n", s.name)
			s.SetRole(Candidate)
			return
		}
	}
}
