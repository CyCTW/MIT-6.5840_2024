package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/rpc"
	"os"
	"sort"
	"time"
)

// for sorting by key.
type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

const (
	MapperType = iota
	ReducerType
)

func HandleMapWork(reply *AskForJobReply, mapf func(string, string) []KeyValue) {
	// 1. Create temp file for output
	filename := reply.InputFilename
	file, err := os.Open(reply.InputFilename)
	if err != nil {
		log.Fatalf("cannot open %v", reply.InputFilename)
	}
	content, err := io.ReadAll(file)

	if err != nil {
		log.Fatalf("cannot read %v", reply.InputFilename)
	}
	file.Close()

	kva := mapf(filename, string(content))
	// 3. Rename tmp file to real filename

	encs := []json.Encoder{}
	file_names := []string{}

	for i := 0; i < reply.NReduce; i++ {
		dir, err := os.MkdirTemp("", "tmp")
		if err != nil {
			log.Fatalf("create tmp file failed")
		}
		defer os.RemoveAll(dir)

		file, err := os.CreateTemp(dir, "")
		if err != nil {
			log.Fatalf("can't open temp file")
		}

		file_names = append(file_names, file.Name())
		encs = append(encs, *json.NewEncoder(file))
	}

	for _, kv := range kva {
		partition := ihash(kv.Key) % reply.NReduce
		err := encs[partition].Encode(&kv)
		if err != nil {
			log.Fatalf("encode error")
		}
	}

	// Rename files
	for i := 0; i < reply.NReduce; i++ {
		out_file := fmt.Sprintf("mr-%d-%d", reply.Id, i)
		os.Rename(file_names[i], out_file)
	}
}

func HandleReduceWork(reply *AskForJobReply, reducef func(string, []string) string) {
	// exec reducer func
	decs := []*json.Decoder{}
	nMapper := reply.NMapper

	// 1. Read all file that belongs to the reduce task id. "mr-*-Y".
	for i := 0; i < nMapper; i++ {
		input_file := fmt.Sprintf("mr-%d-%d", i, reply.Id)
		file, ok := os.Open(input_file)
		if ok != nil {
			log.Fatalf("Can't open file %v", input_file)
		}
		decs = append(decs, json.NewDecoder(file))
	}

	intermediate := []KeyValue{}
	// 2. Group all intermediate file that belongs to id reduce task
	for i := 0; i < nMapper; i++ {
		for {
			var kv KeyValue
			if err := decs[i].Decode(&kv); err != nil {
				break
			}
			intermediate = append(intermediate, kv)
		}
	}
	// 3. Sort by key
	sort.Sort(ByKey(intermediate))

	// 4. Create tmp file for output
	dir, err := os.MkdirTemp("", "tmp2")
	if err != nil {
		log.Fatalf("create tmp file failed")
	}
	defer os.RemoveAll(dir)

	file, err := os.CreateTemp(dir, "")
	if err != nil {
		log.Fatalf("can't open temp file")
	}

	i := 0
	for i < len(intermediate) {
		j := i + 1
		for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}
		output := reducef(intermediate[i].Key, values)

		// this is the correct format for each line of Reduce output.
		fmt.Fprintf(file, "%v %v\n", intermediate[i].Key, output)

		i = j
	}

	file.Close()

	// Atomic rename
	// 5. Write file
	oname := fmt.Sprintf("mr-out-%d", reply.Id)
	os.Rename(file.Name(), oname)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.
	for {

		reply := AskForJobByRPC()
		if reply == nil || reply.Id == -1 {
			// log.Print("No job currently")
			time.Sleep(1 * time.Second)
			continue
		}
		wtype := reply.WorkType
		if wtype == MapperType {
			HandleMapWork(reply, mapf)

		} else if wtype == ReducerType {
			HandleReduceWork(reply, reducef)
		} else {
			log.Fatalf("Type %v not exists", wtype)
		}

		NotifyCompletedRPC(reply.Id, wtype)

	}

}

func AskForJobByRPC() *AskForJobReply {
	args := AskForJobArgs{}
	reply := AskForJobReply{}

	ok := call("Coordinator.AskForJob", &args, &reply)
	if ok {
		return &reply
	} else {
		fmt.Printf("call failed!\n")
		return nil
	}
}

func NotifyCompletedRPC(Id int, WorkerType int) *NotifyCompletedReply {
	args := &NotifyCompletedArgs{Id, WorkerType}
	reply := NotifyCompletedReply{}
	// fmt.Print("Args", args)
	ok := call("Coordinator.NotifyCompleted", args, &reply)
	if ok {
		return &reply
	} else {
		// fmt.Printf("call failed!\n")
		return nil
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing error")
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
