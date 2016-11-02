package raft

type LogEntry struct {
	index   uint64
	term    uint64
	command string
}

func (entry *LogEntry) clone() *LogEntry {
	return &LogEntry{entry.index, entry.term, entry.command}
}

// all operations in LogStore are not thread-safe
type LogStore struct {
	entries []*LogEntry
	logger  Logger
}

func NewLogStroe() *LogStore {
	l := &LogStore{entries: make([]*LogEntry, 0, 1024)}
	l.logger.Init()
	return l
}

func (l *LogStore) getLast() (uint64, uint64) {
	if len(l.entries) == 0 {
		return 0, 0
	}
	entry := l.entries[len(l.entries)-1]
	return entry.term, entry.index
}

func (l *LogStore) getEntry(index uint64) *LogEntry {

	if len(l.entries) == 0 || uint64(len(l.entries)) < index || index <= 0 {
		return nil
	}
	return l.entries[index-1]
}

func (l *LogStore) get(from uint64) []*LogEntry {
	if len(l.entries) == 0 || uint64(len(l.entries)) < from {
		return nil
	}
	if from <= 0 {
		from = 1
	}
	l.logger.Trace.Printf("[get] entries = %v from = %d\n", l.entries, from)
	return l.entries[from-1:]
}

func (l *LogStore) append(entry *LogEntry) {
	l.entries = append(l.entries, entry)
}

func (l *LogStore) appendList(entries []*LogEntry) {
	for i := 0; i < len(entries); i++ {
		l.entries = append(l.entries, entries[i].clone())
	}
}

func (l *LogStore) commit(commitIndex uint64) {

}

func (l *LogStore) apply(applyIndex uint64) {

}

func (l *LogStore) truncate(term uint64, index uint64) bool {
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
