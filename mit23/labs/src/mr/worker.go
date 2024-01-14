package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
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
			Mapper(mapf, nextTaskReply.TaskFileName, nextTaskReply.TaskSlot, nextTaskReply.ReducerCount)
			ackArgs := AckTaskCompletionArgs{}
			ackReply := AckTaskCompletionReply{}
			ackArgs.TaskId = nextTaskReply.TaskId
			if ok := call("Coordinator.AckTaskCompletion", &ackArgs, &ackReply); !ok {
				fmt.Printf("!! AckTaskCompletion Failed!")
			}
			time.Sleep(4 * time.Second)
		case Reduce:
			time.Sleep(8 * time.Second)
		default:
			fmt.Printf("!! Worker - Task type not implemented: %s", nextTaskReply.TaskType.String())
		}
	}
}

//
// Mapper handling
//
func Mapper(
	mapf func(string, string) []KeyValue,
	inputFile string,
	mapSlot int,
	reducerCount int) {
	file, err := os.Open(inputFile)
	if err != nil {
		log.Fatalf("cannot open %v", inputFile)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", inputFile)
	}
	file.Close()

	keyValues := mapf(inputFile, string(content))

	keyValuesByReducer := make([][]KeyValue, reducerCount)

	for _, kv := range keyValues {
		reducerSlot := ihash(kv.Key) % reducerCount
		keyValuesByReducer[reducerSlot] = append(keyValuesByReducer[reducerSlot], kv)
	}

	for i := 0; i < reducerCount; i++ {
		tmpFile, err := CreateTempFile()
		if err != nil {
			log.Fatalf("cannot create temp file")
		}
		encoder := json.NewEncoder(tmpFile)
		err = encoder.Encode(keyValuesByReducer[i])
		fileInfo, err := tmpFile.Stat()
		if err != nil {
			log.Fatalf("cannot get path for the temp file, %+v", err)
		}
		tmpFile.Close()
		intFileName := fmt.Sprintf("mr-int-%d-%d.json", i, mapSlot)
		err = os.Rename(fileInfo.Name(), intFileName)
		if err != nil {
			log.Fatalf("cannot rename file: %+v", err)
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
