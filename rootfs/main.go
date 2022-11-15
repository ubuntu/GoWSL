package rootfs

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"wsl/rootfs/mutex"

	"github.com/0xrawsec/golang-utils/log"
	"github.com/cavaliercoder/grab"
)

type Release uint8

const (
	Jammy Release = iota
	Kinetic
	nReleases
)

func (r Release) String() string {
	switch r {
	case Jammy:
		return `jammy`
	case Kinetic:
		return `kinetic`
	}
	return "undefined-release"
}

func Get(r Release, location string) (string, error) {
	if err := validateRelease(r); err != nil {
		return "", err
	}

	endSection, err := protectedSection(location)
	if err != nil {
		return "", nil
	}
	defer endSection()

	if !isLocalOutdated(r, location) {
		return localRootfs(r, location), nil
	}

	removeBackup, restoreBackup, err := stashLocal(r, location)
	if err != nil {
		return localRootfs(r, location), err
	}

	rootfs, err := fetchRemote(r, location)
	if err != nil {
		if err := restoreBackup(); err != nil {
			log.Warnf("%v", err)
		}
		return rootfs, err
	}
	removeBackup()
	return rootfs, nil
}

// protectedSection creates an inter-process mutex and locks onto it. It returns a function to release it.
func protectedSection(location string) (func(), error) {
	noop := func() {}

	// Creating / opening mutex
	cleanName := strings.ReplaceAll(strings.ReplaceAll("rootfs-at-"+location, string(os.PathSeparator), "__"), ":", "")
	mutex, err := mutex.New(cleanName)
	if err != nil {
		return noop, err
	}

	// Waiting for Mutex
	ticker := time.NewTicker(60 * time.Second)
	done := make(chan error)
	go func() {
		done <- mutex.Lock()
	}()

	select {
	case <-ticker.C:
		mutex.Close()
		return noop, fmt.Errorf("timed out while waiting for mutex")
	case err = <-done:
	}

	if err != nil {
		return noop, err
	}

	// Cleanup function
	cleanup := func() {
		mutex.Release()
		mutex.Close()
	}
	return cleanup, nil
}

// validateRelease validates that the passed value matches a value in the release enum.
// Should be used at every API endpoint
func validateRelease(r Release) error {
	if r >= nReleases {
		return fmt.Errorf("release with enum value %d does not exist", r)
	}
	return nil
}

// isLocalOutdated checks if the rootfs held localy is the same than in
// the remote. Returns true if no local rootfs exists.
func isLocalOutdated(r Release, location string) bool {
	// Rootfs is missing locally
	if _, err := safeLocalRootfs(r, location); err != nil {
		return true
	}
	// SHA256SUMS is missing locally
	localChecksum, err := localSha256Contents(r, location)
	if err != nil {
		return true
	}
	// Failed to fetch remote SHA256SUMS: assume outdated
	remoteChecksum, err := remoteSha256Contents(r)
	if err != nil {
		log.Warnf("Failed to fetch remote SHA256SUMS. Assuming local is outated.\n")
		return true
	}
	// It's outdated if the checksums are the same
	return localChecksum != remoteChecksum
}

func safeGetPath(path string) (string, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return path, fmt.Errorf("file %s does not exist", path)
	}
	return path, nil
}

func safeLocalRootfs(r Release, location string) (string, error) {
	return safeGetPath(localRootfs(r, location))
}

func localRootfs(r Release, location string) string {
	return filepath.Join(location, localTarballName(r))
}

func localSha256Contents(r Release, location string) (string, error) {
	return parseSHA256SUMS(r, localSha256(r, location))
}

func localSha256(r Release, location string) string {
	return filepath.Join(location, fmt.Sprintf(`%s.SHA256SUMS`, r))
}

func baseURL(r Release) string {
	switch r {
	case Jammy:
		return `https://cloud-images.ubuntu.com/wsl/jammy/current/`
	case Kinetic:
		return `https://cloud-images.ubuntu.com/wsl/kinetic/current/`
	}

	panic("Unreachable")
}

func remoteTarballName(r Release) string {
	return fmt.Sprintf("ubuntu-%s-wsl-amd64-wsl.rootfs.tar.gz", r)
}

func localTarballName(r Release) string {
	return fmt.Sprintf("%s.tar.gz", r)
}

var downloadFile = func(dir string, url string) (string, error) {
	r, err := grab.Get(dir, url)
	return r.Filename, err
}

func remoteSha256Contents(r Release) (string, error) {
	url := baseURL(r) + `SHA256SUMS`
	fileName, err := downloadFile(os.TempDir(), url)
	if err != nil {
		return "", err
	}
	return parseSHA256SUMS(r, fileName)
}

func parseSHA256SUMS(r Release, path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	targzName := remoteTarballName(r)
	for scanner.Scan() {
		data := strings.Split(scanner.Text(), "  ")
		if len(data) != 2 {
			continue
		}
		if data[1] != targzName {
			continue
		}
		return data[0], nil
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("could not find %q in SHA256SUMS file %q", targzName, path)
}

func stashLocal(r Release, location string) (remove func() error, restore func() error, err error) {
	nilFunc := func() error { return nil }

	removeSHA, restoreSHA, err := stashSingleFile(localSha256(r, location))
	if err != nil {
		return nilFunc, nilFunc, err
	}

	removeRFS, restoreRFS, err := stashSingleFile(localRootfs(r, location))
	if err != nil {
		return nilFunc, nilFunc, compose(restoreRFS, wrap(err))()
	}

	return compose(removeRFS, removeSHA), compose(restoreRFS, restoreSHA), nil
}

func stashSingleFile(original string) (remove func() error, restore func() error, err error) {
	nilFunc := func() error { return nil }

	if _, err := safeGetPath(original); err != nil {
		return nilFunc, nilFunc, nil // No need to back it up beacuse it doesn't exist
	}

	backup := original + ".backup"
	err = os.Rename(original, backup)
	if err != nil {
		return nilFunc, nilFunc, fmt.Errorf("could not back up current %q as %q: %v", original, backup, err)
	}

	// Gets rid of the backup
	remove = func() error {
		err := os.Remove(backup)
		if err != nil {
			return fmt.Errorf("could not remove backup at %q: %v", backup, err)
		}
		return nil
	}

	// Restores the backup
	restore = func() error {
		err := os.Rename(backup, original)
		if err != nil {
			return fmt.Errorf("could not restore backup at %q: %v", backup, err)
		}
		return nil
	}

	return remove, restore, nil
}

func fetchRemote(r Release, location string) (string, error) {
	// Fetching rootfs
	url := baseURL(r) + remoteTarballName(r)
	_, err := downloadFile(localRootfs(r, location), url)
	if err != nil {
		return "", fmt.Errorf("could not fetch remote rootfs at %q: %v", url, err)
	}

	// Fetching SHA256SUMS
	url = baseURL(r) + `SHA256SUMS`
	_, err = downloadFile(localSha256(r, location), url)
	if err != nil {
		return "", fmt.Errorf("could not fetch remote SHA256SUMS at %q: %v", url, err)
	}

	return safeLocalRootfs(r, location)
}

func compose(A func() error, B func() error) func() error {
	return func() error {
		errA := A()
		errB := B()

		if errA == nil && errB == nil {
			return nil
		}
		if errA != nil && errB == nil {
			return errA
		}

		if errA == nil && errB != nil {
			return errB
		}

		return fmt.Errorf(`two errors triggered:
ERROR 1:
%v

ERROR 2:
%v
`, errA, errB)
	}
}

func wrap[T any](x T) func() T {
	return func() T { return x }
}
