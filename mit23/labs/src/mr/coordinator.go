package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
)

type TaskEntry struct {
	TaskId      string
	TaskType    TaskType
	TaskSlot    int
	TaskFile    string
	ScheduledTo string
	ScheduleAt  int
	Status      TaskStatus
}

type Coordinator struct {
	inputFiles   []string
	mapperCount  int
	reducerCount int
	taskTable    []TaskEntry
}

//
// the RPC argument and reply types are defined in rpc.go.
//
func (c *Coordinator) GetNextTask(args *GetNextTaskArgs, reply *GetNextTaskReply) error {
	fmt.Printf("<< GetNextTask Req: %+v\n", args)
	reply.TaskId = ""
	reply.TaskType = Wait
	reply.TaskSlot = -1
	reply.MapperCount = c.mapperCount
	reply.ReducerCount = c.reducerCount
	fmt.Printf(">> GetNextTask Res: %+v\n", reply)
	// fmt.Printf("Task Table:\n%+v\n", c.taskTable)
	return nil
}

func (c *Coordinator) AckTaskCompletion(args *AckTaskCompletionArgs, reply *AckTaskCompletionReply) error {
	fmt.Printf("<< AckTaskCompletion Req: %+v\n", args)
	reply.Status = "ok"
	fmt.Printf(">> AckTaskCompletion Res %+v\n", reply)
	// fmt.Printf("Task Table:\n%+v\n", c.taskTable)
	return nil
}

//
// start a thread that listens for RPCs from worker.go
//
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
	fmt.Printf("Coordinator server started.\n")
}

//
// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
//
func (c *Coordinator) Done() bool {
	for _, entry := range c.taskTable {
		if entry.Status != Completed {
			return false
		}
	}
	fmt.Printf("** All Done() ***\n")
	return true
}

//
// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}
	c.inputFiles = files
	c.mapperCount = len(files)
	c.reducerCount = nReduce

	// Prepare task table
	c.taskTable = []TaskEntry{}

	// Add mapper tasks
	for i, file := range files {
		entry := TaskEntry{}
		entry.TaskSlot = i
		entry.TaskType = Map
		entry.Status = New
		entry.TaskFile = file
		c.taskTable = append(c.taskTable, entry)
	}

	// Add reducer tasks
	for i := 0; i < nReduce; i++ {
		entry := TaskEntry{}
		entry.TaskSlot = i
		entry.TaskType = Reduce
		entry.Status = New
		entry.TaskFile = fmt.Sprintf("mr-out-%d", entry.TaskSlot)
		c.taskTable = append(c.taskTable, entry)
	}
	c.server()
	return &c
}
