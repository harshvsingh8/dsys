package kvsrv

import (
	"log"
	"sync"
)

const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

type ClientState struct {
	ClientId string
	SeqNo    int
	Value    string
}

type KVServer struct {
	mu          sync.Mutex
	store       map[string]string
	clientState map[string]ClientState
}

func (kv *KVServer) init() {
	kv.store = make(map[string]string)
	kv.clientState = make(map[string]ClientState)
}

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	value, exists := kv.store[args.Key]
	if exists {
		reply.Value = value
	} else {
		reply.Value = string("")
	}
}

func (kv *KVServer) Put(args *PutAppendArgs, reply *PutAppendReply) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	kv.store[args.Key] = args.Value
	reply.Value = kv.store[args.Key]
}

func (kv *KVServer) Append(args *PutAppendArgs, reply *PutAppendReply) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	// Check and do early return if it is case of duplicate request.
	prevState, prevStatePresent := kv.clientState[args.ClientId]
	if prevStatePresent && prevState.SeqNo == args.SeqNo {
		reply.Value = prevState.Value
		return
	}

	value, exists := kv.store[args.Key]
	if exists {
		kv.store[args.Key] = value + args.Value
	} else {
		kv.store[args.Key] = args.Value
		value = string("")
	}
	// return the existing value
	reply.Value = value

	// Update the client request prev. state for handling duplicate calls.
	clientNewPrevState := ClientState{
		ClientId: args.ClientId,
		SeqNo:    args.SeqNo,
		Value:    value,
	}
	kv.clientState[args.ClientId] = clientNewPrevState
}

func StartKVServer() *KVServer {
	kv := new(KVServer)
	kv.init()
	return kv
}
