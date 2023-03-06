package flags

import "fmt"

// Unpacked contains the same information as WslFlags but in a struct instead of an integer.
type Unpacked struct {
	InteropEnabled         bool  // Whether interop with windows is enabled
	PathAppended           bool  // Whether Windows paths are appended
	DriveMountingEnabled   bool  // Whether drive mounting is enabled
	UndocumentedWSLVersion uint8 // Undocumented variable. WSL1 vs. WSL2.
}

// Unpack examines a WslFlags object and stores its data in a Unpacked flags struct.
func Unpack(f WslFlags) Unpacked {
	var up Unpacked

	up.InteropEnabled = false
	if f&flag_ENABLE_INTEROP != 0 {
		up.InteropEnabled = true
	}

	up.PathAppended = false
	if f&flag_APPEND_NT_PATH != 0 {
		up.PathAppended = true
	}

	up.DriveMountingEnabled = false
	if f&flag_ENABLE_DRIVE_MOUNTING != 0 {
		up.DriveMountingEnabled = true
	}

	up.UndocumentedWSLVersion = 1
	if f&flag_undocumented_WSL_VERSION != 0 {
		up.UndocumentedWSLVersion = 2
	}

	return up
}

// Pack generates a WslFlags object from the Unpacked struct.
func (conf Unpacked) Pack() (WslFlags, error) {
	var f WslFlags

	if conf.InteropEnabled {
		f = f | flag_ENABLE_INTEROP
	}

	if conf.PathAppended {
		f = f | flag_APPEND_NT_PATH
	}

	if conf.DriveMountingEnabled {
		f = f | flag_ENABLE_DRIVE_MOUNTING
	}

	switch conf.UndocumentedWSLVersion {
	case 1:
	case 2:
		f = f | flag_undocumented_WSL_VERSION
	default:
		return f, fmt.Errorf("unknown WSL version %d", conf.UndocumentedWSLVersion)
	}

	return f, nil
}
