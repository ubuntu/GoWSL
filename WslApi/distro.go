package WslApi

import "syscall"

var (
	wslApiDll = syscall.NewLazyDLL("wslapi.dll")
)

// Distro is an abstraction around a WSL instance.
type Distro struct {
	Name string
}
