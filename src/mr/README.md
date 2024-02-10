# Lab1

## Your Job (moderate/hard)
[Link](https://pdos.csail.mit.edu/6.824/labs/lab-mr.html)
> Your job is to implement a distributed MapReduce, consisting of two programs, the coordinator and the worker. There will be just one coordinator process, and one or more worker processes executing in parallel. In a real system the workers would run on a bunch of different machines, but for this lab you'll run them all on a single machine. The workers will talk to the coordinator via RPC. Each worker process will ask the coordinator for a task, read the task's input from one or more files, execute the task, and write the task's output to one or more files. The coordinator should notice if a worker hasn't completed its task in a reasonable amount of time (for this lab, use ten seconds), and give the same task to a different worker.

## Status
- Pass all `test-mr.sh` tests.
- Lock-free implementation.

## Data structures design thoughts

### First thoughts
- Use a `map` to stored map/reduce jobs status
  - Key: Id, Value: A struct that stored the job status.
- Use boolean to record whether map phase finished, and all phase finished.
- Record the last sent time(time.Time) of a task.
  - Multiple thread may access the time, so need to think a sync mechanism like lock.

- Problem: With only `map` data structure, some operation may time-consuming
  1. Coordinator find unprocessed job.
     - With only map, we need to iterate all value in map, check job status to find which job is unprocessed.
     - Time complexity: O(n), n == size of map
  2. It's hard to know whether map phase is finished, or reduce phase is finished.
     - With only map, need to iterate all value and count finished task
     - Time complexity: O(n)
  
### Further Optimization I

- For problem 1, we can apply another data structure like **Queue** to stored unprocessed jobs.
  - See a simple queue implementation in `queue.go`
  - For coordinator, only need to pop a element from queue, and process it.
  - Time complexity: O(1)
  - For expired job, need to re-push to the queue for further handling.
  - Sync:
    - There may be two thread accessing queue(Enqueue/Deque), so a lock may be needed.
- For problem 2, we can use a **atomic counter** to record how many task has finished.
  - The counter should only be increate monotonically.
  - Need to be careful that task we think is expired, can't count twice.

### Further Optimization II
- For problem 1, queue is good to go. But is there any approach that can be lock-free?
  - Actually, go already provide a builtin mechanism that's similar to SPSC queue between goroutine/thread. i.e. `channel`
  - Then, we can use `channel` to replace queue, and do lock-free.
- For time field, actually we only need the unix timestamp, and we can stored it as atomic.Int64.

## Developing Notes:
Problem: How does reducer works? 
- Mapper will map result into nReduce tasks using hash function. And it's guarantee that same key will in same reduce task. So the reducer pick a task and collect all data in the task and sort by key, process them.


RPC Schema: 
1. Worker ask coordinator for a work.
- AskForJobArgs
  - id: int
- AskForJobReply
  - filename: processed filename
  - worktype: map/reduce
  - nReduce
1. Worker notfy coordinator that it finished task.
- NotifyCompletedArgs
  - id: int
- NotifyCompletedReply
  - ?

## Data structure design
1. Directlyl use map
2. Use a vector like struct to quickly find unprocessed job

## TODO
1. Clean code

## Edge cases:

1. Handle expired requests:
- Scenario: Coordinator think the requests has expired, and send job to another worker. But actually the origin worker send back the result.
  - It's inevitable that may have extra map job run.

- Lock
  - Queue Lock: Lock for enqueue/dequeue
  - Time Lock: Lock for accessing 

## Lesson learned
- Capitalize Member of RPC struct
- Think carefully of race condition case.