// Package windows contains the production backend. It is the
// one used in production code, and makes real syscalls and
// accesses to the registry.
//
// All functions will return an error when ran on Linux.
package windows

// Backend implements the Backend interface.
type Backend struct{}
