
test:
	go test -v -run Test_PersistRaftState raft
	go test -v -run Test_PersistLogEntry raft
	go test -v -run Test_AppendSync  raft
	go test -v -run Test_Election  raft
	go test -v -run Test_ElectionFirstBroadcast raft
	go test -v -run Test_KV raft
	
clean:
	rm -rf ./bin
	rm -rf ./pkg