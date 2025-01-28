package utils

import (
	"sync"
	"testing"
)

func TestRingBuffer_Basic(t *testing.T) {
	rb := NewRingBuffer[int](3)

	if rb.Len() != 0 {
		t.Errorf("Expected empty buffer, got len %d", rb.Len())
	}

	rb.Push(1)
	rb.Push(2)

	if rb.Len() != 2 {
		t.Errorf("Expected len 2, got %d", rb.Len())
	}

	if v := rb.At(0); v != 1 {
		t.Errorf("Expected 1 at index 0, got %d", v)
	}
}

func TestRingBuffer_Overflow(t *testing.T) {
	rb := NewRingBuffer[int](2)
	rb.Push(1)
	rb.Push(2)
	rb.Push(3)

	if rb.Len() != 2 {
		t.Errorf("Expected len 2, got %d", rb.Len())
	}

	if v := rb.At(0); v != 2 {
		t.Errorf("Expected 2 at index 0, got %d", v)
	}
}

func TestRingBuffer_Last(t *testing.T) {
	rb := NewRingBuffer[int](2)

	_, exists := rb.Last()
	if exists {
		t.Error("Expected Last() to return false for empty buffer")
	}

	rb.Push(1)
	rb.Push(2)

	v, exists := rb.Last()
	if !exists {
		t.Error("Expected Last() to return true")
	}
	if v != 2 {
		t.Errorf("Expected last value 2, got %d", v)
	}
}

func TestRingBuffer_Values(t *testing.T) {
	rb := NewRingBuffer[int](3)
	rb.Push(1)
	rb.Push(2)
	rb.Push(3)
	rb.Push(4)

	vals := rb.Values()
	expected := []int{2, 3, 4}

	if len(vals) != len(expected) {
		t.Errorf("Expected %v, got %v", expected, vals)
	}
	for i := range vals {
		if vals[i] != expected[i] {
			t.Errorf("At index %d: expected %d, got %d", i, expected[i], vals[i])
		}
	}
}

func TestRingBuffer_Concurrent(t *testing.T) {
	rb := NewRingBuffer[int](100)
	var wg sync.WaitGroup
	workers := 10
	pushes := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < pushes; j++ {
				rb.Push(j)
				_ = rb.Len()
				_, _ = rb.Last()
				if rb.Len() > 0 {
					_ = rb.At(0)
				}
			}
		}(i)
	}

	wg.Wait()
	if rb.Len() > rb.Cap() {
		t.Errorf("Buffer overflow: len %d > cap %d", rb.Len(), rb.Cap())
	}
}

func TestRingBuffer_AtPanic(t *testing.T) {
	rb := NewRingBuffer[int](2)

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected At() to panic on invalid index")
		}
	}()

	rb.At(0) // Should panic
}
