package gowsl_test

// This file contains testing functionality

import (
	"context"
	"os"
	"testing"

	"github.com/0xrawsec/golang-utils/log"
	wsl "github.com/ubuntu/gowsl"
	"github.com/ubuntu/gowsl/mock"
)

const (
	namePrefix  string = "wsltesting"
	emptyRootFs string = `./images/empty.tar.gz`  // Empty non-functional image. It registers instantly.
	rootFs      string = `./images/rootfs.tar.gz` // Fully functional rootfs
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	if wsl.MockAvailable() {
		ctx = wsl.WithMock(ctx, mock.New())
	}

	// In case a previous run was interrupted
	cleanUpTestWslInstances(ctx)

	restore, err := backUpDefaultDistro(ctx)
	if err != nil {
		log.Errorf("setup: %v", err)
		os.Exit(1)
	}
	defer restore()

	exitVal := m.Run()

	err = wsl.Shutdown(ctx)
	if err != nil {
		log.Warnf("cleanup: Failed to shutdown WSL")
	}

	cleanUpTestWslInstances(ctx)

	os.Exit(exitVal)
}
