package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"os"
	"strconv"
)

// Add your RPC definitions here.
type AskForJobArgs struct {
}

type AskForJobReply struct {
	// WorkType      string // map or reduce type
	WorkType      int    // map or reduce type
	Id            int    // Id (map or reduce id)
	InputFilename string // input filename
	NReduce       int    // reduce size
	NMapper       int
}

type NotifyCompletedArgs struct {
	Id int // Id for recognize which tasks
	// workType string
	WorkType int
}

type NotifyCompletedReply struct {
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
