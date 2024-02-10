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

type OldValue struct {
	value string
	Id    int64
}
type KVServer struct {
	mu sync.Mutex

	// Your definitions here.
	data              map[string]string  // real data for k-v store
	last_client_op_id map[int64]OldValue // Last op id for each client
}

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.
	// ? Get op doesn't need to record act id?

	key := args.Key
	kv.mu.Lock()
	val, ok := kv.data[key]
	kv.mu.Unlock()

	if ok {
		reply.Value = val
	} else {
		reply.Value = ""
	}
}

func (kv *KVServer) Put(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	cid := args.ClientId
	opid := args.OpId

	kv.mu.Lock()
	if kv.last_client_op_id[cid].Id != opid {
		// apply action
		// For put, overwrite value
		// old := kv.data[key]
		kv.data[args.Key] = args.Value
		// kv.last_client_op_id[cid].Id = opid
		kv.last_client_op_id[cid] = OldValue{"", opid}
		// kv.last_client_op_id[cid] = OldValue{old, opid}
	}
	kv.mu.Unlock()

}

func (kv *KVServer) Append(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	cid := args.ClientId
	opid := args.OpId
	key := args.Key
	val := args.Value

	kv.mu.Lock()
	if kv.last_client_op_id[cid].Id != opid {
		// apply action
		old := kv.data[key]
		kv.data[key] = old + val

		// Return old value
		reply.Value = old
		// cache last value
		kv.last_client_op_id[cid] = OldValue{old, opid}
	} else {
		// Need cache old value
		reply.Value = kv.last_client_op_id[cid].value
	}

	kv.mu.Unlock()
}

func StartKVServer() *KVServer {
	kv := new(KVServer)

	// You may need initialization code here.
	kv.data = make(map[string]string)
	kv.last_client_op_id = make(map[int64]OldValue)
	return kv
}
