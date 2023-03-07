package gowsl

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ubuntu/gowsl/internal/flags"
)

func TestUnpackFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wants Configuration
		input flags.WslFlags
	}{
		{input: flags.WslFlags(0x0), wants: Configuration{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: false}},
		{input: flags.WslFlags(0x1), wants: Configuration{InteropEnabled: true, PathAppended: false, DriveMountingEnabled: false}},
		{input: flags.WslFlags(0x2), wants: Configuration{InteropEnabled: false, PathAppended: true, DriveMountingEnabled: false}},
		{input: flags.WslFlags(0x3), wants: Configuration{InteropEnabled: true, PathAppended: true, DriveMountingEnabled: false}},
		{input: flags.WslFlags(0x4), wants: Configuration{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: true}},
		{input: flags.WslFlags(0x5), wants: Configuration{InteropEnabled: true, PathAppended: false, DriveMountingEnabled: true}},
		{input: flags.WslFlags(0x6), wants: Configuration{InteropEnabled: false, PathAppended: true, DriveMountingEnabled: true}},
		{input: flags.WslFlags(0x7), wants: Configuration{InteropEnabled: true, PathAppended: true, DriveMountingEnabled: true}},
		// The following may be encountered due to an undocumented fourth flag
		{input: flags.WslFlags(0x8), wants: Configuration{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: false}},
		{input: flags.WslFlags(0x9), wants: Configuration{InteropEnabled: true, PathAppended: false, DriveMountingEnabled: false}},
		{input: flags.WslFlags(0xa), wants: Configuration{InteropEnabled: false, PathAppended: true, DriveMountingEnabled: false}},
		{input: flags.WslFlags(0xb), wants: Configuration{InteropEnabled: true, PathAppended: true, DriveMountingEnabled: false}},
		{input: flags.WslFlags(0xc), wants: Configuration{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: true}},
		{input: flags.WslFlags(0xd), wants: Configuration{InteropEnabled: true, PathAppended: false, DriveMountingEnabled: true}},
		{input: flags.WslFlags(0xe), wants: Configuration{InteropEnabled: false, PathAppended: true, DriveMountingEnabled: true}},
		{input: flags.WslFlags(0xf), wants: Configuration{InteropEnabled: true, PathAppended: true, DriveMountingEnabled: true}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(fmt.Sprintf("input_0x%x", int(tc.input)), func(t *testing.T) {
			t.Parallel()
			got := Configuration{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: false}
			got.unpackFlags(tc.input)
			assert.Equal(t, tc.wants.InteropEnabled, got.InteropEnabled)
			assert.Equal(t, tc.wants.PathAppended, got.PathAppended)
			assert.Equal(t, tc.wants.DriveMountingEnabled, got.DriveMountingEnabled)
		})
	}
}

func TestPackFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input Configuration
		wants flags.WslFlags
	}{
		{wants: flags.WslFlags(0x0), input: Configuration{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: false}},
		{wants: flags.WslFlags(0x1), input: Configuration{InteropEnabled: true, PathAppended: false, DriveMountingEnabled: false}},
		{wants: flags.WslFlags(0x2), input: Configuration{InteropEnabled: false, PathAppended: true, DriveMountingEnabled: false}},
		{wants: flags.WslFlags(0x3), input: Configuration{InteropEnabled: true, PathAppended: true, DriveMountingEnabled: false}},
		{wants: flags.WslFlags(0x4), input: Configuration{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: true}},
		{wants: flags.WslFlags(0x5), input: Configuration{InteropEnabled: true, PathAppended: false, DriveMountingEnabled: true}},
		{wants: flags.WslFlags(0x6), input: Configuration{InteropEnabled: false, PathAppended: true, DriveMountingEnabled: true}},
		{wants: flags.WslFlags(0x7), input: Configuration{InteropEnabled: true, PathAppended: true, DriveMountingEnabled: true}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(fmt.Sprintf("expects_0x%x", int(tc.wants)), func(t *testing.T) {
			t.Parallel()
			got, _ := tc.input.packFlags()
			require.Equal(t, tc.wants, got)
		})
	}
}
