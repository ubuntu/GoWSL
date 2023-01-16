package GoWSL

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnpackFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wants Configuration
		input wslFlags
	}{
		{input: wslFlags(0x0), wants: Configuration{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: false}},
		{input: wslFlags(0x1), wants: Configuration{InteropEnabled: true, PathAppended: false, DriveMountingEnabled: false}},
		{input: wslFlags(0x2), wants: Configuration{InteropEnabled: false, PathAppended: true, DriveMountingEnabled: false}},
		{input: wslFlags(0x3), wants: Configuration{InteropEnabled: true, PathAppended: true, DriveMountingEnabled: false}},
		{input: wslFlags(0x4), wants: Configuration{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: true}},
		{input: wslFlags(0x5), wants: Configuration{InteropEnabled: true, PathAppended: false, DriveMountingEnabled: true}},
		{input: wslFlags(0x6), wants: Configuration{InteropEnabled: false, PathAppended: true, DriveMountingEnabled: true}},
		{input: wslFlags(0x7), wants: Configuration{InteropEnabled: true, PathAppended: true, DriveMountingEnabled: true}},
		// The following may be encountered due to an undocumented fourth flag
		{input: wslFlags(0x8), wants: Configuration{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: false}},
		{input: wslFlags(0x9), wants: Configuration{InteropEnabled: true, PathAppended: false, DriveMountingEnabled: false}},
		{input: wslFlags(0xa), wants: Configuration{InteropEnabled: false, PathAppended: true, DriveMountingEnabled: false}},
		{input: wslFlags(0xb), wants: Configuration{InteropEnabled: true, PathAppended: true, DriveMountingEnabled: false}},
		{input: wslFlags(0xc), wants: Configuration{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: true}},
		{input: wslFlags(0xd), wants: Configuration{InteropEnabled: true, PathAppended: false, DriveMountingEnabled: true}},
		{input: wslFlags(0xe), wants: Configuration{InteropEnabled: false, PathAppended: true, DriveMountingEnabled: true}},
		{input: wslFlags(0xf), wants: Configuration{InteropEnabled: true, PathAppended: true, DriveMountingEnabled: true}},
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
		wants wslFlags
	}{
		{wants: wslFlags(0x0), input: Configuration{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: false}},
		{wants: wslFlags(0x1), input: Configuration{InteropEnabled: true, PathAppended: false, DriveMountingEnabled: false}},
		{wants: wslFlags(0x2), input: Configuration{InteropEnabled: false, PathAppended: true, DriveMountingEnabled: false}},
		{wants: wslFlags(0x3), input: Configuration{InteropEnabled: true, PathAppended: true, DriveMountingEnabled: false}},
		{wants: wslFlags(0x4), input: Configuration{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: true}},
		{wants: wslFlags(0x5), input: Configuration{InteropEnabled: true, PathAppended: false, DriveMountingEnabled: true}},
		{wants: wslFlags(0x6), input: Configuration{InteropEnabled: false, PathAppended: true, DriveMountingEnabled: true}},
		{wants: wslFlags(0x7), input: Configuration{InteropEnabled: true, PathAppended: true, DriveMountingEnabled: true}},
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
