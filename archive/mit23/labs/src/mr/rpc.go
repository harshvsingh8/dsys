package mr

//
// RPC definitions.
//

import (
	"os"
	"strconv"
)

// Request/Reply for requesting new task to execute.
type GetNextTaskArgs struct {
	WorkerId string
}

type GetNextTaskReply struct {
	TaskId       string
	TaskType     TaskType
	TaskSlot     int
	TaskFileName string
	MapperCount  int
	ReducerCount int
}

// Request/reply for to ack task completion (by worker)
type AckTaskCompletionArgs struct {
	TaskId string
}

type AckTaskCompletionReply struct {
	Status string
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
