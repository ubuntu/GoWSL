package mock

// This file contains mocks for Win32 API definitions and imports.

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/ubuntu/decorate"
	"github.com/ubuntu/gowsl/internal/flags"
	"github.com/ubuntu/gowsl/mock/internal/distrostate"
)

// When something fails Windows-side, it returns (1<<32 - 1).
const windowsError = math.MaxUint32

// WslConfigureDistribution mocks the WslConfigureDistribution call to the Win32 API.
func (b *Backend) WslConfigureDistribution(distributionName string, defaultUID uint32, wslDistributionFlags flags.WslFlags) (err error) {
	defer decorate.OnError(&err, "WslConfigureDistribution")

	if b.WslConfigureDistributionError {
		return Error{}
	}

	if err := validDistroName(distributionName); err != nil {
		return err
	}

	_, key := b.findDistroKey(distributionName)
	if key == nil {
		return errors.New("failed syscall: distro not registered")
	}

	key.mu.Lock()
	defer key.mu.Unlock()

	key.data["Flags"] = wslDistributionFlags
	key.data["DefaultUid"] = defaultUID

	return nil
}

// WslGetDistributionConfiguration mocks the WslGetDistributionConfiguration call to the Win32 API.
func (b *Backend) WslGetDistributionConfiguration(distributionName string,
	distributionVersion *uint8,
	defaultUID *uint32,
	wslDistributionFlags *flags.WslFlags,
	defaultEnvironmentVariables *map[string]string) (err error) {
	defer decorate.OnError(&err, "WslGetDistributionConfiguration")

	if b.WslGetDistributionConfigurationError {
		return Error{}
	}

	if err := validDistroName(distributionName); err != nil {
		return err
	}

	b.lxssRootKey.mu.RLock()
	defer b.lxssRootKey.mu.RUnlock()

	_, key := b.findDistroKey(distributionName)
	if key == nil {
		return errors.New("failed syscall: not registered")
	}

	key.mu.RLock()
	defer key.mu.RUnlock()

	// Ignoring tipe assert linter because we're the only ones with access to these fields
	*distributionVersion = key.data["Version"].(uint8)         //nolint: forcetypeassert
	*defaultUID = key.data["DefaultUid"].(uint32)              //nolint: forcetypeassert
	*wslDistributionFlags = key.data["Flags"].(flags.WslFlags) //nolint: forcetypeassert

	*defaultEnvironmentVariables = map[string]string{
		"HOSTTYPE": "x86_64",
		"LANG":     "en_US.UTF-8",
		"PATH":     "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games",
		"TERM":     "xterm-256color",
	}

	return nil
}

// WslLaunch mocks the WslLaunch call to the Win32 API.
func (b *Backend) WslLaunch(distributionName string,
	command string,
	useCWD bool,
	stdin *os.File,
	stdout *os.File,
	stderr *os.File) (process *os.Process, err error) {
	defer decorate.OnError(&err, "WslLaunch")

	if b.WslLaunchError {
		return nil, Error{}
	}

	if err := validWin32String(distributionName); err != nil {
		return nil, err
	}

	if err := validWin32String(command); err != nil {
		return nil, err
	}

	b.lxssRootKey.mu.RLock()

	_, distroKey := b.findDistroKey(distributionName)
	if distroKey == nil {
		b.lxssRootKey.mu.RUnlock()
		return nil, errors.New("Failed syscall: distro does not exist")
	}

	b.lxssRootKey.mu.RUnlock()

	if !isPipeOrNull(stdin) {
		panic("Stdin must be a pipe")
	}
	if !isPipeOrNull(stdout) {
		panic("Stdout must be a pipe")
	}
	if !isPipeOrNull(stderr) {
		panic("Stderr must be a pipe")
	}

	p, err := newMockedCommand(command).start(stdin, stdout, stderr)
	if err != nil {
		return nil, err
	}

	if err := distroKey.state.AttachProcess(p); err != nil {
		_ = p.Kill()
		return nil, err
	}

	return p, nil
}

// WslLaunchInteractive mocks the WslLaunchInteractive call to the Win32 API.
func (b *Backend) WslLaunchInteractive(distributionName string, command string, useCurrentWorkingDirectory bool) (exitCode uint32, err error) {
	defer decorate.OnError(&err, "WslLaunchInteractive")

	if b.WslLaunchInteractiveError {
		return windowsError, Error{}
	}

	if err := validWin32String(distributionName); err != nil {
		return windowsError, err
	}

	if err := validWin32String(command); err != nil {
		return windowsError, err
	}

	b.lxssRootKey.mu.RLock()

	_, distroKey := b.findDistroKey(distributionName)
	if distroKey == nil {
		b.lxssRootKey.mu.RUnlock()
		return windowsError, errors.New("failed syscall: distro not found")
	}

	b.lxssRootKey.mu.RUnlock()

	if err := distroKey.state.Touch(); err != nil {
		return windowsError, fmt.Errorf("failed syscall: %v", err)
	}

	switch command {
	case "":
		s, err := distroKey.state.NewShell()
		if err != nil {
			return windowsError, err
		}
		exit := s.Wait()

		return exit, nil
	case "exit 0":
		return 0, nil
	case "exit 42":
		return 42, nil
	case "[ `pwd` = /root ]":
		if useCurrentWorkingDirectory {
			// We are wherever wsl.exe was called from
			return 1, nil
		}
		// We are home (hence /root)
		return 0, nil
	case "[ `pwd` != /root ]":
		if useCurrentWorkingDirectory {
			// We are wherever wsl.exe was called from
			return 0, nil
		}
		// We are home (hence /root)
		return 1, nil
	default:
		panic(fmt.Sprintf("WslLaunchInteractive command not supported: %q", command))
	}
}

// WslRegisterDistribution mocks the WslRegisterDistribution call to the Win32 API.
func (b *Backend) WslRegisterDistribution(distributionName string, tarGzFilename string) (err error) {
	defer decorate.OnError(&err, "WslRegisterDistribution")

	if b.WslRegisterDistributionError {
		return Error{}
	}

	if err := validDistroName(distributionName); err != nil {
		return err
	}

	if err := validWin32String(tarGzFilename); err != nil {
		return err
	}

	b.lxssRootKey.mu.Lock()
	defer b.lxssRootKey.mu.Unlock()

	if _, key := b.findDistroKey(distributionName); key != nil {
		return errors.New("failed syscall: distro already exists")
	}

	GUID, err := uuid.NewRandom()
	if err != nil {
		return fmt.Errorf("could not generate UUID: %v", err)
	}

	guidStr := fmt.Sprintf("{%s}", GUID.String())

	b.lxssRootKey.children[guidStr] = &RegistryKey{
		path: filepath.Join("HKEY_CURRENT_USER", lxssPath, guidStr),
		data: map[string]any{
			"DistributionName": distributionName,
			"Flags":            flags.WslFlags(0xf),
			"Version":          uint8(2),
			"DefaultUid":       uint32(0),
		},
		state: distrostate.New(),
	}

	// When registering the first distro, DefaultDistribution
	// is updated with its GUID

	if b.lxssRootKey.data["DefaultDistribution"] == "" {
		b.lxssRootKey.data["DefaultDistribution"] = guidStr
	}

	return nil
}

// WslUnregisterDistribution mocks the WslUnregisterDistribution call to the Win32 API.
func (b *Backend) WslUnregisterDistribution(distributionName string) (err error) {
	defer decorate.OnError(&err, "WslUnregisterDistribution")

	if b.WslUnregisterDistributionError {
		return Error{}
	}

	if err := validDistroName(distributionName); err != nil {
		return err
	}

	b.lxssRootKey.mu.Lock()
	defer b.lxssRootKey.mu.Unlock()

	GUID, key := b.findDistroKey(distributionName)
	if key == nil {
		return errors.New("failed syscall: distro not registered")
	}

	err = key.state.MarkUninstalled()
	delete(b.lxssRootKey.children, GUID)

	//  When you unregister the default distro, the one with the lowest GUID
	// (lexicographically) is set as default. If there are none, the field is
	// set to empty string.

	if b.lxssRootKey.data["DefaultDistribution"] != GUID {
		return nil
	}

	var firstGUID string
	for GUID := range b.lxssRootKey.children {
		if _, err := uuid.Parse(GUID); err != nil {
			continue // Not a distro
		}

		if strings.Compare(GUID, firstGUID) == -1 {
			firstGUID = GUID
		}
	}

	b.lxssRootKey.data["DefaultDistribution"] = firstGUID

	return err
}

func validWin32String(str string) error {
	if strings.ContainsRune(str, rune(0)) {
		return fmt.Errorf("could not convert %q to UTF-16", str)
	}
	return nil
}

func validDistroName(distroName string) error {
	if err := validWin32String(distroName); err != nil {
		return err
	}

	p := regexp.MustCompile(`^[A-Za-z0-9-_\.]+$`)
	if !p.MatchString(distroName) {
		return errors.New("name contains invalid characters")
	}

	return nil
}

func (b *Backend) findDistroKey(distroName string) (GUID string, key *RegistryKey) {
	for GUID, key := range b.lxssRootKey.children {
		if _, err := uuid.Parse(GUID); err != nil {
			continue // Not a distro
		}

		name, ok := key.data["DistributionName"]
		if !ok {
			continue
		}
		if name != distroName {
			continue
		}

		return GUID, key
	}

	return "", nil
}

// isPipeOrNull checks whether f is a pipe or a null device
// These are the types of file WslLaunch will write on.
func isPipeOrNull(f *os.File) bool {
	// Option 1: it is a pipe
	fStat, err := f.Stat()
	if err != nil {
		panic(fmt.Sprintf("Could not stat %q: %v", f.Name(), err))
	}

	if fStat.Mode()&os.ModeNamedPipe != 0 {
		return true
	}

	// Option 2: it is the null device
	null, err := os.Open(os.DevNull)
	if err != nil {
		panic(fmt.Sprintf("Failed call to open null device: %v", err))
	}

	nullStat, err := null.Stat()
	if err != nil {
		panic(fmt.Sprintf("Could not stat null device: %v", err))
	}

	if os.SameFile(fStat, nullStat) {
		return true
	}

	return false
}
