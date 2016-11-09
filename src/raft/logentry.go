package raft

type LogEntry struct {
	Index uint64
	Term  uint64
	Cmd   interface{}
}

// all operations in LogStore are not thread-safe
type LogStore struct {
	entries   []LogEntry
	logger    Logger
	persister *Persister
}

func newLogStroe() *LogStore {
	l := &LogStore{entries: make([]LogEntry, 0, 1024)}
	l.logger.Init()
	return l
}

func (l *LogStore) setPersister(persister *Persister) {
	l.persister = persister
}

func (l *LogStore) persist() {
	l.persister.writeLog(l.entries)
}

func (l *LogStore) recover() {
	l.entries = l.persister.readLog()
}

func (l *LogStore) getLast() (uint64, uint64) {
	if len(l.entries) == 0 {
		return 0, 0
	}
	entry := l.entries[len(l.entries)-1]
	return entry.Term, entry.Index
}

func (l *LogStore) getEntry(index uint64) *LogEntry {

	if len(l.entries) == 0 || uint64(len(l.entries)) < index || index <= 0 {
		return nil
	}
	le := l.entries[index-1]
	return &le
}

func (l *LogStore) get(from uint64) []LogEntry {
	if len(l.entries) == 0 || uint64(len(l.entries)) < from {
		return nil
	}
	if from <= 0 {
		from = 1
	}
	l.logger.Trace.Printf("[get] entries = %v from = %d\n", l.entries, from)
	entries := l.entries[from-1:]
	result := make([]LogEntry, 0, len(entries))
	for _, le := range entries {
		result = append(result, le)
	}
	return result
}

func (l *LogStore) getRange(from uint64, to uint64) []LogEntry {
	if to < from {
		return nil
	}
	if from <= 0 {
		return nil
	}

	return l.entries[from-1 : to]
}

func (l *LogStore) append(entry LogEntry) {
	l.entries = append(l.entries, entry)
}

func (l *LogStore) appendList(entries []LogEntry) {
	for i := 0; i < len(entries); i++ {
		l.entries = append(l.entries, entries[i])
	}
}

func (l *LogStore) appendFromMatched(term uint64, index uint64, entries []LogEntry) (bool, int) {
	if index < 0 {
		index = 0
	}
	if index == 0 {
		l.entries = l.entries[0:0]
		if len(entries) > 0 {
			l.appendList(entries)
		}
		return true, len(entries)
	}

	if len(l.entries) == 0 {
		if index > 1 {
			return false, 0
		}
		if len(entries) > 0 {
			l.appendList(entries)
		}
		return true, len(entries)
	}

	entry := l.entries[len(l.entries)-1]
	if entry.Term == term && entry.Index == index {
		if len(entries) > 0 {
			l.appendList(entries)
		}
		return true, len(entries)
	}
	if index > entry.Index {
		return false, 0
	}

	pos := -1
	for ; index >= 1; index-- {
		if l.entries[index-1].Term == term {
			pos = int(index - 1)
			break
		}
	}
	l.entries = l.entries[:pos+1]
	if len(entries) > 0 {
		l.appendList(entries)
	}
	return true, len(entries)
}
