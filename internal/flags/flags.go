// Package flags contains the enum used by WSL to display
// some configuration of a WSL distro.
package flags

// WslFlags is an alias for Windows' WSL_DISTRIBUTION_FLAGS
// https://learn.microsoft.com/en-us/windows/win32/api/wslapi/ne-wslapi-wsl_distribution_flags
type WslFlags int32

// Allowing underscores in names to keep it as close to Windows as possible.
const (
	// All nolints are regarding the use of UPPPER_CASE.

	flag_ENABLE_INTEROP        WslFlags = 0x1 //nolint:revive
	flag_APPEND_NT_PATH        WslFlags = 0x2 //nolint:revive
	flag_ENABLE_DRIVE_MOUNTING WslFlags = 0x4 //nolint:revive

	// Per the conversation at https://github.com/microsoft/WSL-DistroLauncher/issues/96
	// the information about version 1 or 2 is on the 4th bit of the flags, which is
	// currently referenced neither by the API nor the documentation.
	flag_undocumented_WSL_VERSION WslFlags = 0x8 //nolint:revive
)
