// This file contains windows-specific typedefs and constants
package WslApi

// Windows' typedefs
type winBOOL = int     // Windows' BOOL
type winULONG = uint32 // Windows' ULONG
type ExitCode = uint32 // Windows' DWORD
type char byte         // Windows' CHAR (which is the same as C's char)


// Windows' constants
const (
	WindowsError  ExitCode = 4294967295 // Underflowed -1
	ActiveProcess ExitCode = 259
)

// Windows' WSL_DISTRIBUTION_FLAGS
// https://learn.microsoft.com/en-us/windows/win32/api/wslapi/ne-wslapi-wsl_distribution_flags
type winWslFlags int

const (
	flagNONE                  winWslFlags = 0x0
	flagENABLE_INTEROP        winWslFlags = 0x1
	flagAPPEND_NT_PATH        winWslFlags = 0x2
	flagENABLE_DRIVE_MOUNTING winWslFlags = 0x4

	// Per conversation at https://github.com/microsoft/WSL-DistroLauncher/issues/96
	// the information about version 1 or 2 is on the 4th bit of the distro flags, which is not
	// currently referenced by the API nor docs.
	flagUNDOCUMENTED_WSL_VERSION winWslFlags = 0x8
)
