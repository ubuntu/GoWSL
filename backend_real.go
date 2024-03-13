//go:build !gowslmock

package gowsl

import (
	"context"

	"github.com/ubuntu/gowsl/internal/backend"
	"github.com/ubuntu/gowsl/internal/backend/windows"
	"github.com/ubuntu/gowsl/mock"
)

// MockAvailable indicates if the mock can be accessed at runtime.
// It is always accessible at compile-time to make writing tests easier, but you should use:
//
//	   ctx := context.Background()
//	   if wsl.MockAvailable() {
//		    m := mock.New()
//	        // set up the mock ...
//	        ctx = wsl.WithMock(ctx, m)
//	   }
func MockAvailable() bool {
	return false
}

// WithMock adds the mock back-end to the context when GoWSL has been compiled with the gowslmock tag.
// Otherwise, it panics.
func WithMock(ctx context.Context, m *mock.Backend) context.Context {
	panic("Cannot use mock without build flag gowslmock")
}

func selectBackend(ctx context.Context) backend.Backend {
	return windows.Backend{}
}

// ErrNotExist is the error returned when a distro does not exist.
var ErrNotExist = windows.ErrNotExist
