package kvsrv

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"math/big"

	"6.5840/labrpc"
)

type Clerk struct {
	server      *labrpc.ClientEnd
	clientId    string
	updateSeqNo int
	// You will have to modify this struct.
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

// Generate a unique identifier
func generateUniqueID() (string, error) {
	// Create a byte slice to hold the random data
	bytes := make([]byte, 16) // 16 bytes = 128 bits

	// Read random data into the byte slice
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", err
	}

	// Encode the byte slice to a hex string
	return hex.EncodeToString(bytes), nil
}

func MakeClerk(server *labrpc.ClientEnd) *Clerk {
	// You'll have to add code here.
	ck := new(Clerk)
	ck.server = server
	clientId, err := generateUniqueID()
	if err != nil {
		panic("Could not generate client unique Id")
	}
	ck.clientId = clientId
	ck.updateSeqNo = 0
	return ck
}

// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.server.Call("KVServer.Get", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) Get(key string) string {
	// You will have to modify this function.
	args := GetArgs{
		Key: key,
	}
	reply := GetReply{}
	for !ck.server.Call("KVServer.Get", &args, &reply) {
	}
	return reply.Value
}

// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.server.Call("KVServer."+op, &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) PutAppend(key string, value string, op string) string {
	// You will have to modify this function.
	reply := PutAppendReply{}
	args := PutAppendArgs{
		Key:      key,
		Value:    value,
		ClientId: ck.clientId,
		SeqNo:    ck.updateSeqNo,
	}

	for !ck.server.Call("KVServer."+op, &args, &reply) {
	}
	ck.updateSeqNo++
	return reply.Value
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}

// Append value to key's value and return that value
func (ck *Clerk) Append(key string, value string) string {
	return ck.PutAppend(key, value, "Append")
}
