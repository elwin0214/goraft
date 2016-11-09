package raft

import (
	"encoding/gob"
	"strconv"
	"testing"
)
import "net/http"
import _ "net/http/pprof"
import "log"

func Test_KV(t *testing.T) {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	peers := []string{"server1", "server2", "server3"}

	gob.Register(Command{})
	c := newCluster(peers)

	server1 := c.Get("server1")
	server2 := c.Get("server2")
	server3 := c.Get("server2")

	setKVStore(server1)
	setKVStore(server2)
	setKVStore(server3)

	c.Start()
	c.waitElected()
	leader := c.GetLeader()
	count := 10
	for i := 1; i < count; i++ {
		key := strconv.Itoa(i)
		cmd := &Command{"set", key, key}
		r := leader.sendCommand(cmd)
		t.Log(cmd)
		t.Log(r)
		leader.logger.Debug.Println(cmd)
	}
	leader.Stop()
	c.waitElected()
	leader = c.GetLeader()

	for i := 1; i < count; i++ {
		key := strconv.Itoa(i)
		cmd := &Command{"get", key, ""}
		r := leader.sendCommand(cmd)
		t.Log(cmd)
		t.Log(r)
		if cmd.Key != cmd.Value {
			t.Errorf("%s != %s dont match!", cmd.Key, cmd.Value)
		}
		leader.logger.Debug.Println(cmd)
	}

	c.Stop()
}
