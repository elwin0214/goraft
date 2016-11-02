package raft

import (
	"math/rand"
	"time"
)

func random(delta uint64) uint64 {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	return uint64(rand.Int63n(int64(delta)))
}

func f() int {
	return 1
}

func afterBetween(min time.Duration, max time.Duration) <-chan time.Time {

	d := min
	delta := max - min
	if delta > 0 {
		d += time.Duration(random(uint64(delta)))
	}

	return time.After(d)
}

func min(a uint64, b uint64) uint64 {
	if a > b {
		return a
	} else {
		return b
	}
}

type uint64Slice []uint64

func (p uint64Slice) Len() int           { return len(p) }
func (p uint64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p uint64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
