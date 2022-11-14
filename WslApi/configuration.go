package WslApi

import (
	"fmt"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

// Configuration is the configuration of the instance.
type Configuration struct {
	Version                     uint8             // Type of filesystem used (lxfs vs. wslfs, relevnt only to WSL1)
	DefaultUID                  uint32            // User ID of default user
	InteropEnabled              bool              // Whether interop with windows is enabled
	PathAppended                bool              // Whether Windows paths are appended
	DriveMountingEnabled        bool              // Whether drive mounting is enabled
	undocumentedWSLVersion      uint8             // Undocumented variable. WSL1 vs. WSL2.
	DefaultEnvironmentVariables map[string]string // Environment variables passed to the instance by default
}

// Configure is a wrapper around Win32's WslConfigureDistribution.
// Note that only the following config is mutable:
//  - DefaultUID
//  - InteropEnabled
//  - PathAppended
//  - DriveMountingEnabled
func (i *Instance) Configure(config Configuration) error {

	instanceUTF16, err := syscall.UTF16PtrFromString(i.Name)
	if err != nil {
		return fmt.Errorf("failed to convert '%s' to UTF16", i.Name)
	}

	flags, err := config.packFlags()
	if err != nil {
		return err
	}

	r1, _, _ := wslConfigureDistribution.Call(
		uintptr(unsafe.Pointer(instanceUTF16)),
		uintptr(config.DefaultUID),
		uintptr(flags),
	)

	if r1 != 0 {
		return fmt.Errorf("failed syscall to WslConfigureDistribution")
	}

	return nil
}

// GetConfiguration is a wrapper around Win32's WslGetDistributionConfiguration.
func (i Instance) GetConfiguration() (Configuration, error) {
	var conf Configuration

	instanceUTF16, err := syscall.UTF16PtrFromString(i.Name)
	if err != nil {
		return conf, fmt.Errorf("failed to convert '%s' to UTF16", i.Name)
	}

	var (
		flags        wslFlags
		envVarsBegin **char
		envVarsLen   uint64 // size_t
	)

	r1, _, _ := wslGetDistributionConfiguration.Call(
		uintptr(unsafe.Pointer(instanceUTF16)),
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

// String deserializes a Configuration object as a yaml string
func (conf Configuration) String() string {
	fmtEnvs := []string{}
	for k, v := range conf.DefaultEnvironmentVariables {
		fmtEnvs = append(fmtEnvs, fmt.Sprintf(`    - %s: %s`, k, v))
	}

	envJSON := ""
	if len(fmtEnvs) != 0 {
		envJSON = fmt.Sprintf("\n%s\n", strings.Join(fmtEnvs, "\n"))
	}

	return fmt.Sprintf(`configuration:
  - Version: %d
  - DefaultUID: %d
  - InteropEnabled: %t
  - PathAppended: %t
  - DriveMountingEnabled: %t
  - undocumentedWSLVersion: %d
  - DefaultEnvironmentVariables:%s
`, conf.Version, conf.DefaultUID, conf.InteropEnabled, conf.PathAppended, conf.DriveMountingEnabled, conf.undocumentedWSLVersion, envJSON)
}

// unpackFlags examines a winWslFlags object and stores its findings in the Configuration
func (conf *Configuration) unpackFlags(flags wslFlags) {
	conf.InteropEnabled = false
	if flags&fENABLE_INTEROP != 0 {
		conf.InteropEnabled = true
	}

	conf.PathAppended = false
	if flags&fAPPEND_NT_PATH != 0 {
		conf.PathAppended = true
	}

	conf.DriveMountingEnabled = false
	if flags&fENABLE_DRIVE_MOUNTING != 0 {
		conf.DriveMountingEnabled = true
	}

	conf.undocumentedWSLVersion = 1
	if flags&fUNDOCUMENTED_WSL_VERSION != 0 {
		conf.undocumentedWSLVersion = 2
	}
}

// packFlags generates a winWslFlags object from the Configuration
func (conf Configuration) packFlags() (wslFlags, error) {
	flags := fNONE

	if conf.InteropEnabled {
		flags = flags | fENABLE_INTEROP
	}

	if conf.PathAppended {
		flags = flags | fAPPEND_NT_PATH
	}

	if conf.DriveMountingEnabled {
		flags = flags | fENABLE_DRIVE_MOUNTING
	}

	switch conf.undocumentedWSLVersion {
	case 1:
	case 2:
		flags = flags | fUNDOCUMENTED_WSL_VERSION
	default:
		return flags, fmt.Errorf("unknown WSL version %d", conf.undocumentedWSLVersion)
	}

	return flags, nil
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
			goStr := stringCtoGo(cStr, 32768)
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
// maxlen is the max distance that will searched. It is meant to mitigate buffer overflow.
func stringCtoGo(cString *char, maxlen uint64) (goString string) {
	size := strnlen(cString, maxlen)
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
