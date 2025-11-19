// filepath: /Users/williamhouse/go/src/github.com/bhouse1273/knapsack/tests/go/knapsack_cgo_test.go
package knapsack_test

/*
#cgo CFLAGS: -I../../knapsack-library/include
#cgo darwin LDFLAGS: -L../../knapsack-library/lib/macos-metal -lknapsack_metal
#cgo linux LDFLAGS: -L../../knapsack-library/lib/linux-cpu -lknapsack_cpu

#include <stdlib.h>
#include "knapsack_c.h"
*/
import "C"
import (
	"fmt"
	"testing"
	"unsafe"
)

// TestBasicKnapsack validates the basic knapsack solver call
func TestBasicKnapsack(t *testing.T) {
	t.Log("Test 1: Basic Knapsack Problem")

	// Problem setup
	n := C.int(5)
	weights := []C.int{C.int(2), C.int(3), C.int(4), C.int(5), C.int(9)}
	values := []C.int{C.int(3), C.int(4), C.int(5), C.int(8), C.int(10)}
	capacity := C.int(10)
	selection := make([]C.int, 5)

	t.Logf("Items: %d", n)
	t.Logf("Weights: %v", weights)
	t.Logf("Values: %v", values)
	t.Logf("Capacity: %d", capacity)

	// Call C function
	totalValue := C.knapsack(
		n,
		(*C.int)(unsafe.Pointer(&weights[0])),
		(*C.int)(unsafe.Pointer(&values[0])),
		capacity,
		(*C.int)(unsafe.Pointer(&selection[0])),
	)

	t.Logf("Total value: %d", totalValue)
	t.Logf("Selection: %v", selection)

	// Validate results
	if totalValue < 0 {
		t.Errorf("Total value should be non-negative, got %d", totalValue)
	}

	selectedWeight := C.int(0)
	selectedValue := C.int(0)
	for i := 0; i < 5; i++ {
		if selection[i] == 1 {
			selectedWeight += weights[i]
			selectedValue += values[i]
			t.Logf("  Selected item %d: weight=%d, value=%d", i, weights[i], values[i])
		}
	}

	if selectedWeight > capacity {
		t.Errorf("Total weight %d exceeds capacity %d", selectedWeight, capacity)
	}

	if selectedValue != totalValue {
		t.Errorf("Sum of selected values %d != returned total %d", selectedValue, totalValue)
	}

	t.Logf("✅ Total weight used: %d/%d", selectedWeight, capacity)
	t.Logf("✅ Total value: %d", totalValue)
}

// TestSingleItem validates single item selection
func TestSingleItem(t *testing.T) {
	t.Log("Test 2: Single Item")

	n := C.int(1)
	weights := []C.int{C.int(5)}
	values := []C.int{C.int(10)}
	capacity := C.int(10)
	selection := make([]C.int, 1)

	totalValue := C.knapsack(
		n,
		(*C.int)(unsafe.Pointer(&weights[0])),
		(*C.int)(unsafe.Pointer(&values[0])),
		capacity,
		(*C.int)(unsafe.Pointer(&selection[0])),
	)

	if totalValue != 10 {
		t.Errorf("Expected value 10, got %d", totalValue)
	}

	if selection[0] != 1 {
		t.Errorf("Item should be selected")
	}

	t.Logf("✅ Single item selected correctly: value=%d", totalValue)
}

// TestSingleItemTooHeavy validates rejection of overweight items
func TestSingleItemTooHeavy(t *testing.T) {
	t.Log("Test 3: Single Item Too Heavy")

	n := C.int(1)
	weights := []C.int{C.int(15)}
	values := []C.int{C.int(10)}
	capacity := C.int(10)
	selection := make([]C.int, 1)

	totalValue := C.knapsack(
		n,
		(*C.int)(unsafe.Pointer(&weights[0])),
		(*C.int)(unsafe.Pointer(&values[0])),
		capacity,
		(*C.int)(unsafe.Pointer(&selection[0])),
	)

	if totalValue != 0 {
		t.Errorf("Expected value 0 (item too heavy), got %d", totalValue)
	}

	if selection[0] != 0 {
		t.Errorf("Item should not be selected")
	}

	t.Logf("✅ Overweight item correctly rejected")
}

// TestZeroCapacity validates zero capacity edge case
func TestZeroCapacity(t *testing.T) {
	t.Log("Test 4: Zero Capacity")

	n := C.int(3)
	weights := []C.int{C.int(1), C.int(2), C.int(3)}
	values := []C.int{C.int(10), C.int(20), C.int(30)}
	capacity := C.int(0)
	selection := make([]C.int, 3)

	totalValue := C.knapsack(
		n,
		(*C.int)(unsafe.Pointer(&weights[0])),
		(*C.int)(unsafe.Pointer(&values[0])),
		capacity,
		(*C.int)(unsafe.Pointer(&selection[0])),
	)

	if totalValue != 0 {
		t.Errorf("Expected value 0 (zero capacity), got %d", totalValue)
	}

	for i := 0; i < 3; i++ {
		if selection[i] != 0 {
			t.Errorf("No items should be selected with zero capacity")
		}
	}

	t.Logf("✅ Zero capacity handled correctly")
}

// TestPointerValidity validates that pointers are passed correctly
func TestPointerValidity(t *testing.T) {
	t.Log("Test 5: Pointer Validity Check")

	n := C.int(3)
	weights := []C.int{C.int(1), C.int(2), C.int(3)}
	values := []C.int{C.int(5), C.int(10), C.int(15)}
	capacity := C.int(5)
	selection := make([]C.int, 3)

	// Print pointer addresses for debugging
	t.Logf("Pointer addresses:")
	t.Logf("  weights: %p", &weights[0])
	t.Logf("  values:  %p", &values[0])
	t.Logf("  selection: %p", &selection[0])

	totalValue := C.knapsack(
		n,
		(*C.int)(unsafe.Pointer(&weights[0])),
		(*C.int)(unsafe.Pointer(&values[0])),
		capacity,
		(*C.int)(unsafe.Pointer(&selection[0])),
	)

	t.Logf("Result: totalValue=%d, selection=%v", totalValue, selection)

	// Verify memory wasn't corrupted
	if weights[0] != 1 || weights[1] != 2 || weights[2] != 3 {
		t.Errorf("Weights array was corrupted")
	}
	if values[0] != 5 || values[1] != 10 || values[2] != 15 {
		t.Errorf("Values array was corrupted")
	}

	selectedWeight := C.int(0)
	for i := 0; i < 3; i++ {
		if selection[i] == 1 {
			selectedWeight += weights[i]
		}
	}

	if selectedWeight > capacity {
		t.Errorf("Capacity constraint violated: %d > %d", selectedWeight, capacity)
	}

	t.Logf("✅ Pointers passed correctly, memory intact")
}

// BenchmarkKnapsack benchmarks the knapsack solver
func BenchmarkKnapsack(b *testing.B) {
	n := C.int(20)
	weights := make([]C.int, 20)
	values := make([]C.int, 20)
	selection := make([]C.int, 20)

	// Initialize with sample data
	for i := 0; i < 20; i++ {
		weights[i] = C.int(i + 1)
		values[i] = C.int((i + 1) * 2)
	}
	capacity := C.int(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		C.knapsack(
			n,
			(*C.int)(unsafe.Pointer(&weights[0])),
			(*C.int)(unsafe.Pointer(&values[0])),
			capacity,
			(*C.int)(unsafe.Pointer(&selection[0])),
		)
	}
}

// Example demonstrates correct CGO usage for chariot-ecosystem
func ExampleKnapsack() {
	// Problem setup
	n := C.int(4)
	weights := []C.int{C.int(5), C.int(10), C.int(15), C.int(20)}
	values := []C.int{C.int(10), C.int(15), C.int(20), C.int(25)}
	capacity := C.int(30)
	selection := make([]C.int, 4)

	// Solve
	totalValue := C.knapsack(
		n,
		(*C.int)(unsafe.Pointer(&weights[0])),
		(*C.int)(unsafe.Pointer(&values[0])),
		capacity,
		(*C.int)(unsafe.Pointer(&selection[0])),
	)

	// Print results
	fmt.Printf("Total value: %d\n", totalValue)
	fmt.Printf("Selection: ")
	for i := 0; i < 4; i++ {
		if selection[i] == 1 {
			fmt.Printf("%d ", i)
		}
	}
	fmt.Println()

	// Output:
	// Total value: 45
	// Selection: 0 1 2
}
