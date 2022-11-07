package WslApi

import (
	"fmt"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

// GetConfiguration is a wrapper around Win32's WslGetDistributionConfiguration.
func (distro Distro) GetConfiguration() (Configuration, error) {
	var conf Configuration

	distroNameUTF16, err := syscall.UTF16PtrFromString(distro.Name)
	if err != nil {
		return conf, fmt.Errorf("failed to convert '%s' to UTF16", distro.Name)
	}

	var (
		flags        wslFlags
		envVarsBegin = new(*char)
		envVarsLen   uint64 // size_t
	)

	r1, _, _ := wslGetDistributionConfiguration.Call(
		uintptr(unsafe.Pointer(distroNameUTF16)),
		uintptr(unsafe.Pointer(&conf.Version)),
		uintptr(unsafe.Pointer(&conf.DefaultUID)),
		uintptr(unsafe.Pointer(&flags)),
		uintptr(unsafe.Pointer(&envVarsBegin)),
		uintptr(unsafe.Pointer(&envVarsLen)),
	)

	if r1 != 0 {
		return conf, fmt.Errorf("failed syscall to WslGetDistributionConfiguration")
	}

	conf.unpackFlags(flags)
	conf.DefaultEnvironmentVariables = processEnvVariables(envVarsBegin, envVarsLen)
	return conf, nil
}

// processEnvVariables takes the **char and length obtained from Win32's API and returs a
// map[variableName]variableValue
func processEnvVariables(cStringArray **char, len uint64) map[string]string {
	stringPtrs := unsafe.Slice(cStringArray, len)

	keys := make(chan string)
	values := make(chan string)

	wg := sync.WaitGroup{}
	for _, cStr := range stringPtrs {
		cStr := cStr
		wg.Add(1)
		go func() {
			defer wg.Done()
			goStr := stringCtoGo(cStr)
			idx := strings.Index(goStr, "=")
			keys <- goStr[:idx]
			values <- goStr[idx+1:]
		}()
	}

	go func() {
		defer close(keys)
		defer close(values)
		wg.Wait()
	}()

	// Collecting results
	m := map[string]string{}

	k, okk := <-keys
	v, okv := <-values
	for okk && okv {
		m[k] = v

		k, okk = <-keys
		v, okv = <-values
	}

	return m
}

// stringCtoGo converts a null-terminated *char into a string
func stringCtoGo(cString *char) (goString string) {
	size := strnlen(cString, 32768)
	return string(unsafe.Slice(cString, size))
}

// strnlen finds the null terminator to determine *char length.
// The null terminator itself is not counted towards the length.
// maxlen is the max distance that will searched. It is meant to mitigate buffer overflow.
func strnlen(ptr *char, maxlen uint64) (length uint64) {
	length = 0
	for ; *ptr != 0 && length <= maxlen; ptr = charNext(ptr) {
		length++
	}
	return length
}

// charNext advances *char by one position
func charNext(ptr *char) *char {
	return (*char)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + unsafe.Sizeof(char(0))))
}
