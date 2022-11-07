package WslApi

// Distro is an abstraction around a WSL instance.
type Distro struct {
	Name string
}

// Windows' typedefs
type wBOOL = int       // Windows' BOOL
type wULONG = uint32   // Windows' ULONG
type ExitCode = uint32 // Windows' DWORD
type char byte         // Windows' CHAR (which is the same as C's char)

// Windows' constants
const (
	WindowsError  ExitCode = 4294967295 // Underflowed -1
	ActiveProcess ExitCode = 259
)

// Windows' WSL_DISTRIBUTION_FLAGS
// https://learn.microsoft.com/en-us/windows/win32/api/wslapi/ne-wslapi-wsl_distribution_flags
type wslFlags int

const (
	fNONE                  wslFlags = 0x0
	fENABLE_INTEROP        wslFlags = 0x1
	fAPPEND_NT_PATH        wslFlags = 0x2
	fENABLE_DRIVE_MOUNTING wslFlags = 0x4

	// Per conversation at https://github.com/microsoft/WSL-DistroLauncher/issues/96
	// the information about version 1 or 2 is on the 4th bit of the distro flags, which is not
	// currently referenced by the API nor docs.
	fUNDOCUMENTED_WSL_VERSION wslFlags = 0x8
)
