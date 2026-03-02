package workerpool

import (
	"fmt"
	"runtime"
	"sync"
)

// Pool provides a reusable worker pool pattern with generic task type T
type Pool[T any] struct {
	numWorkers int
	taskCh     chan T
	wg         sync.WaitGroup
}

// New creates a new worker pool with the specified number of workers
func New[T any](numWorkers int) *Pool[T] {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}

	return &Pool[T]{
		numWorkers: numWorkers,
		taskCh:     make(chan T, numWorkers*2), // buffer size to reduce contention
	}
}

// Start initializes and starts the worker pool
func (wp *Pool[T]) Start(processFn func(taskCh <-chan T)) {
	for i := 0; i < wp.numWorkers; i++ {
		wp.wg.Add(1)
		go func() {
			defer wp.wg.Done()
			defer func() {
				if r := recover(); r != nil {
					// ToDo: May be better to add errCh
					fmt.Printf("Worker panicked: %v\n", r)
				}
			}()
			processFn(wp.taskCh)
		}()
	}
}

// Submit adds a task to the worker pool
func (wp *Pool[T]) Submit(task T) {
	wp.taskCh <- task
}

// CloseTaskChannel signals that no more tasks will be submitted
func (wp *Pool[T]) CloseTaskChannel() {
	close(wp.taskCh)
}

// Wait blocks until all workers have completed their tasks
func (wp *Pool[T]) Wait() {
	wp.wg.Wait()
}

// CalculateOptimalWorkers returns an optimal number of workers based on input size
func CalculateOptimalWorkers(inputSize int) int {
	cpus := runtime.NumCPU()

	// for small inputs, don't use too many workers
	if inputSize < 10 {
		return 1
	} else if inputSize < 100 {
		return min(cpus/2, 4)
	}

	return cpus
}
