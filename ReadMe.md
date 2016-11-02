
The raft implementation by golang. 
The RPC is based on channel.


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