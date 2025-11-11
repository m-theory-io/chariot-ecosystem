//go:build !cgo

package chariot

// This file intentionally forces a build failure when CGO is disabled.
// go-chariot requires CGO (Metal/Knapsack native solver). Disable CGO => fail fast.
// The undefined type below ensures a compile-time error.

// CGORequired is deliberately undefined; referencing it triggers the error.
var _ CGORequired
