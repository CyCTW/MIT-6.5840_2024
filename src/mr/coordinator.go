package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync/atomic"
	"time"
)

const (
	TaskInit int32 = iota
	TaskStart
	TaskFinish
)

type MapTask struct {
	Status     *atomic.Int32 // status, default is init
	Start_tick SafeTime      // start tick time
	Filename   string
	Id         int
}

type ReduceTask struct {
	Status     *atomic.Int32
	Start_tick SafeTime
	Id         int
}

type Coordinator struct {
	// Your definitions here.
	map_unprocessed_chan    chan *MapTask
	map_task_finished       map[int]*MapTask // Record which map task is finished.
	map_finish_count        *atomic.Int32
	reduce_unprocessed_chan chan *ReduceTask
	reduce_task_finished    map[int]*ReduceTask // Record which map task is finished.
	reduce_finish_count     *atomic.Int32       // Count reduce finish
	is_map_finished         *atomic.Bool        // Check if it's map phase or reduce phase
	all_finish              *atomic.Bool        // Check if all phase finished
	// mtx                     sync.Mutex          // Lock for accessing queue
}

// RPC handler that worker.go call for a job (map or reduce)
// Return a not-yet-start job.
func (c *Coordinator) AskForJob(args *AskForJobArgs, reply *AskForJobReply) error {

	nReduce := len(c.reduce_task_finished)
	nMapper := len(c.map_task_finished)

	if !c.is_map_finished.Load() {

		// * Atomic accessed queue
		maptask, ok := <-c.map_unprocessed_chan

		// task may has finished. Check here
		if ok && maptask.Status.Load() != TaskFinish {
			// Write reply
			reply.Id = maptask.Id
			reply.InputFilename = c.map_task_finished[maptask.Id].Filename
			reply.NReduce = nReduce
			reply.NMapper = nMapper
			reply.WorkType = MapperType

			// * Record time
			maptask.Start_tick.Set(time.Now())
			maptask.Status.Store(TaskStart)
		} else {
			reply.Id = -1
		}

	} else {
		// reduce task
		reducetask, ok := <-c.reduce_unprocessed_chan

		if ok && reducetask.Status.Load() != TaskFinish {
			// Write reply
			reply.Id = reducetask.Id
			// reply.InputFilename No used here.
			reply.NReduce = nReduce
			reply.NMapper = nMapper
			reply.WorkType = ReducerType

			// * Record time
			reducetask.Start_tick.Set(time.Now())
			reducetask.Status.Store(TaskStart)
		} else {
			reply.Id = -1
		}

	}
	return nil

}

// RPC handler that worker.go call for completed a job (map or reduce)
func (c *Coordinator) NotifyCompleted(args *NotifyCompletedArgs, reply *NotifyCompletedReply) error {
	wid := args.Id
	wtype := args.WorkType

	// * Consider case that same id return twice

	if wtype == MapperType {
		status := c.map_task_finished[wid].Status.Load()

		if status != TaskFinish {
			_, ok := c.map_task_finished[wid]
			if ok == false {
				log.Fatalf("Key %d not exists", wid)
			}
			c.map_task_finished[wid].Status.Store(TaskFinish)

			c.map_finish_count.Add(1)
			if int(c.map_finish_count.Load()) == len(c.map_task_finished) {
				c.is_map_finished.Store(true)
			}
		}
	} else if wtype == ReducerType {
		_, ok := c.reduce_task_finished[wid]
		if ok == false {
			log.Fatalf("Key %d not exists", wid)
		}

		status := c.reduce_task_finished[wid].Status.Load()
		if status != TaskFinish {

			c.reduce_task_finished[wid].Status.Store(TaskFinish)

			c.reduce_finish_count.Add(1)

			if int(c.reduce_finish_count.Load()) == len(c.reduce_task_finished) {
				c.all_finish.Store(true)
			}
		}
	} else {
		log.Fatalf("Type %v not exists", wtype)
	}
	return nil
}

// start a thread that listens for RPCs from worker.go
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
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := c.all_finish.Load()
	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	map_mp := make(map[int]*MapTask)
	reduce_mp := make(map[int]*ReduceTask)
	// mp_queue := NewQueue[*MapTask]()
	// red_queue := NewQueue[*ReduceTask]()
	nMapper := len(files)
	mp_chan := make(chan *MapTask, nMapper+5)
	reduce_chan := make(chan *ReduceTask, nReduce+5)

	for i, file := range files {
		task := MapTask{&atomic.Int32{}, SafeTime{}, file, i}
		map_mp[i] = &task

		mp_chan <- &task
		// mp_queue.Enqueue(&task)
	}

	for i := 0; i < nReduce; i++ {
		task := ReduceTask{&atomic.Int32{}, SafeTime{}, i}
		reduce_mp[i] = &task

		reduce_chan <- &task
		// red_queue.Enqueue(&task)
	}

	// c := Coordinator{mp_queue, map_mp, &atomic.Int32{}, red_queue, reduce_mp, &atomic.Int32{}, &atomic.Bool{}, &atomic.Bool{}, sync.Mutex{}}
	c := Coordinator{mp_chan, map_mp, &atomic.Int32{}, reduce_chan, reduce_mp, &atomic.Int32{}, &atomic.Bool{}, &atomic.Bool{}}
	log.Printf("Total %d nmap, %d nreduce", len(map_mp), len(reduce_mp))
	// Your code here.
	c.server()

	// periodically check if time expired.
	go func() {
		timeout_bound := 10 * time.Second
		for {
			time.Sleep(1 * time.Second)

			now_tick := time.Now()
			for _, task := range c.map_task_finished {
				if now_tick.Sub(task.Start_tick.Get()) > timeout_bound && task.Status.Load() == TaskStart {
					// log.Print(now_tick, task.Start_tick, timeout_bound)
					// log.Printf("Add new map timeout item!, id: %d", task.Id)
					// Only grap start task
					change_successfully := task.Status.CompareAndSwap(TaskStart, TaskInit)

					if change_successfully {
						// If not change successfully, means task is finished between the if branch and cas operation.
						c.map_unprocessed_chan <- task
					}

				}
			}

			for _, task := range c.reduce_task_finished {
				if now_tick.Sub(task.Start_tick.Get()) > timeout_bound && task.Status.Load() == TaskStart {
					task.Status.Store(TaskInit)
					c.reduce_unprocessed_chan <- task
				}
			}
		}

	}()
	return &c
}
