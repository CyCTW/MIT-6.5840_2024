# Lab1

## Your Job (moderate/hard)
[Link](https://pdos.csail.mit.edu/6.824/labs/lab-mr.html)
> Your job is to implement a distributed MapReduce, consisting of two programs, the coordinator and the worker. There will be just one coordinator process, and one or more worker processes executing in parallel. In a real system the workers would run on a bunch of different machines, but for this lab you'll run them all on a single machine. The workers will talk to the coordinator via RPC. Each worker process will ask the coordinator for a task, read the task's input from one or more files, execute the task, and write the task's output to one or more files. The coordinator should notice if a worker hasn't completed its task in a reasonable amount of time (for this lab, use ten seconds), and give the same task to a different worker.

## Status
- Pass all `test-mr.sh` tests.

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