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

// WithMock adds the mock back-end to the context.
func WithMock(ctx context.Context, backend mock.Backend) context.Context {
	return context.WithValue(ctx, backendQuery, backend)
}

func selectBackend(ctx context.Context) backend.Backend {
	v := ctx.Value(backendQuery)

	if v == nil {
		return windows.Backend{}
	}

	//nolint: forcetypeassert // The panic is expected and welcome
	return v.(backend.Backend)
}
