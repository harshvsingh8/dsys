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
	"bytes"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.5840/labgob"
	"6.5840/labgob"
	"6.5840/labrpc"
)

// Define an enum for Peer Role
type PeerRole int

const (
	Follower PeerRole = iota
	Candidate
	Leader
)

func (d PeerRole) String() string {
	return [...]string{"Follower", "Candidate", "Leader"}[d]
}

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 3D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 3D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (3A, 3B, 3C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	role        PeerRole // current role of this server peer.
	currentTerm int      // current term number
	votedFor    int      // voted for the current term (-1 if not)
	heartBeats  int32    // to track if it received any heart-beat pulse during the last election timeout.
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	var term int
	var isleader bool

	// Your code here (3A).
	term = rf.currentTerm
	isleader = rf.role == Leader
	return term, isleader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
func (rf *Raft) persist() {
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	raftstate := w.Bytes()
	rf.persister.Save(raftstate, nil)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		rf.currentTerm = 0
		rf.role = Follower
		rf.votedFor = -1
		return
	}

	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var term int
	var votedFor int
	if d.Decode(&term) != nil ||
		d.Decode(&votedFor) != nil {
		DPrintf("Failed to read persisted properties, falling back to the default state.")
		rf.currentTerm = 0
		rf.votedFor = -1
	} else {
		rf.currentTerm = term
		rf.votedFor = votedFor
	}
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (3D).

}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (3A, 3B).
	Term        int
	CandidateId int
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (3A).
	Term        int
	VoteGranted bool
}

// RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (3A, 3B).
	DPrintf("Peer %d - received RequestVote from peer: %d, for term: %d", rf.me, args.CandidateId, args.Term)
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
	} else if args.Term == rf.currentTerm {
		if rf.votedFor == -1 {
			rf.votedFor = args.CandidateId
			reply.Term = args.Term
			reply.VoteGranted = true
		} else {
			reply.Term = args.Term
			reply.VoteGranted = false
		}
	} else {
		// my term is less - update my term, grant vote and change role if needed.
		rf.currentTerm = args.Term
		rf.votedFor = args.CandidateId
		reply.Term = args.Term
		reply.VoteGranted = true
		if rf.role == Leader {
			// TODO - Leader should bailout from the heartbeat sending loop.
			go rf.handleRoleChange(Follower)
		}
	}
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
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

type AppendEntriesArgs struct {
	Term     int
	LeaderId int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	// handle heartbeats for now.
	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.votedFor = -1
		reply.Term = args.Term
		reply.Success = true
		rf.incrementHeartBeats()
		if rf.role == Leader {
			// TODO - need a mechanism to centrally update the current peer role (streamline processing)
			rf.role = Follower
		}
	} else if args.Term == rf.currentTerm {
		reply.Term = args.Term
		reply.Success = true
		rf.incrementHeartBeats()
	} else {
		reply.Term = rf.currentTerm
		reply.Success = false
	}
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

	// Your code here (3B).
	return index, term, isLeader
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

func (rf *Raft) resetHeartBeats() {
	atomic.StoreInt32(&rf.heartBeats, 0)
}

func (rf *Raft) incrementHeartBeats() {
	atomic.AddInt32(&rf.heartBeats, 1)
}

func (rf *Raft) getHeartBeats() int32 {
	z := atomic.LoadInt32(&rf.heartBeats)
	return z
}

// Handles Peer role change - set up as per the new state
func (rf *Raft) handleRoleChange(role PeerRole) {
	// TODO - check if we need to wrap-up things for the previous role (ideally we should not)
	DPrintf("Peer %d, moved to role:%s, with term:%d, voted:%d", rf.me, role, rf.currentTerm, rf.votedFor)
	if role == Follower {
		rf.role = Follower
		go rf.electionTicker()
	} else if role == Candidate {
		rf.role = Candidate
		go rf.conductElection()
	} else if role == Leader {
		rf.role = Leader
		go rf.lead()
	}
}

func (rf *Raft) electionTicker() {
	for !rf.killed() {

		// Your code here (3A)
		// Check if a leader election should be started.
		DPrintf("Check for election in peer: %d", rf.me)

		rf.resetHeartBeats()

		// pause for a random amount of time between 50 and 350 milliseconds.
		ms := 50 + (rand.Int63() % 300)
		time.Sleep(time.Duration(ms) * time.Millisecond)

		// Check if the peer need to move from the follower role to candidate role.
		if rf.getHeartBeats() == 0 {
			DPrintf("Did not receive any heart-beat in peer: %d, moving to the candidate role", rf.me)
			break
		}
	}

	if !rf.killed() {
		go rf.handleRoleChange(Candidate)
	}
}

func (rf *Raft) conductElection() {
	// increment current term
	electionStartTime := time.Now()

	// increment self term
	rf.currentTerm = rf.currentTerm + 1
	rf.votedFor = -1

	var votes int32 = 0
	var rollbackTerm int32 = int32(rf.currentTerm)

	// vote self
	votes = 1
	rf.votedFor = rf.me

	for i := 0; i < len(rf.peers); i++ {
		go func(peerIndex int) {
			if peerIndex != rf.me {
				args := &RequestVoteArgs{}
				args.CandidateId = rf.me
				args.Term = rf.currentTerm
				reply := &RequestVoteReply{}
				rf.sendRequestVote(peerIndex, args, reply)
				if reply.VoteGranted {
					atomic.AddInt32(&votes, 1)
					votes = votes + 1
				}
				if reply.Term > rf.currentTerm {
					// update the current term, and indicate the flag to rollback to the Follower role.
					// TODO - check if we need to safe-guard the updates to rf fields.
					atomic.StoreInt32(&rollbackTerm, int32(reply.Term))
				}
			}
		}(i)
	}

	// check and handle for time-out, win, or role-change
	for {
		// exit the election process - if the current peer becomes a Follower
		updatedTerm := int(atomic.LoadInt32(&rollbackTerm))
		if updatedTerm != rf.currentTerm {
			DPrintf("Peer: %d, need to rollback as a Follower due to stale term", rf.me)
			rf.currentTerm = updatedTerm
			rf.votedFor = -1
			// TODO - check if we need to pass around term and voted-for with the role change
			go rf.handleRoleChange(Follower)
			break
		}

		voteCount := int(atomic.LoadInt32(&votes))
		if voteCount > len(rf.peers)/2 {
			DPrintf("Peer: %d, won the election for term: %d", rf.me, rf.currentTerm)
			go rf.handleRoleChange(Leader)
			break
		}

		// wait for some max timeout for results for a given term.
		electionResultsWaitTimeout := 300
		if int32(time.Since(electionStartTime).Milliseconds()) > int32(electionResultsWaitTimeout) {
			DPrintf("Peer: %d, need to restart the election process, as results awaiting timeouts.", rf.me)
			go rf.handleRoleChange(Candidate)
			break
		}

		ms := 10 // wait for 10 ms to check again.
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
}

func (rf *Raft) lead() {
	// Send heartbeat as long as its leader, else fallback.
	var rollbackTerm int32 = int32(rf.currentTerm)
	for {
		for i := 0; i < len(rf.peers); i++ {
			if i == rf.me {
				continue
			}
			go func(peer int) {
				args := AppendEntriesArgs{}
				reply := AppendEntriesReply{}
				args.LeaderId = rf.me
				args.Term = rf.currentTerm
				rf.sendAppendEntries(peer, &args, &reply)
				if reply.Term > rf.currentTerm {
					atomic.StoreInt32(&rollbackTerm, int32(reply.Term))
				}
			}(i)
		}

		ms := 10 // send heartbeat every 10 second
		updatedTerm := int(atomic.LoadInt32(&rollbackTerm))
		time.Sleep(time.Duration(ms) * time.Millisecond)
		if updatedTerm != rf.currentTerm {
			rf.currentTerm = updatedTerm
			rf.votedFor = -1
			go rf.handleRoleChange(Follower)
			break
		}
		// TODO - ensure proper state change handling
		if rf.role == Follower {
			go rf.handleRoleChange(Follower)
			break
		}
	}
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

	// Your initialization code here (3A, 3B, 3C).
	DPrintf("Making peer: %d", me)

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// initialize the peer in a follower role.
	rf.handleRoleChange(Follower)
	return rf
}
