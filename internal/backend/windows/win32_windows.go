package windows

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

	"github.com/ubuntu/decorate"
	"github.com/ubuntu/gowsl/internal/flags"
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
type wBOOL = int            // Windows' BOOL
type char = byte            // Windows' CHAR (which is the same as C's char)
const fileTypePipe = 0x0003 // Windows' FILE_TYPE_PIPE

// IsPipe checks if a file's descriptor is a pipe vs. any other type of object.
func (Backend) IsPipe(f *os.File) (bool, error) {
	n, err := windows.GetFileType(windows.Handle(f.Fd()))
	if err != nil {
		return false, err
	}

	var isPipe bool
	if n == fileTypePipe {
		isPipe = true
	}

	return isPipe, nil
}

// WslLaunch is a wrapper around the WslLaunch
// function in the wslApi.dll Win32 library.
func (Backend) WslLaunch(
	distroName string,
	command string,
	useCWD bool,
	stdin *os.File,
	stdout *os.File,
	stderr *os.File) (process *os.Process, err error) {
	defer decorate.OnError(&err, "WslLaunch")

	distroUTF16, err := syscall.UTF16PtrFromString(distroName)
	if err != nil {
		return nil, errors.New("could not convert distro name to UTF16")
	}

	commandUTF16, err := syscall.UTF16PtrFromString(command)
	if err != nil {
		return nil, errors.New("could not convert command to UTF16")
	}

	var useCwdInt wBOOL
	if useCWD {
		useCwdInt = 1
	}

	var handle windows.Handle
	_, err = callDll(apiWslLaunch,
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(unsafe.Pointer(commandUTF16)),
		uintptr(useCwdInt),
		stdin.Fd(),
		stdout.Fd(),
		stderr.Fd(),
		uintptr(unsafe.Pointer(&handle)))
	if err != nil {
		return nil, err
	}

	if handle == windows.Handle(0) {
		return nil, errors.New("syscall returned a null handle")
	}

	pid, err := windows.GetProcessId(handle)
	if err != nil {
		return nil, errors.New("failed to find launched process")
	}

	return os.FindProcess(int(pid))
}

// WslConfigureDistribution is a wrapper around the WslConfigureDistribution
// function in the wslApi.dll Win32 library.
func (Backend) WslConfigureDistribution(distributionName string, defaultUID uint32, wslDistributionFlags flags.WslFlags) (err error) {
	defer decorate.OnError(&err, "WslConfigureDistribution")

	distroUTF16, err := syscall.UTF16PtrFromString(distributionName)
	if err != nil {
		return errors.New("could not convert distro name to UTF16")
	}

	_, err = callDll(apiWslConfigureDistribution,
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(defaultUID),
		uintptr(wslDistributionFlags),
	)
	if err != nil {
		return err
	}

	return nil
}

// WslGetDistributionConfiguration is a wrapper around the WslGetDistributionConfiguration
// function in the wslApi.dll Win32 library.
func (Backend) WslGetDistributionConfiguration(distributionName string,
	distributionVersion *uint8,
	defaultUID *uint32,
	wslDistributionFlags *flags.WslFlags,
	defaultEnvironmentVariables *map[string]string) (err error) {
	defer decorate.OnError(&err, "WslGetDistributionConfiguration")

	distroUTF16, err := syscall.UTF16PtrFromString(distributionName)
	if err != nil {
		return errors.New("could not convert distro name to UTF16")
	}

	var (
		envVarsBegin **char
		envVarsLen   uint64 // size_t
	)

	_, err = callDll(apiWslGetDistributionConfiguration,
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(unsafe.Pointer(distributionVersion)),
		uintptr(unsafe.Pointer(defaultUID)),
		uintptr(unsafe.Pointer(wslDistributionFlags)),
		uintptr(unsafe.Pointer(&envVarsBegin)),
		uintptr(unsafe.Pointer(&envVarsLen)),
	)

	if err != nil {
		return err
	}

	*defaultEnvironmentVariables = processEnvVariables(envVarsBegin, envVarsLen)
	return nil
}

// WslLaunchInteractive is a wrapper around the WslLaunchInteractive
// function in the wslApi.dll Win32 library.
func (Backend) WslLaunchInteractive(distributionName string, command string, useCurrentWorkingDirectory bool) (exitCode uint32, err error) {
	defer decorate.OnError(&err, "WslLaunchInteractive")

	exitCode = math.MaxUint32
	distroUTF16, err := syscall.UTF16PtrFromString(distributionName)
	if err != nil {
		return exitCode, errors.New("could not convert distro name to UTF16")
	}

	commandUTF16, err := syscall.UTF16PtrFromString(command)
	if err != nil {
		return exitCode, errors.New("could not convert command to UTF16")
	}

	var useCwd wBOOL
	if useCurrentWorkingDirectory {
		useCwd = 1
	}

	r, err := callDll(apiWslLaunchInteractive,
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(unsafe.Pointer(commandUTF16)),
		uintptr(useCwd),
		uintptr(unsafe.Pointer(&exitCode)))

	if err != nil {
		return r, err
	}

	return exitCode, nil
}

// WslRegisterDistribution is a wrapper around the WslRegisterDistribution
// function in the wslApi.dll Win32 library.
func (Backend) WslRegisterDistribution(distributionName string, tarGzFilename string) (err error) {
	defer decorate.OnError(&err, "WslRegisterDistribution")

	distroUTF16, err := syscall.UTF16PtrFromString(distributionName)
	if err != nil {
		return errors.New("could not convert distro name to UTF16")
	}

	tarGzFilenameUTF16, err := syscall.UTF16PtrFromString(tarGzFilename)
	if err != nil {
		return errors.New("could not convert rootfs path to UTF16")
	}

	_, err = callDll(apiWslRegisterDistribution,
		uintptr(unsafe.Pointer(distroUTF16)),
		uintptr(unsafe.Pointer(tarGzFilenameUTF16)))

	if err != nil {
		return err
	}

	return nil
}

// WslUnregisterDistribution is a wrapper around the WslUnregisterDistribution
// function in the wslApi.dll Win32 library.
func (Backend) WslUnregisterDistribution(distributionName string) (err error) {
	defer decorate.OnError(&err, "WslUnregisterDistribution")

	distroUTF16, err := syscall.UTF16PtrFromString(distributionName)
	if err != nil {
		return errors.New("could not convert distro name to UTF16")
	}

	_, err = callDll(apiWslUnregisterDistribution, uintptr(unsafe.Pointer(distroUTF16)))
	if err != nil {
		return err
	}

	return nil
}

func callDll(proc *syscall.LazyProc, args ...uintptr) (uint32, error) {
	if err := proc.Find(); err != nil {
		return 0, err
	}

	r, _, err := proc.Call(args...)
	if r == 0 {
		return uint32(r), nil
	}
	if err == nil {
		return uint32(r), fmt.Errorf("failed syscall: exit code %d", r)
	}
	return uint32(r), fmt.Errorf("failed syscall: exit code %d: %v", r, err)
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
