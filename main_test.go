package gowsl_test

// This file contains testing functionality

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"testing"

	wsl "github.com/ubuntu/gowsl"
	"github.com/ubuntu/gowsl/mock"
)

const (
	namePrefix string = "wsltesting"
)

var (
	emptyRootFS string // Empty non-functional image. It registers instantly.
	rootFS      string // Fully functional rootfs
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	if wsl.MockAvailable() {
		ctx = wsl.WithMock(ctx, mock.New())
	}

	defer setUpRootFS()()

	// In case a previous run was interrupted
	cleanUpTestWslInstances(ctx)

	restore, err := backUpDefaultDistro(ctx)
	if err != nil {
		log.Fatalf("Setup: %v", err)
	}
	defer restore()

	exitVal := m.Run()

	err = wsl.Shutdown(ctx)
	if err != nil {
		log.Println("Cleanup: Failed to shutdown WSL")
	}

	cleanUpTestWslInstances(ctx)

	os.Exit(exitVal)
}

// setUpRootFS sets the rootFS and emptyRootFS variables, and optionally creates the files.
//
// - For the real tests, .\prepare-repository.ps1 should be ran before running the tests.
// - For the mocked tests, emporary empty files are created to stand for the rootFS.
func setUpRootFS() (cleanup func()) {
	if !wsl.MockAvailable() {
		rootFS = "images/rootfs.tar.gz"
		emptyRootFS = "images/empty.tar.gz"
		return func() {}
	}

	rootFSDir, err := os.MkdirTemp(os.TempDir(), "GoWSL")
	if err != nil {
		log.Fatalf("Setup: could not create images dir: %v", err)
	}

	rootFS = filepath.Join(rootFSDir, "empty1.tar.gz")
	emptyRootFS = filepath.Join(rootFSDir, "empty2.tar.gz")

	for _, fname := range []string{rootFS, emptyRootFS} {
		f, err := os.Create(fname)
		if err != nil {
			log.Fatalf("Setup: could not touch rootfs %q: %v", fname, err)
		}
		f.Close()
	}

	return func() {
		os.RemoveAll(rootFSDir)
	}
}
