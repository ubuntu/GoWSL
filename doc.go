// Package gowsl implements an idiomatic interface to manage the Windows Subsystem for
// Linux from Go. It uses WslApi calls when possible, and otherwise reads the registry
// or, as a last resort, may subprocess wsl.exe.
//
// This package also contains a mock WSL backend which can be useful for testing, as
// setting up WSL distros for every test-case can be quite time-consuming. This mock back-end
// is disabled by default, and can be enabled by using the context returned by the WithMock
// function.
package gowsl
