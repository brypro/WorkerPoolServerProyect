package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// Job represents a job to be executed, with a name and a number and a delay
type Job struct {
	Name   string        // name of the job
	Delay  time.Duration // delay between each job
	Number int           // number to calculate on the fibonacci sequence
}

// Worker will be our concurrency-friendly worker
type Worker struct {
	Id         int           // id of the worker
	JobQueue   chan Job      // Jobs to be processed
	WorkerPool chan chan Job // Pool of workers
	Quit       chan bool     // Quit worker
}

// Dispatcher is a dispatcher that will dispatch jobs to workers
type Dispatcher struct {
	WorkerPool chan chan Job // Pool of workers
	MaxWorkers int           // Maximum number of workers
	JobQueue   chan Job      // Jobs to be processed
}

// NewWorker constructor returns a new Worker with the provided id and workerpool
func NewWorker(id int, workerPool chan chan Job) *Worker {
	return &Worker{
		Id:         id,
		WorkerPool: workerPool,
		JobQueue:   make(chan Job),  // create a job queue
		Quit:       make(chan bool), // Channel to end jobs
	}
}

// Start method starts all workers
func (w Worker) Start() {
	go func() {
		for true {
			w.WorkerPool <- w.JobQueue // add job to pool
			// Multiplexing
			select {
			case job := <-w.JobQueue: // get job from queue
				fmt.Printf("worker%d: started %s, %d\n", w.Id, job.Name, job.Number)
				fib := Fibonacci(job.Number)
				time.Sleep(job.Delay)
				fmt.Printf("worker%d: finished %s, %d with result %d\n", w.Id, job.Name, job.Number, fib)
			case <-w.Quit: // quit worker
				fmt.Printf("Worker with id %d Stopped\n", w.Id)
				return
			}
		}
	}()
}

// Stop method of worker
func (w Worker) Stop() {
	go func() {
		w.Quit <- true
	}()
}

// Fibonacci calculates the fibonacci sequence
func Fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return Fibonacci(n-1) + Fibonacci(n-2)
}

// NewDispatcher returns a new Dispatcher with the provided maxWorkers
func NewDispatcher(jobQueue chan Job, maxWorkers int) *Dispatcher {
	pool := make(chan chan Job, maxWorkers)
	return &Dispatcher{
		MaxWorkers: maxWorkers,
		JobQueue:   jobQueue,
		WorkerPool: pool,
	}
}

// Dispatch will dispatch jobs to workers
func (d *Dispatcher) dispatch() {
	for {
		select {
		case job := <-d.JobQueue: // get job from queue
			// Asign the job to a worker
			go func() {
				jobChannel := <-d.WorkerPool // get worker from pool
				jobChannel <- job            // Workers will read from this channel
			}()
		}
	}
}

func (d *Dispatcher) Run() {
	for i := 0; i < d.MaxWorkers; i++ {
		worker := NewWorker(i+1, d.WorkerPool)
		worker.Start()
	}

	go d.dispatch()
}

func main() {
	const (
		maxWorkers   = 4
		maxQueueSize = 20
		port         = ":3000"
	)
	jobQueue := make(chan Job, maxQueueSize)
	dispatcher := NewDispatcher(jobQueue, maxWorkers)
	dispatcher.Run()

	//http://localhost:3000/fibonacci
	http.HandleFunc("/fibonacci", func(w http.ResponseWriter, r *http.Request) {
		RequestHandler(w, r, jobQueue)
	})
	fmt.Println("http://localhost:3000/fibonacci")
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal("Cannot run server")
	}
}

func RequestHandler(w http.ResponseWriter, r *http.Request, jobQueue chan Job) {
	if r.Method != "POST" {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
	delay, err := time.ParseDuration(r.FormValue("delay"))
	if err != nil {
		http.Error(w, "Invalid delay", http.StatusBadRequest)
		return
	}
	value, err := strconv.Atoi(r.FormValue("value"))
	if err != nil {
		http.Error(w, "Invalid value", http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "Invalid name", http.StatusBadRequest)
		return
	}
	job := Job{name, delay, value}
	jobQueue <- job
	w.WriteHeader(http.StatusCreated)
}
