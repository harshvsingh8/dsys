package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
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
	tableLock    sync.Mutex
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

	{
		c.tableLock.Lock()
		defer c.tableLock.Unlock()

		selectedTaskIndex := -1
		allMappersDone := true

		// First check pending map tasks
		for i := 0; i < c.mapperCount; i++ {
			if c.taskTable[i].Status != Completed {
				allMappersDone = false
			}
			if c.taskTable[i].Status == New {
				selectedTaskIndex = i
				break
			}
			if c.taskTable[i].Status == Scheduled && c.taskTable[i].ScheduleAt+10 < time.Now().Second() {
				selectedTaskIndex = i
				break
			}
		}

		if selectedTaskIndex != -1 {
			// Update task table entry for the scheduling
			selectedTask := &c.taskTable[selectedTaskIndex]
			selectedTask.Status = Scheduled
			selectedTask.ScheduleAt = time.Now().Second()
			selectedTask.ScheduledTo = args.WorkerId

			// Prepare reply
			reply.TaskType = Map
			reply.TaskId = selectedTask.TaskId
			reply.TaskFileName = selectedTask.TaskFile
			reply.TaskSlot = selectedTaskIndex

		} else if !allMappersDone {
			// There is no tasks to schedule but mappers are not done yet - so ask to wait.
			reply.TaskType = Wait
		} else {
			allReducersDone := true

			// Check pending reduce tasks - which are in the latter side of the task table
			for i := c.mapperCount; i < c.mapperCount+c.reducerCount; i++ {
				if c.taskTable[i].Status != Completed {
					allReducersDone = false
				}
				if c.taskTable[i].Status == New {
					selectedTaskIndex = i
					break
				}
				if c.taskTable[i].Status == Scheduled && c.taskTable[i].ScheduleAt+10 < time.Now().Second() {
					selectedTaskIndex = i
					break
				}
				if c.taskTable[i].Status != Completed {
					allReducersDone = false
				}
			}

			if selectedTaskIndex != -1 {
				// Update task table entry for the scheduling
				selectedTask := &c.taskTable[selectedTaskIndex]
				selectedTask.Status = Scheduled
				selectedTask.ScheduleAt = time.Now().Second()
				selectedTask.ScheduledTo = args.WorkerId

				// Prepare reply
				reply.TaskType = Reduce
				reply.TaskId = selectedTask.TaskId
				reply.TaskFileName = selectedTask.TaskFile
				reply.TaskSlot = selectedTaskIndex

			} else if !allReducersDone {
				// There is no task to schedule but all reducers are not done yet - so ask to wait.
				reply.TaskType = Wait
			} else {
				// Both Mapper and Reducer tasks are done - Worker can exit.
				reply.TaskType = None
			}
		}
	}

	fmt.Printf(">> GetNextTask Res: %+v\n", reply)
	// fmt.Printf("Task Table:\n%+v\n", c.taskTable)
	return nil
}

func (c *Coordinator) AckTaskCompletion(args *AckTaskCompletionArgs, reply *AckTaskCompletionReply) error {

	fmt.Printf("<< AckTaskCompletion Req: %+v\n", args)
	{
		c.tableLock.Lock()
		defer c.tableLock.Unlock()
		reply.Status = "not-found"
		for i := 0; i < len(c.taskTable); i++ {
			if c.taskTable[i].TaskId == args.TaskId {
				c.taskTable[i].Status = Completed
				reply.Status = "ok"
				break
			}
		}
	}
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
	c.tableLock.Lock()
	defer c.tableLock.Unlock()
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
		entry.TaskId = fmt.Sprintf("Map-%d", i)
		entry.TaskSlot = i
		entry.TaskType = Map
		entry.Status = New
		entry.TaskFile = file
		c.taskTable = append(c.taskTable, entry)
	}

	// Add reducer tasks
	for i := 0; i < nReduce; i++ {
		entry := TaskEntry{}
		entry.TaskId = fmt.Sprintf("Reduce-%d", i)
		entry.TaskSlot = i
		entry.TaskType = Reduce
		entry.Status = New
		entry.TaskFile = fmt.Sprintf("mr-out-%d", entry.TaskSlot)
		c.taskTable = append(c.taskTable, entry)
	}

	// Clear previous files
	RemoveFiles("mr-int-*")
	RemoveFiles("mr-out-*")

	// Start RPC server and return
	c.server()
	return &c
}
