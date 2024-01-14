package mr

import (
	"fmt"
	"hash/fnv"
	"log"
	"net/rpc"
	"time"
)

//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

//
// main/mrworker.go calls this function.
//
func Worker(
	mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {
	workerId := pseudo_uuid()
	fmt.Printf("Worker - started with Id: %s\n", workerId)

	for {
		nextTaskArgs := GetNextTaskArgs{}
		nextTaskArgs.WorkerId = workerId
		nextTaskReply := GetNextTaskReply{}

		fmt.Printf("Worker - Request Next Task...\n")
		if ok := call("Coordinator.GetNextTask", &nextTaskArgs, &nextTaskReply); !ok {
			fmt.Printf("!! GetNextTask Failed!")
			continue

		}

		fmt.Printf("Worker - Next Task Received: %+v\n", nextTaskReply)

		switch nextTaskReply.TaskType {
		case None:
			break
		case Wait:
			time.Sleep(2 * time.Second)
		case Map:
		case Reduce:
		default:
			fmt.Printf("!! Worker - Task type not implemented: %s", nextTaskReply.TaskType.String())
		}
	}
}

//
// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
