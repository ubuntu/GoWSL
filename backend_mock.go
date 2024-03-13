//go:build gowslmock

package gowsl

import (
	"context"

	"github.com/ubuntu/gowsl/internal/backend"
	"github.com/ubuntu/gowsl/internal/backend/windows"
	"github.com/ubuntu/gowsl/mock"
)

type backendQueryType int

const backendQuery backendQueryType = 0

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
	return true
}

// WithMock adds the mock back-end to the context when GoWSL has been compiled with the gowslmock tag.
// Otherwise, it panics.
func WithMock(ctx context.Context, m *mock.Backend) context.Context {
	return context.WithValue(ctx, backendQuery, m)
}

func selectBackend(ctx context.Context) backend.Backend {
	v := ctx.Value(backendQuery)

	if v == nil {
		return windows.Backend{}
	}

	//nolint:forcetypeassert // The panic is expected and welcome
	return v.(backend.Backend)
}

// ErrNotExist is the error returned when a distro does not exist.
var ErrNotExist = mock.ErrNotExist
