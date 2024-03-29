package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.5840/labgob"
	"6.5840/labrpc"
)

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

const (
	Follower = iota
	Candidate
	Leader
)

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	currentTerm int
	votedFor    int // candidate Id that voted for, may be nil(-1). When election end(find new leader), reset it.

	commitIndex         int // last log index that is committed
	lastApplied         int // last log index that applied to state machine
	role                int
	last_heartbeat_time time.Time
	timeout             time.Duration
	// electionTimer     *ResettableTicker
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	// Your code here (2A).
	var currentTerm int
	var role int
	rf.Sync(func() {
		currentTerm = rf.currentTerm
		role = rf.role
	})

	return currentTerm, role == Leader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// raftstate := w.Bytes()
	// rf.persister.Save(raftstate, nil)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (2A, 2B).

	Term         int // candidate's term
	CandidateId  int // candidate requesting vote
	LastLogIndex int
	LastLogTerm  int
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool
}

const (
	null_candidate = -1
)

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).

	// 1. Check valid
	rf.Sync(func() {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		DPrintf("%d receive requestvote from %d", rf.me, args.CandidateId)

		if args.Term < rf.currentTerm {
			return
		}

		// If term is larger, reset term and become follower
		rf.UpdateTerm(args.Term)
		reply.Term = rf.currentTerm // Use term that may be updated

		if rf.votedFor == null_candidate || rf.votedFor == args.CandidateId {
			// if args.LastLogTerm >= rf.currentTerm && args.LastLogIndex >= rf.commitIndex {
			// Voted for candidate(Sender)
			rf.votedFor = args.CandidateId
			rf.resetElectionTimer()

			reply.VoteGranted = true
			// }
		}
	})
	return
}

type AppendEntriesArgs struct {
	Term     int
	LeaderId int
}
type AppendEntriesReply struct {
	Term    int
	Success bool
}

// * AppendEntries RPC Handler
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	// updated heartbeat
	rf.Sync(func() {
		DPrintf("%d receive heartbeat term %d from %d", rf.me, args.Term, args.LeaderId)
		rf.resetElectionTimer()
		// TODO: If receive new leader's appendEntries, become follower
		if rf.role == Candidate {
			rf.UpdateCandidateTerm(args.Term)
		} else if rf.role == Follower {
			rf.UpdateTerm(args.Term)
		}
		reply.Term = rf.currentTerm
	})
}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	DPrintf("%d Send request vote to %d", rf.me, server)
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) sendHeartBeatsRPC(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	// send heartbeats, only leader to this.
	DPrintf("%d Send heartbeat to %d", rf.me, server)
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).

	return index, term, isLeader
}

func (rf *Raft) Sync(cb func()) {
	rf.mu.Lock()
	cb()
	defer rf.mu.Unlock()
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

const (
	election_timeout = 10
)

func (rf *Raft) isElectionTimeOut() bool {
	return time.Now().Sub(rf.last_heartbeat_time) >= rf.timeout
}

func (rf *Raft) hasPassed() time.Duration {
	return time.Now().Sub(rf.last_heartbeat_time)
}

func (rf *Raft) resetElectionTimer() {
	rf.last_heartbeat_time = time.Now()
}

func (rf *Raft) UpdateTerm(term int) {
	// ensure currentTerm and role integrity.

	if term > rf.currentTerm {
		DPrintf("%d Update term to %d", rf.me, rf.currentTerm)
		rf.role = Follower
		rf.votedFor = null_candidate
		rf.currentTerm = term
	}

}

func (rf *Raft) UpdateCandidateTerm(term int) {
	if term >= rf.currentTerm {
		DPrintf("%d Candidate Update term to %d", rf.me, rf.currentTerm)

		rf.role = Follower
		rf.votedFor = null_candidate
		rf.currentTerm = term
	}
}

func (rf *Raft) ticker() {
	/*
		Conditions that reset timer
		1. Get Heartbeat from leader
		- If getting when start election, old leader will have smaller term.
		2. Start election(after timeout)
		3. Grant vote(before self-voted)

		How to reset timer?
		1. goroutine
		2. When there's a timer running(sleeping), should stop it?
		3.
	*/

	DPrintf("%d Start ticker", rf.me)
	for rf.killed() == false {
		// Perform the action
		// Your code here (2A)
		// Check if a leader election should be started.
		rf.Sync(func() {

			if rf.role != Leader && rf.isElectionTimeOut() {
				// start election, becoming candidate.
				DPrintf("%d Found timeout, become candidate.\n", rf.me)

				rf.role = Candidate

				// 1. Increment term
				rf.currentTerm += 1
				// 2. vote for self.
				rf.votedFor = rf.me
				// 3. reset timer
				rf.resetElectionTimer()

				// 4. Send RPC requests
				// TODO: Send rpc async
				votes := make(chan bool)
				currentTerm := rf.currentTerm

				for i := range rf.peers {
					if i == rf.me {
						continue
					}

					i := i
					go func() {

						args := RequestVoteArgs{
							currentTerm,
							rf.me,
							rf.commitIndex,
							currentTerm,
						}
						reply := RequestVoteReply{}
						ok := rf.sendRequestVote(i, &args, &reply)

						if ok {

							// gather replys
							rf.Sync(func() {
								rf.UpdateTerm(reply.Term)
							})

							if reply.VoteGranted {
								DPrintf("%d Recv vote reply from %d. be voted", rf.me, i)
								votes <- true
							} else {
								DPrintf("%d Recv vote reply from %d. vote fail", rf.me, i)
								votes <- false
							}
						} else {
							votes <- false
							// retry?
							DPrintf("%d Recv vote error. retry", rf.me)
						}
						// }
					}()
				}

				// Gather replys
				majority := (1 + len(rf.peers)) / 2

				go func() {
					winVotes := 1 // vote for itself already
					cnt := 1

					has_enter := false
					for {
						res := <-votes
						cnt++
						if res {
							winVotes++
						}

						if !has_enter && winVotes == majority {

							// Win election, become leader
							rf.Sync(func() {
								rf.role = Leader
							})

							// try start sending heartbeats goroutine
							rf.sendHeartBeats()
							has_enter = true
							// Don't break here to gracefully shutdown
						}

						if cnt == len(rf.peers) {
							// TODO: Consider case that rpc has no response?
							break
						}
					}
				}()
			}

			if rf.role == Leader {
				rf.resetElectionTimer()
			}

		})

		// Start Pause
		var passed time.Duration
		rf.Sync(func() {
			passed = rf.hasPassed()
		})
		// pause for a random amount of time between 450 and 600
		// milliseconds.
		// ms := 50 + (rand.Int63() % 300)
		ms := 450 + (rand.Int63() % 150)
		timeout := time.Duration(ms)*time.Millisecond - passed

		rf.timeout = timeout
		DPrintf("%d Sleep %d miliseconds?", rf.me, timeout.Milliseconds())
		time.Sleep(timeout)
	}
}

func (rf *Raft) sendHeartBeats() {
	for i := range rf.peers {
		if i == rf.me {
			continue
		}
		reply := AppendEntriesReply{}
		go func(i int) {
			var term int
			rf.Sync(func() {
				term = rf.currentTerm
			})
			rf.sendHeartBeatsRPC(i, &AppendEntriesArgs{term, rf.me}, &reply)
			rf.Sync(func() {
				rf.UpdateTerm(reply.Term)
			})
		}(i)
	}
}
func (rf *Raft) startSendHeartBeats() {
	// send heartbeat periodically if it's leader.
	// one sec at most 10 times
	DPrintf("%d Start send heartbeat goroutine", rf.me)

	for rf.killed() == false {
		var role int
		rf.Sync(func() {
			role = rf.role
		})
		if role != Leader {
			continue
		}
		rf.sendHeartBeats()

		DPrintf("%d Start send heartbeats\n", rf.me)

		time.Sleep(150 * time.Millisecond)
	}
	DPrintf("%d Exit send heartbeat goroutine", rf.me)

}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	rf.currentTerm = 0
	rf.votedFor = null_candidate

	rf.commitIndex = 0
	rf.lastApplied = 0
	rf.role = Follower
	rf.last_heartbeat_time = time.Now()
	rf.timeout = time.Hour // Set timeout to a very large value initially

	// Your initialization code here (2A, 2B, 2C).

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()
	go rf.startSendHeartBeats()

	return rf
}
