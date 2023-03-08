//go:build !gowslmock

package gowsl

import (
	"context"

	"github.com/ubuntu/gowsl/internal/backend"
	"github.com/ubuntu/gowsl/internal/backend/windows"
)

func selectBackend(ctx context.Context) backend.Backend {
	return windows.Backend{}
}
