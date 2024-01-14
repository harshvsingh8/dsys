// Common definitions related to tasks

package mr

type TaskType int

const (
	Map TaskType = iota
	Reduce
	Wait
	None
)

func (t TaskType) String() string {
	return [...]string{"Map", "Reduce", "Wait", "None"}[t]
}

type TaskStatus int

const (
	New TaskStatus = iota
	Scheduled
	Delayed
	Rescheduled
	Completed
)

func (s TaskStatus) String() string {
	return [...]string{"New", "Scheduled", "Delayed", "Rescheduled", "Completed"}[s]
}
