package raft

import (
	"encoding/gob"
	"testing"
)

func Test_Election(t *testing.T) {
	peers := []string{"server1", "server2", "server3"}
	c := newCluster(peers)
	c.Start()
	msg := c.waitElected()
	t.Log(msg)
	c.Stop()
}

func Test_ElectionFirstBroadcast(t *testing.T) {
	peers := []string{"server1", "server2", "server3"}
	c := newCluster(peers)
	server1 := c.Get("server1")
	server3 := c.Get("server3")
	server1.testing = true
	server1.electionTimeout = int64(3 * DefaultElectionTimeout)
	server3.testing = true
	server3.electionTimeout = int64(3 * DefaultElectionTimeout)

	c.Start()
	msg := c.waitElected()
	if msg.Server != "server2" {
		t.Error("server2 is not leader!")
	}

	c.Stop()
}

func Test_ElectionHighTerm(t *testing.T) {
	peers := []string{"server1", "server2", "server3"}

	gob.Register(Command{})
	c := newCluster(peers)

	server1 := c.Get("server1")
	server1.testing = true
	server1.electionTimeout = int64(3 * DefaultElectionTimeout)
	server1.log.append(LogEntry{1, 1, &Command{"set", "k", "1"}})
	server1.raft.CurrentTerm = 2
	server1.commit(1)
	server1.persist()

	server2 := c.Get("server2")
	server2.testing = true
	server2.electionTimeout = int64(3 * DefaultElectionTimeout)
	server2.log.append(LogEntry{1, 1, &Command{"set", "k", "1"}})
	server2.raft.CurrentTerm = 2
	server2.commit(1)
	server2.persist()

	c.Start()
	msg := c.waitElected()
	if msg.Server != "server1" && msg.Server != "server2" {
		t.Error("server1 or server2 is not leader!")
	}
	c.Stop()
}
