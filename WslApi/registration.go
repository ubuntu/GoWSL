package WslApi

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Register is a wrapper around Win32's WslRegisterDistribution
func (instance Instance) Register(rootFsPath string) error {

	r, err := instance.IsRegistered()
	if err != nil {
		return fmt.Errorf("failed to detect if '%s' is installed already", instance.Name)
	}
	if r {
		return fmt.Errorf("WSL instance '%s' is already registered", instance.Name)
	}

	instanceUTF16, err := syscall.UTF16PtrFromString(instance.Name)
	if err != nil {
		return fmt.Errorf("failed to convert '%s' to UTF16", instance.Name)
	}

	rootFsPathUTF16, err := syscall.UTF16PtrFromString(rootFsPath)
	if err != nil {
		return fmt.Errorf("failed to convert '%s' to UTF16", instance.Name)
	}

	r1, _, _ := wslRegisterDistribution.Call(
		uintptr(unsafe.Pointer(instanceUTF16)),
		uintptr(unsafe.Pointer(rootFsPathUTF16)))

	if r1 != 0 {
		return fmt.Errorf("failed syscall to wslRegisterDistribution")
	}

	registered, err := instance.IsRegistered()
	if err != nil {
		return err
	}
	if !registered {
		return fmt.Errorf("WSL instance %s was not succesfully registered", instance.Name)
	}

	return nil
}

// RegisteredIntances returns a slice of the registered instances
func RegisteredIntances() ([]Instance, error) {
	return registeredInstances()
}

// IsRegistered returns whether an instance is registered in WSL or not.
func (target Instance) IsRegistered() (bool, error) {
	instances, err := RegisteredIntances()
	if err != nil {
		return false, err
	}

	for _, i := range instances {
		if i.Name != target.Name {
			continue
		}
		return true, nil
	}
	return false, nil
}

// Register is a wrapper around Win32's WslUnregisterDistribution.
func (instance *Instance) Unregister() error {
	r, err := instance.IsRegistered()
	if err != nil {
		return fmt.Errorf("failed to detect if '%s' is installed already", instance.Name)
	}
	if !r {
		return fmt.Errorf("WSL instance '%s' is not registered", instance.Name)
	}

	instanceUTF16, err := syscall.UTF16PtrFromString(instance.Name)
	if err != nil {
		return fmt.Errorf("failed to convert '%s' to UTF16", instance.Name)
	}

	r1, _, _ := wslUnregisterDistribution.Call(uintptr(unsafe.Pointer(instanceUTF16)))

	if r1 != 0 {
		return fmt.Errorf("failed syscall to WslLaunchInteractive")
	}
	return nil
}
