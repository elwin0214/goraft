package raft

func (s *Server) broadVote() chan *VoteResponse {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	responseChan := make(chan *VoteResponse, len(s.peers))
	s.raft.CurrentTerm++
	term := s.raft.CurrentTerm
	s.raft.VotedFor = s.name
	lastLogTerm, lastLogIndex := s.log.getLast()
	self := s.name
	for peer, _ := range s.peers {
		if self == peer {
			continue
		}
		go func(to string) {
			req := &VoteRequest{term, s.name, lastLogTerm, lastLogIndex}
			s.logger.Debug.Printf("[%s][broadVote] peer = %s, term = %d\n", s.name, to, term)
			resp := s.trans.sendVote(s.name, to, req)
			responseChan <- resp
		}(peer)
	}
	return responseChan
}

func (s *Server) candidateLoop() {

	vote := true
	votedNumber := 0
	timeoutChan := afterBetween(DefaultElectionTimeout, DefaultElectionTimeout*2)
	var responseChan chan *VoteResponse
	for s.GetState() == Candidate {

		if vote {
			votedNumber++
			responseChan = s.broadVote()
			vote = false
		}
		select {
		case <-s.stopChan:
			s.logger.Info.Printf("[%s][candidateLoop] go to stop\n", s.name)
			s.SetState(Stopped)
			return

		case resp := <-responseChan:
			succ := s.processVoteResponse(resp)
			s.logger.Debug.Printf("[%s][candidateLoop] vote term = %d, vote result = %t\n", s.name, resp.term, succ)
			if succ {
				votedNumber++
			}
			if votedNumber >= s.quorumSize() {
				s.logger.Info.Printf("[%s][candidateLoop] term = %d, votedNumber = %d\n", s.name, s.raft.CurrentTerm, votedNumber)
				s.SetState(Leader)
				s.observeChan <- &ElectMessage{Server: s.name, Term: s.raft.CurrentTerm}
				return
			}
		case msg := <-s.voteChan:
			req := msg.request.(*VoteRequest)
			resp := s.handleVoteRequest(req)
			msg.responseChan <- resp

		case msg := <-s.appendChan:
			req := msg.request.(*AppendRequest)
			resp := s.handleAppendRequest(req)
			msg.responseChan <- resp

		case <-timeoutChan:
			s.logger.Info.Printf("[%s][candidateLoop] go to vote again \n", s.name)
			s.SetState(Candidate)
			vote = true
			votedNumber = 0
			return
		}
	}
}
