package raft

import (
	"log"
	"sync"
)

type LogEntry struct {
	index   uint64
	term    uint64
	command string
}

func (entry *LogEntry) clone() *LogEntry {
	return &LogEntry{entry.index, entry.term, entry.command}
}

type LogStore struct {
	mutex   sync.Mutex
	entries []*LogEntry
	//commitIndex uint64
	//applyIndex  uint64
}

func NewLogStroe() *LogStore {
	return &LogStore{entries: make([]*LogEntry, 0, 1024)}
}

func (l *LogStore) getLast() (uint64, uint64) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if len(l.entries) == 0 {
		return 0, 0
	}
	entry := l.entries[len(l.entries)-1]
	return entry.term, entry.index
}

func (l *LogStore) get(from uint64) []*LogEntry {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if len(l.entries) == 0 || uint64(len(l.entries)) < from {
		return nil
	}
	log.Printf("[DEBUG] entries = %v from = %d\n", l.entries, from)
	return l.entries[from-1:]
}

func (l *LogStore) append(entry *LogEntry) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.entries = append(l.entries, entry)
}

func (l *LogStore) appendList(entries []*LogEntry) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	for i := 0; i < len(entries); i++ {
		l.entries = append(l.entries, entries[i].clone())
	}
}

func (l *LogStore) commit(commitIndex uint64) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
}

func (l *LogStore) apply(applyIndex uint64) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
}

// func (l *LogStore) getApplyIndex() uint64 {
// 	l.mutex.Lock()
// 	defer l.mutex.Unlock()
// 	return l.applyIndex
// }
// func (l *LogStore) getCommitIndex() uint64 {
// 	l.mutex.Lock()
// 	defer l.mutex.Unlock()
// 	return l.commitIndex
// }

func (l *LogStore) truncate(term uint64, index uint64) bool {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if len(l.entries) == 0 {
		return true
	}

	entry := l.entries[len(l.entries)-1]
	if entry.term == term && entry.index == index {
		return true
	}

	if index > entry.index {
		return false
	}

	pos := -1
	for ; index >= 1; index-- {
		if l.entries[index-1].term == term {
			pos = int(index - 1)
			break
		}
	}
	l.entries = l.entries[:pos+1]
	return true
}
