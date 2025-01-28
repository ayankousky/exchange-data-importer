package utils

import "sync"

// RingBuffer is a fixed-size circular buffer storing up to `capacity` items of type T.
// When full, new pushes overwrite the oldest items.
type RingBuffer[T any] struct {
	data     []T
	start    int
	size     int
	capacity int
	mu       sync.RWMutex
}

// NewRingBuffer allocates a new ring buffer with the given capacity.
func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	return &RingBuffer[T]{
		data:     make([]T, capacity),
		capacity: capacity,
	}
}

// Push adds a new item to the ring. If full, overwrites the oldest.
func (r *RingBuffer[T]) Push(value T) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.size < r.capacity {
		// Not full: place item at (start+size) % capacity
		idx := (r.start + r.size) % r.capacity
		r.data[idx] = value
		r.size++
	} else {
		// Full: overwrite the oldest item at `start`
		r.data[r.start] = value
		r.start = (r.start + 1) % r.capacity
	}
}

// At returns the element at index i (0=oldest, size-1=newest).
func (r *RingBuffer[T]) At(i int) T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if i < 0 || i >= r.size {
		panic("ring: index out of range")
	}
	return r.data[(r.start+i)%r.capacity]
}

// Len returns the number of items in the ring.
func (r *RingBuffer[T]) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.size
}

// Cap returns the total ring capacity.
func (r *RingBuffer[T]) Cap() int {
	return r.capacity
}

// Last returns the newest item, if any.
func (r *RingBuffer[T]) Last() (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var zero T
	if r.size == 0 {
		return zero, false
	}
	idx := (r.start + r.size - 1) % r.capacity
	return r.data[idx], true
}

// Values returns a copy of all items in order from oldest to newest.
func (r *RingBuffer[T]) Values() []T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]T, r.size)
	for i := 0; i < r.size; i++ {
		out[i] = r.data[(r.start+i)%r.capacity]
	}
	return out
}
