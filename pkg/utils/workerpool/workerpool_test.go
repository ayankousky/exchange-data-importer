package workerpool

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		numWorkers int
		expected   int
	}{
		{
			name:       "positive workers count",
			numWorkers: 4,
			expected:   4,
		},
		{
			name:       "zero workers count",
			numWorkers: 0,
			expected:   runtime.NumCPU(),
		},
		{
			name:       "negative workers count",
			numWorkers: -1,
			expected:   runtime.NumCPU(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := New[int](tt.numWorkers)
			assert.Equal(t, tt.expected, pool.numWorkers)
			assert.NotNil(t, pool.taskCh)
			assert.Equal(t, tt.expected*2, cap(pool.taskCh))
		})
	}
}

func TestPool_Start(t *testing.T) {
	pool := New[int](2)

	var processedCount int32
	var mutex sync.Mutex
	processedTasks := make(map[int]bool)

	pool.Start(func(taskCh <-chan int) {
		for task := range taskCh {
			atomic.AddInt32(&processedCount, 1)
			mutex.Lock()
			processedTasks[task] = true
			mutex.Unlock()
		}
	})

	numTasks := 10
	for i := 0; i < numTasks; i++ {
		pool.Submit(i)
	}

	pool.CloseTaskChannel()
	pool.Wait()

	assert.Equal(t, int32(numTasks), processedCount)

	mutex.Lock()
	defer mutex.Unlock()
	assert.Equal(t, numTasks, len(processedTasks))
	for i := 0; i < numTasks; i++ {
		assert.True(t, processedTasks[i])
	}
}

func TestPool_Panic(t *testing.T) {
	pool := New[int](1)

	var processed bool
	pool.Start(func(taskCh <-chan int) {
		for task := range taskCh {
			if task == 0 {
				processed = true
			} else {
				panic("test panic")
			}
		}
	})

	pool.Submit(0)
	pool.Submit(1) // This should cause a panic in the worker

	pool.CloseTaskChannel()
	pool.Wait() // Should not hang despite the panic

	assert.True(t, processed)
}

func TestCalculateOptimalWorkers(t *testing.T) {
	cpus := runtime.NumCPU()

	tests := []struct {
		name      string
		inputSize int
		expected  int
	}{
		{
			name:      "small input",
			inputSize: 5,
			expected:  1,
		},
		{
			name:      "medium input",
			inputSize: 50,
			expected:  min(cpus/2, 4),
		},
		{
			name:      "large input",
			inputSize: 500,
			expected:  cpus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateOptimalWorkers(tt.inputSize)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPool_Concurrency(t *testing.T) {
	numWorkers := 4
	pool := New[func()](numWorkers)

	var counter int32
	var wg sync.WaitGroup

	pool.Start(func(taskCh <-chan func()) {
		for task := range taskCh {
			task()
		}
	})

	numTasks := 100
	wg.Add(numTasks)

	for i := 0; i < numTasks; i++ {
		pool.Submit(func() {
			atomic.AddInt32(&counter, 1)
			time.Sleep(10 * time.Millisecond) // Simulate work
			wg.Done()
		})
	}

	pool.CloseTaskChannel()
	wg.Wait()   // Wait for all tasks to complete
	pool.Wait() // Wait for the pool to finish

	assert.Equal(t, int32(numTasks), counter)
}
