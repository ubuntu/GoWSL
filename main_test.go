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
	emptyRootFs = `images/empty.tar.gz`  // Empty non-functional image. It registers instantly.
	rootFs      = `images/rootfs.tar.gz` // Fully functional rootfs
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	if wsl.MockAvailable() {
		ctx = wsl.WithMock(ctx, mock.New())

		// Touch rootfs so that tests work.
		// For the real tests, .\prepare-repository.ps1 should be ran before running the tests
		tmp := filepath.Join(os.TempDir(), filepath.Dir(rootFs))
		if err := os.MkdirAll(tmp, 0700); err != nil {
			log.Fatalf("Setup: could not create images dir: %v", err)
		}
		defer os.RemoveAll(tmp)

		rootFs = filepath.Join(tmp, rootFs)
		emptyRootFs = filepath.Join(tmp, emptyRootFs)

		for _, fname := range []string{rootFs, emptyRootFs} {
			f, err := os.OpenFile(fname, os.O_RDONLY|os.O_CREATE, 0600)
			if err != nil {
				log.Fatalf("Setup: could not touch rootfs %q: %v", fname, err)
			}
			f.Close()
		}
	}

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
