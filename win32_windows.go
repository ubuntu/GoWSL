package gowsl

// This file contains Win32 API definitions and imports.

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	// WSL api.
	wslAPIDll                          = syscall.NewLazyDLL("wslapi.dll")
	apiWslConfigureDistribution        = wslAPIDll.NewProc("WslConfigureDistribution")
	apiWslGetDistributionConfiguration = wslAPIDll.NewProc("WslGetDistributionConfiguration")
	apiWslLaunch                       = wslAPIDll.NewProc("WslLaunch")
	apiWslLaunchInteractive            = wslAPIDll.NewProc("WslLaunchInteractive")
	apiWslRegisterDistribution         = wslAPIDll.NewProc("WslRegisterDistribution")
	apiWslUnregisterDistribution       = wslAPIDll.NewProc("WslUnregisterDistribution")
)

// Windows' typedefs.
type wBOOL = int     // Windows' BOOL
type wULONG = uint32 // Windows' ULONG
type char = byte     // Windows' CHAR (which is the same as C's char)

func queryFileType(f *os.File) (fileType, error) {
	n, err := windows.GetFileType(windows.Handle(f.Fd()))
	if err != nil {
		return fileTypeUnknown, err
	}
	return fileType(n), nil
}

// wslLaunch replaces os.StartProcess with WSL commands.
func wslLaunch(
	distroName string,
	command string,
	useCWD bool,
	stdin *os.File,
	stdout *os.File,
	stderr *os.File) (process *os.Process, err error) {
	distroUTF16, err := syscall.UTF16PtrFromString(c.distro.Name())
	if err != nil {
		return nil, errors.New("failed to convert distro name to UTF16")
	}

	commandUTF16, err := syscall.UTF16PtrFromString(command)
	if err != nil {
		return nil, fmt.Errorf("failed to convert command %q to UTF16", c.command)
	}

	var useCwdInt wBOOL
	if useCWD {
		useCwdInt = 1
	}

	var handle windows.Handle
	r1, _, _ := apiWslLaunch.Call(
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(unsafe.Pointer(commandUTF16)),
		uintptr(useCwdInt),
		stdin.Fd(),
		stdout.Fd(),
		stderr.Fd(),
		uintptr(unsafe.Pointer(&handle)))

	if r1 != 0 {
		return nil, fmt.Errorf("failed syscall to WslLaunch")
	}
	if handle == windows.Handle(0) {
		return nil, fmt.Errorf("syscall to WslLaunch returned a null handle")
	}

	pid, err := windows.GetProcessId(handle)
	if err != nil {
		return nil, errors.New("failed to find launched process")
	}

	return os.FindProcess(int(pid))
}

func wslConfigureDistribution(distributionName string, defaultUID uint32, wslDistributionFlags wslFlags) error {
	distroUTF16, err := syscall.UTF16PtrFromString(distributionName)
	if err != nil {
		return fmt.Errorf("failed to convert %q to UTF16", distributionName)
	}

	r1, _, _ := apiWslConfigureDistribution.Call(
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(defaultUID),
		uintptr(wslDistributionFlags),
	)

	if r1 != 0 {
		return fmt.Errorf("failed syscall to WslConfigureDistribution")
	}

	return nil
}

func wslGetDistributionConfiguration(distributionName string,
	distributionVersion *uint8,
	defaultUID *uint32,
	wslDistributionFlags *wslFlags,
	defaultEnvironmentVariables *map[string]string) error {
	distroUTF16, err := syscall.UTF16PtrFromString(distributionName)
	if err != nil {
		return fmt.Errorf("failed to convert %q to UTF16", distributionName)
	}

	var (
		envVarsBegin **char
		envVarsLen   uint64 // size_t
	)

	r1, _, _ := apiWslGetDistributionConfiguration.Call(
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(unsafe.Pointer(distributionVersion)),
		uintptr(unsafe.Pointer(defaultUID)),
		uintptr(unsafe.Pointer(wslDistributionFlags)),
		uintptr(unsafe.Pointer(&envVarsBegin)),
		uintptr(unsafe.Pointer(&envVarsLen)),
	)

	if r1 != 0 {
		return fmt.Errorf("failed syscall to WslGetDistributionConfiguration")
	}

	*defaultEnvironmentVariables = processEnvVariables(envVarsBegin, envVarsLen)
	return nil
}

func wslLaunchInteractive(distributionName string, command string, useCurrentWorkingDirectory bool) (exitCode uint32, err error) {
	exitCode = math.MaxUint32
	distroUTF16, err := syscall.UTF16PtrFromString(distributionName)
	if err != nil {
		return exitCode, errors.New("failed to convert distro name to UTF16")
	}

	commandUTF16, err := syscall.UTF16PtrFromString(command)
	if err != nil {
		return exitCode, fmt.Errorf("failed to convert command %q to UTF16", command)
	}

	var useCwd wBOOL
	if useCurrentWorkingDirectory {
		useCwd = 1
	}

	r1, _, _ := apiWslLaunchInteractive.Call(
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(unsafe.Pointer(commandUTF16)),
		uintptr(useCwd),
		uintptr(unsafe.Pointer(&exitCode)))

	if r1 != 0 {
		return exitCode, fmt.Errorf("failed syscall to WslLaunchInteractive")
	}

	return exitCode, nil
}

func wslRegisterDistribution(distributionName string, tarGzFilename string) error {
	distroUTF16, err := syscall.UTF16PtrFromString(distributionName)
	if err != nil {
		return errors.New("failed to convert distro name to UTF16")
	}

	tarGzFilenameUTF16, err := syscall.UTF16PtrFromString(tarGzFilename)
	if err != nil {
		return fmt.Errorf("failed to convert rootfs '%q' to UTF16", tarGzFilename)
	}

	r1, _, _ := apiWslRegisterDistribution.Call(
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(unsafe.Pointer(tarGzFilenameUTF16)))

	if r1 != 0 {
		return fmt.Errorf("failed syscall to wslRegisterDistribution")
	}

	return nil
}

func wslUnregisterDistribution(distributionName string) error {
	distroUTF16, err := syscall.UTF16PtrFromString(distributionName)
	if err != nil {
		return errors.New("failed to convert distro name to UTF16")
	}

	r1, _, _ := apiWslUnregisterDistribution.Call(uintptr(unsafe.Pointer(distroUTF16)))

	if r1 != 0 {
		return fmt.Errorf("failed syscall to WslLaunchInteractive")
	}
	return nil
}

// processEnvVariables takes the (**char, length) obtained from Win32's API and returs a
// map[variableName]variableValue. It also deallocates each of the *char strings as well
// as the **char array.
func processEnvVariables(cStringArray **char, len uint64) map[string]string {
	stringPtrs := unsafe.Slice(cStringArray, len)

	env := make(chan struct {
		key   string
		value string
	})

	wg := sync.WaitGroup{}
	for _, cStr := range stringPtrs {
		cStr := cStr
		wg.Add(1)
		go func() {
			defer wg.Done()
			goStr := stringCtoGo(cStr, 32768)
			idx := strings.Index(goStr, "=")
			env <- struct {
				key   string
				value string
			}{
				key:   strings.Clone(goStr[:idx]),
				value: strings.Clone(goStr[idx+1:]),
			}
			windows.CoTaskMemFree(unsafe.Pointer(cStr))
		}()
	}

	// Cleanup
	go func() {
		wg.Wait()
		windows.CoTaskMemFree(unsafe.Pointer(cStringArray))
		close(env)
	}()

	// Collecting results
	m := map[string]string{}

	for kv := range env {
		m[kv.key] = kv.value
	}

	return m
}

// stringCtoGo converts a null-terminated *char into a string
// maxlen is the max distance that will searched. It is meant
// to prevent or mitigate buffer overflows.
func stringCtoGo(cString *char, maxlen uint64) (goString string) {
	size := strnlen(cString, maxlen)
	return string(unsafe.Slice(cString, size))
}

// strnlen finds the null terminator to determine *char length.
// The null terminator itself is not counted towards the length.
// maxlen is the max distance that will searched. It is meant to
// prevent or mitigate buffer overflows.
func strnlen(ptr *char, maxlen uint64) (length uint64) {
	length = 0
	for ; *ptr != 0 && length <= maxlen; ptr = charNext(ptr) {
		length++
	}
	return length
}

// charNext advances *char by one position.
func charNext(ptr *char) *char {
	return (*char)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + unsafe.Sizeof(char(0))))
}