package raft

import (
	"encoding/gob"
	"testing"
)

func Test_AppendSync(t *testing.T) {
	peers := []string{"server1", "server2", "server3"}

	gob.Register(Command{})
	c := newCluster(peers)

	server1 := c.Get("server1")
	server1.log.append(LogEntry{1, 1, &Command{"set", "k", "1"}})
	server1.raft.CurrentTerm = 2
	server1.commit(1)
	server1.persist()

	server2 := c.Get("server2")
	server2.log.append(LogEntry{1, 1, &Command{"set", "k", "1"}})
	server2.raft.CurrentTerm = 2
	server2.commit(1)
	server2.persist()

	c.Start()

	c.waitElected()
	leader := c.GetLeader()
	r := leader.sendCommand(&Command{"set", "k", "2"})
	t.Log("execute cmd ")
	t.Log(r)
	if r && leader.getCommitIndex() != 2 {
		t.Errorf("leader[%s] commitIndex is %d \n", leader.name, leader.getCommitIndex())
	}
	c.Stop()
}
