# 6.824 2021 Final Exam

This exam is 120 minutes. Please write down any assumptions you make.
You are allowed to consult 6.824 papers, notes, and lab code, but no
internet use other than for taking the exam and asking questions.
Please ask questions on Piazza in a private post or in the lecture
Zoom meeting in a chat to the staff. Please don't collaborate or
discuss the exam with anyone.

In particular, please do not discuss the contents of the exam with
anyone at all besides the course staff until at least 5:00 PM EDT on
Tuesday, May 25th.

## Lab3

Arthur Availability, Bea Byzantium, and Cameron Communication have
all implemented GET requests in their Lab3 KVServer code in different
ways, and none of them can agree which of their solutions is correct.

Please read over each of the three implementations they propose. For
each implementation, state whether it is correct, and briefly justify
why or why not. In particular, pay attention to whether GETs are
linearizable. You may assume that snapshotting is disabled.

For each of these implementations, PUT and APPEND are (correctly)
implemented in the same way:

    func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
        kv.mu.Lock()
        defer kv.mu.Unlock()

        newOp := Op{args.Op, args.Key, args.Value, args.ClientId, args.SequenceId}

        for !kv.killed() {
            index, term, ok := kv.rf.Start(newOp)
            if !ok {
                // not leader
                break
            }

            for kv.lastApplied < index && term == kv.rf.GetTerm() {
                kv.cond.Wait()
            }

            // was it our command that was executed?
            client := kv.Clients[args.ClientId]
            if client.SequenceId == args.SequenceId {
                reply.Ok = true
                return
            }
        }

        reply.Ok = false
    }

You may assume that 'kv.cond.Broadcast()' is called whenever a log
entry is received on the apply channel or when the term changes.

A. Arthur claims that any change made to the key/value store will
eventually reach the key/value store, as long as progress being made.
Therefore, he implements his GET RPC handler as follows:

    func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
        kv.mu.Lock()
        defer kv.mu.Unlock()

        term := kv.rf.GetTerm()
        index := kv.lastApplied + 1

        for !kv.killed() && kv.rf.IsLeader() && term == kv.rf.GetTerm() {
            if kv.lastApplied >= index {
                reply.Ok = true
                reply.Result = kv.keyvalue[args.Key]
                return
            }

            kv.cond.Wait()
        }

        reply.Ok = false
    }

Is his implementation of GET correct? Briefly justify why or why not.

|____|

B. Bea claims that Arthur's solution is wrong, because it doesn't
commit anything to the log. They argue that it's necessary to commit
a log entry once for every GET operation.

Here's how they implement their GET handler:

    func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
        newOp := Op{"Get", args.Key, "", args.ClientId, args.SequenceId}

        for !kv.killed() {
            index, term, ok := kv.rf.Start(newOp)
            if !ok {
                // not leader
                break
            }

            for kv.lastApplied < index && term == kv.rf.GetTerm() {
                kv.cond.Wait()
            }

            // was it our command that was executed?
            client := kv.Clients[args.ClientId]
            if client.SequenceId == args.SequenceId {
                reply.Ok = true
                reply.Result = kv.keyvalue[args.Key]
                return
            }
        }

        reply.Ok = false
    }

Is their implementation of GET correct? Briefly justify why or why not.

|____|

C. Cameron has a problem with Bea's design. She purports that their
design puts the linearization point in the wrong place; she counters
that it should be in the apply channel thread instead of the RPC
handler thread.

She modifies the apply channel thread to fix this problem:

    func (kv *KVServer) perform(index int, op Op) {
        /* ... other code ... */

        if op.Opcode == "Get" {
            kv.Clients[op.ClientId] = &ClientState{
                SequenceId: op.SequenceId,
                LastResult: kv.keyvalue[op.Key],
            }
        }

        /* ... other cases ... */
    }

And fixes the purported bug in Bea's implementation:

    func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
        newOp := Op{"Get", args.Key, "", args.ClientId, args.SequenceId}

        for !kv.killed() {
            index, term, ok := kv.rf.Start(newOp)
            if !ok {
                // not leader
                break
            }

            for kv.lastApplied < index && term == kv.rf.GetTerm() {
                kv.cond.Wait()
            }

            // was it our command that was executed?
            client := kv.Clients[args.ClientId]
            if client.SequenceId == args.SequenceId {
                reply.Ok = true
                reply.Result = client.LastResult
                return
            }
        }

        reply.Ok = false
    }

Is her implementation of GET correct? Briefly justify why or why not.

|____|

## Chain replication

Consider the paper "Chain replication for supporting high throughput
and availability" by van Renesse et al.  Chain replication provides
strong consistency while performing updates at the head of the chain
and performing reads at the tail of the chain.  How does chain
replication guarantee that a client will read the value of the
most-recently completed write if the client and the tail are in a
network partition by themselves, separate from the head?  (Briefly
explain your answer.)

|____|


## Frangipani

Consider the paper "Frangipani: a scalable distributed file system".
In Frangipani, a workstation returns a lock to the lock service after
it has written all its file system modifications to its log in Petal.
 
A. What would go wrong if Frangipani first released the lock and then
wrote to the log and updated the file system?  (Give a scenario and
explain your answer briefly.)

|____|


B. A workstation A may crash while holding a lock on file. How can
another workstation B obtain the lock for this file and observe the
updates that the workstation may have made? (Briefly explain your
answer.)

|____|


## Spanner

Consider the paper "Spanner: Google's Globally-Distributed Database"
and the following timeline with 3 transactions:


  r/w T0 @  1: Wx1 C



                   |1-----------5| |E1-----9|
  r/w T1:               Wx2 C          CW



                                    |E2-------------L2|
  r/o T2:                                    Rx

  
                          ------------- time ----->
  
T0 is a read/write transaction that has committed (C) at timestamp 1,
writing 1 to x.  T1 is a read/write transaction, writing 2 to x, and
which starts after T0 completes.  T2 is a read-only transaction,
reading x and starts after T1 completes.

Real-time goes from left to right. [E...L] indicates an interval with
Earliest and Latest returned by TrueTime.

When committing T1 at C, the coordinators calls TT.now() and it
returns [1,5], as shown above.

A. What commit timestamp will T1 choose? (Briefly explain your answer)

|____|

At CW T1 waits, repeatedly calling TT.now().

B. What must TT.now().earliest (E1) at least be for T1 to stop waiting?
(Briefly explain your answer)

|____|

When T2 reads x, it uses the value TT.now().latest for the timestamp
at which it reads x.

C. What is the lowest possible timestamp for reading that T2 could
receive?  (Briefly explain your answer)

|____|

## FaRM

Consider Figure 4 of the FaRM paper "No compromises: distributed
transactions with consistency, availability, and performance".

Zara Zoom simplifies the FaRM protocol by merging the validate phase
with the LOCK phase.  She modifies the LOCK phase to acquire also
locks for all the objects that a transaction *reads*.  She eliminates
the validate phase.

A. Is Zara's protocol still correct? (Briefly explain your answer)

|____|


B. What is the downside of Zara's change? (Briefly explain your answer
and be specific)

|____|


## Spark

Consider Figure 3 and the code below it from the Spark paper
"Resilient distributed datasets: a fault-tolerant abstraction for
in-memory cluster computing" by Zaharia et al.

Ben changes the line:
  
	val links = spark.textFile(...).map(...).persist()
  
to

	val links = spark.textFile(...).map(...)


A. Is Ben's program still correct? (Briefly explain your answer)

|____|


B. What is the downside of Ben's change? (Briefly explain your answer
and be specific)

|____|


## Memcache

Section 4.3 of "Scaling Memcache at Facebook" by Nishtala et
al. describes warming up a cold cluster and a two-second hold-off rule
for avoiding a race condition. The rule is that after a client deletes
a key in the cold cluster, no client can set the key in the cold
cluster for 2 seconds.

Consider a single region with one storage cluster and two front-end
clusters, one cold and one warm.

Cara Cache proposes a different plan than the two-second hold-off
rule, inspired by the "remote" marker trick.  When a client deletes a
key in the cold cluster, the client marks the key as "deleted". The
first client that reads that key receives a lease as usual and fetches
the data from the database, instead of from the warm cluster.  The
client then sets the key (if its lease still valid), removing the
"deleted" marker.

Does Cara's plan avoid the race discussed in section 4.3?  (Briefly
explain your answer)

|____|


## Blockstack

You are hired to write a decentralized version of a simplified piazza
application using Blockstack.  The application needs to support only
public and private questions and answers. It needs to support only 1
class (6.824) and you can assume that the application knows the name
of each student and staff member in 6.824.

Sketch out a design by answering the following questions, briefly
explaining your answer to each question.

A. When a user writes a question or a response where does the
application store this data?

|____|


B. How does the app ensure that private questions stay private?

|____|

C. How does the application collect all questions and answers to
display them to its user?

|____|

D. How can a user be sure they find the correct public key for another
user?

|____|

E. What is the role of piazza.com in this decentralized design?

|____|



## 6.824

A. Which papers/lectures should we omit in future 6.824 years,
because they are not useful or are too hard to understand?

[ ] Chain replication
[ ] Frangipani
[ ] Distributed transactions
[ ] Spanner
[ ] FaRM
[ ] Spark
[ ] Memcache
[ ] SUNDR
[ ] Bitcoin
[ ] Blockstack
[ ] AnalogicFS

B. Which papers/lectures did you find most useful?

[ ] Chain replication
[ ] Frangipani
[ ] Distributed transactions
[ ] Spanner
[ ] FaRM
[ ] Spark
[ ] Memcache
[ ] SUNDR
[ ] Bitcoin
[ ] Blockstack
[ ] AnalogicFS

C. What should we change about the course to make it better?

|____|

-----  answers -----

## Lab3

A. Arthur's solution is INCORRECT. Consider what happens if a leader
is partitioned into a minority of the network, and a new leader is
elected in the majority. The minority leader may continue answering
GET requests with stale data!

B. There are two acceptable answers for this question. 
(1) Bea's solution is CORRECT. The linearization point will be after 
the commit point for the GET operation, but that's fine; all that 
matters is that the linearization point is both after the GET is called 
and before the GET returns.

(2) The second answer points out that the lock must be acquired 
in the GET code provided in order for this solution to avoid races 
and be correct (e.g., calling cond.Wait() without holding the lock 
will panic).

C. There are two acceptable answers.
(1) Cameron's solution is CORRECT (assuming correct locking in the code). 
The linearization point will be the commit point for the GET operation, 
which is both after the GET is called and before the GET returns. 

(2) As in part (B), the lock should be acquired in the GET code to avoid races
and panics.

## Chain replication

One correct explanation is to point out that once the tail is
partitioned, CR will not apply updates, since updates must be
replicated down the complete chain, but this will not succeed since
the tail is in a separate partition.  So any further updates will be
stalled until the partition heals or the master changes the
configuration to remove the tail. Until then it is safe for the tail
to serve queries---it has the up-to-date state, and the state isn't
changing.

Once the master changes the configuration and ejects the tail, all
clients and other nodes must learn of this change. The paper doesn't
spell out exactly how, but either this involves the dispatcher
(section 5.2) and/or the master waiting until a lease at the tail has
ran out (so that it won't serve requests anymore).

Another correct explanation is to point out that the client requests
always go through the dispatcher, and that if the tail is in the same
partition as the dispatcher, then the tail could process updates and
reads. Otherwise, the system is stalled until reconfiguration happens
and the tail is removed from the system. CR won't yield a stale value.


## Frangipani

A. This change can lead to a race condition. A releases the lock on
say directory d.  A starts applying its changes to the directory,
while B acquires the lock on the directory.  B reads an inconsistent
directory because A hasn't completed writing yet.  B could also flush
changes to Petal that conflict with A's, because it could make its
changes quickly and a workstation C may ask for the lock, causing B to
write to Petal while A is still writing too.

B. Locks have leases. Once the lease expires, the lock server tells B
to try to recover A's log in case it failed without completing the
operation. Once B tells the lock server that it recovered A's log, all
locks from A will be released and the ones B wants will be granted.


## Spanner

A. 5. The start rule says no less than the value of TT.now().latest
(which is 5 in the setup of the question).

B. 5+epsilon (or 6 if you restrict to whole numbers), according to
commit wait rule, which makes the transaction wait until 5 < TT.now().

C. 5+epsilon. T2 starts after T1 completed (i.e., after T1 stopped
waiting), so the absolute time when T2 starts is at least 5+epsilon.
Therefore, TT.now().latest is at least that.

## FaRM

A. Yes, it is correct. Acquiring a lock still checks that the version
hasn't been changed and that the lock hasn't been acquired.

B. It reduces performance for reads, for a number of reasons.
First, read-only transactions would have to be serialized, when
previously they could run in parallel. Second, locking requires
remote CPU time, whereas validation does not require any work from
the remote CPU.

## Spark

A. Yes, it is correct

B. It will be slow, because links won't be kept in memory and each
iteration of the loop requires a join with links.

## Memcache

Yes.  The second client will observe that the key is marked as
deleted, and will read the data from the database instead of the warm
cluster.  If will successfully set the key to the new value, if its
lease is still valid.  If a concurrent client does an update to the
key in the database, its delete will invalidate the lease, and the
client wouldn't be able to set a now stale value.

## Blockcache

A. Each participant has their own cloud storage (S3, GDrive, AWS, &c), 
accessible through the Gaia storage service, in which they store their 
questions and answers.

B. To make private questions, students encrypt them using the instructors'
public key. Instructors reply encrypting with student's public key.
Both also include encrypted copies with their public key so they can read things
back. Since storage is mutable and untrusted, integrity can be provided by 
having each user sign their questions/comments with their private key.

C. The app is programmed on Javascript and runs entirely on each
user's browser, periodically checking all participant's public
directory for the latest questions and answers. Questions are stored
as mutable storage in the storage layer, so the client will retrieve the relevant URIs
via the zonefiles of each member of the class. Client will verify the integrity
of the signatures using the public keys of the other class members 
and decrypt private posts using the user's private key.

D. The block chain records the name and indirectly the public key for
a user through the zone hash.  All users see the same block chain.

E. None.

## 6.824

Omit:
CR                       XXXXX
Frangipani               XXXXXXXXXXXXXXXXX
Distributed Transactions XX
Spanner                  XXXXXXXXX
FaRM                     XXXXXXXXX
Spark                    XXXXX
Memcache                 XXXXXXXXX
SUNDR                    XXXXXXXXXXXXXXXXXXXXXXXXX
Bitcoin                  XXXX
Blockstack               XXXXXXXXXXXXXXXXXXXXXXX
AnalogicFS               XXXXXXXXXXXXXXXXXXXXXXXXX

Useful:
CR                       XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
Frangipani               XXXXXXXXXXXXXXXXXX
Distributed Transactions XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
Spanner                  XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
FaRM                     XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
Spark                    XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
Memcache                 XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
SUNDR                    XXXXXXXXXXXXXXX
Bitcoin                  XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
Blockstack               XXXXXXXXXXXXXX
AnalogicFS               XXXXXXXXXXX

Feedback:
TA exam review sessions XXXXXXX
Release a lab 2 solution to build on XXXXXX
Smaller programming assignments focusing on different papers besides Raft XXXXX
No ASCII notes, post slides with diagrams. XXXX
More details/lecture on lab 4 design XXXX
Q&A lecture for lab 3 XXX
Release Notes on important reading points in papers beforehand XXX
Spread papers out throughout semester instead of backloading XXX
Post lab tips in easier to find place, keep Jose's pretty printing notes. XX
Refresher on Linux filesystem topics (inodes) X
More office hours/staffing X
Consistent machine/environment/VM grader for students to test on X
More female representation in paper authors or guest speaker. X
Byzantine fault tolerance. X
Include notes on follow-up material for the course X
Include design write-up for 4B for partial credit with broken code. X
Declining late policy instead of D. X
More readings about systems like Tor, Kademlia and less on datacenters X
Give answers to lecture questions after each lecture. X
Switch SUNDR out for a more modern paper like Ethereum X
More collaboration opportunities/make lab 4 collaborative. X
Less weight on exams X
Checkoffs on code to get feedback X
