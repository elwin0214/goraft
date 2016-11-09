package raft

import (
	"encoding/gob"
	"log"
	"testing"
)

func Test_PersistRaftState(t *testing.T) {
	p := new(Persister)
	rf := RaftState{"server1", 2, 2, 1}
	p.writeRaft(rf)
	rf2 := p.readRaft()
	if rf2 != rf {
		t.Fatalf("fail in write&read RaftState\n")
	}
}

func Test_PersistLogEntry(t *testing.T) {
	gob.Register(Command{})
	p := new(Persister)
	entries := make([]LogEntry, 0, 16)

	for i := 1; i < 9; i++ {
		entries = append(entries, LogEntry{uint64(i), uint64(i), Command{"set", string(i + '1' - 1), string(i + '1' - 1)}})
	}

	p.writeLog(entries)
	entries2 := p.readLog()

	for j := 1; j < 9; j++ {
		log.Println(entries[j-1])
		log.Println(entries2[j-1])
	}
}
