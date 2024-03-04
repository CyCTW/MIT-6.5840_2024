# Lab3


## Lab3-A

- Consider sync mechanism
- Concurrent goroutines
  - AppendEntries RPC
  - RequestVotes RPC
  - ElectionTimeout goroutine(**Follower**)
  - Heartbeat goroutine(Leader)
  
### Implementation
- Ticker
  - If role is Follower
  - Check if election timeout
    - If timeout, become candidate
    - Increment term
    - vote for itself
    - reset timer
    - Send RPC to other for voting
    - Use another go routiine and channel to wait for other responses.
    - 
- SendHeartbeat
  - If role is Leader, send heartbeat to other peers
    - Use goroutine to send heartbeat to other async.
  - sleep 150 milli seconds
  - 