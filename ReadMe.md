
The raft implementation by golang. 
>The RPC is based on channel.


## feature 

Main features in [In Search of an Understandable Consensus Algorithm](https://raft.github.io/raft.pdf)

* Election
* Log replication
* Consistency check


## key points

* Log Matching Property

>If two entries in different logs have the same index and term, then they store the same command.   
>If two entries in different logs have the same index and term, then the logs are identical in all preceding entries. 

* Election restriction(5.4.1)

>Raft uses the voting process to prevent a candidate from winning an election unless its log contains all committed entries.

* Committing entries from previous terms(5.4)

>Raft never commits log entries from previous terms by counting replicas. 

* Client interaction(8)

>If the leader crashes after committing the log entry but before responding to the client, the client will retry the command with a new leader, causing it to be executed a second time.The solution is for clients to assign unique serial numbers to every command. Then, the state machine tracks the latest serial number processed for each client, along with the as-sociated response. If it receives a command whose serial number has already been executed, it responds immediately without re-executing the request.

>To avoid returning stale data for Read-only operations
First each leader commit a blank no-op entry into the log at the start of its term.Second,a leader must check whether it has been de- posed before processing a read-only request

