package WslApi

import (
	"fmt"
	"strings"
)

// Configuration is the configuration of the distro.
type Configuration struct {
	Version                     uint8             // Type of filesystem used (lxfs vs. wslfs, relevnt only to WSL1)
	DefaultUID                  uint32            // User ID of default user
	InteropEnabled              bool              // Whether interop with windows is enabled
	PathAppended                bool              // Whether Windows paths are appended
	DriveMountingEnabled        bool              // Whether drive mounting is enabled
	undocumentedWSLVersion      uint8             // Undocumented variable. WSL1 vs. WSL2.
	DefaultEnvironmentVariables map[string]string // Environment variables passed to the distro by default
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
