package flags_test

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
		wants flags.Unpacked
		input flags.WslFlags
	}{
		{input: flags.WslFlags(0x0), wants: flags.Unpacked{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: false}},
		{input: flags.WslFlags(0x1), wants: flags.Unpacked{InteropEnabled: true, PathAppended: false, DriveMountingEnabled: false}},
		{input: flags.WslFlags(0x2), wants: flags.Unpacked{InteropEnabled: false, PathAppended: true, DriveMountingEnabled: false}},
		{input: flags.WslFlags(0x3), wants: flags.Unpacked{InteropEnabled: true, PathAppended: true, DriveMountingEnabled: false}},
		{input: flags.WslFlags(0x4), wants: flags.Unpacked{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: true}},
		{input: flags.WslFlags(0x5), wants: flags.Unpacked{InteropEnabled: true, PathAppended: false, DriveMountingEnabled: true}},
		{input: flags.WslFlags(0x6), wants: flags.Unpacked{InteropEnabled: false, PathAppended: true, DriveMountingEnabled: true}},
		{input: flags.WslFlags(0x7), wants: flags.Unpacked{InteropEnabled: true, PathAppended: true, DriveMountingEnabled: true}},
		// The following may be encountered due to an undocumented fourth flagcd
		{input: flags.WslFlags(0x8), wants: flags.Unpacked{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: false}},
		{input: flags.WslFlags(0x9), wants: flags.Unpacked{InteropEnabled: true, PathAppended: false, DriveMountingEnabled: false}},
		{input: flags.WslFlags(0xa), wants: flags.Unpacked{InteropEnabled: false, PathAppended: true, DriveMountingEnabled: false}},
		{input: flags.WslFlags(0xb), wants: flags.Unpacked{InteropEnabled: true, PathAppended: true, DriveMountingEnabled: false}},
		{input: flags.WslFlags(0xc), wants: flags.Unpacked{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: true}},
		{input: flags.WslFlags(0xd), wants: flags.Unpacked{InteropEnabled: true, PathAppended: false, DriveMountingEnabled: true}},
		{input: flags.WslFlags(0xe), wants: flags.Unpacked{InteropEnabled: false, PathAppended: true, DriveMountingEnabled: true}},
		{input: flags.WslFlags(0xf), wants: flags.Unpacked{InteropEnabled: true, PathAppended: true, DriveMountingEnabled: true}},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("input_0x%x", int(tc.input)), func(t *testing.T) {
			t.Parallel()
			got := flags.Unpack(tc.input)
			assert.Equal(t, tc.wants.InteropEnabled, got.InteropEnabled, "InteropEnabled does not match the expected value")
			assert.Equal(t, tc.wants.PathAppended, got.PathAppended, "PathAppended does not match the expected value")
			assert.Equal(t, tc.wants.DriveMountingEnabled, got.DriveMountingEnabled, "DriveMountingEnabled does not match the expected value")
		})
	}
}

func TestPackFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input flags.Unpacked
		wants flags.WslFlags
	}{
		{wants: flags.WslFlags(0x0), input: flags.Unpacked{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: false}},
		{wants: flags.WslFlags(0x1), input: flags.Unpacked{InteropEnabled: true, PathAppended: false, DriveMountingEnabled: false}},
		{wants: flags.WslFlags(0x2), input: flags.Unpacked{InteropEnabled: false, PathAppended: true, DriveMountingEnabled: false}},
		{wants: flags.WslFlags(0x3), input: flags.Unpacked{InteropEnabled: true, PathAppended: true, DriveMountingEnabled: false}},
		{wants: flags.WslFlags(0x4), input: flags.Unpacked{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: true}},
		{wants: flags.WslFlags(0x5), input: flags.Unpacked{InteropEnabled: true, PathAppended: false, DriveMountingEnabled: true}},
		{wants: flags.WslFlags(0x6), input: flags.Unpacked{InteropEnabled: false, PathAppended: true, DriveMountingEnabled: true}},
		{wants: flags.WslFlags(0x7), input: flags.Unpacked{InteropEnabled: true, PathAppended: true, DriveMountingEnabled: true}},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("expects_0x%x", int(tc.wants)), func(t *testing.T) {
			t.Parallel()
			got, _ := tc.input.Pack()
			require.Equal(t, tc.wants, got)
		})
	}
}
