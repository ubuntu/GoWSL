package WslApi_test

import (
	"WslApi"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnpackFlags(t *testing.T) {
	tests := map[WslApi.Flags]WslApi.Configuration{
		0x0: {InteropEnabled: false, PathAppended: false, DriveMountingEnabled: false},
		0x1: {InteropEnabled: true, PathAppended: false, DriveMountingEnabled: false},
		0x2: {InteropEnabled: false, PathAppended: true, DriveMountingEnabled: false},
		0x3: {InteropEnabled: true, PathAppended: true, DriveMountingEnabled: false},
		0x4: {InteropEnabled: false, PathAppended: false, DriveMountingEnabled: true},
		0x5: {InteropEnabled: true, PathAppended: false, DriveMountingEnabled: true},
		0x6: {InteropEnabled: false, PathAppended: true, DriveMountingEnabled: true},
		0x7: {InteropEnabled: true, PathAppended: true, DriveMountingEnabled: true},
		// The following may be encountered due to an undocumented fourth flag
		0x8: {InteropEnabled: false, PathAppended: false, DriveMountingEnabled: false},
		0x9: {InteropEnabled: true, PathAppended: false, DriveMountingEnabled: false},
		0xa: {InteropEnabled: false, PathAppended: true, DriveMountingEnabled: false},
		0xb: {InteropEnabled: true, PathAppended: true, DriveMountingEnabled: false},
		0xc: {InteropEnabled: false, PathAppended: false, DriveMountingEnabled: true},
		0xd: {InteropEnabled: true, PathAppended: false, DriveMountingEnabled: true},
		0xe: {InteropEnabled: false, PathAppended: true, DriveMountingEnabled: true},
		0xf: {InteropEnabled: true, PathAppended: true, DriveMountingEnabled: true},
	}

	for flags, wants := range tests {
		flags := flags
		wants := wants
		t.Run(fmt.Sprintf("input_0x%x", int(flags)), func(t *testing.T) {
			got := WslApi.Configuration{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: false}
			got.UnpackFlags(flags)
			require.Equal(t, wants.InteropEnabled, got.InteropEnabled)
			require.Equal(t, wants.PathAppended, got.PathAppended)
			require.Equal(t, wants.DriveMountingEnabled, got.DriveMountingEnabled)
		})
	}
}

func TestPackFlags(t *testing.T) {
	tests := map[WslApi.Flags]WslApi.Configuration{
		0x0: {InteropEnabled: false, PathAppended: false, DriveMountingEnabled: false},
		0x1: {InteropEnabled: true, PathAppended: false, DriveMountingEnabled: false},
		0x2: {InteropEnabled: false, PathAppended: true, DriveMountingEnabled: false},
		0x3: {InteropEnabled: true, PathAppended: true, DriveMountingEnabled: false},
		0x4: {InteropEnabled: false, PathAppended: false, DriveMountingEnabled: true},
		0x5: {InteropEnabled: true, PathAppended: false, DriveMountingEnabled: true},
		0x6: {InteropEnabled: false, PathAppended: true, DriveMountingEnabled: true},
		0x7: {InteropEnabled: true, PathAppended: true, DriveMountingEnabled: true},
	}

	for wants, config := range tests {
		wants := wants
		config := config
		t.Run(fmt.Sprintf("expects_0x%x", int(wants)), func(t *testing.T) {
			got, _ := config.PackFlags()
			require.Equal(t, wants, got)
			require.Equal(t, wants, got)
			require.Equal(t, wants, got)
		})
	}
}
