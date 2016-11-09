package raft

import (
	"bytes"
	"encoding/gob"
	"log"
)

type Persister struct {
	raft []byte // represent raft state
	log  []byte //  log entries
}

func (p *Persister) readLog() []LogEntry {
	if len(p.log) == 0 {
		return make([]LogEntry, 0, 1024)
	}
	buf := bytes.NewBuffer(p.log)
	var le []LogEntry
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&le)
	if err != nil {
		log.Println(err)
		panic("fail to decode logentries")
	}
	return le
}

func (p *Persister) writeLog(logs []LogEntry) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(logs)
	if nil != err {
		log.Println(err)
		panic("fail to encode logentries")
	}
	p.log = buf.Bytes()
}

func (p *Persister) readRaft() RaftState {
	if len(p.raft) == 0 {
		return newRaftState()
	}
	buf := bytes.NewBuffer(p.raft)
	var rf RaftState
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&rf)
	if err != nil {
		log.Println(err)
		panic("fail to decode raft")
	}
	return rf
}

func (p *Persister) writeRaft(rf RaftState) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(&rf)
	if nil != err {
		log.Println(err)
		panic("fail to encode raft")
	}
	p.raft = buf.Bytes()
}
