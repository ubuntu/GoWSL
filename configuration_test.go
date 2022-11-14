package wsl_test

import (
	"fmt"
	"testing"
	"wsl"

	"github.com/stretchr/testify/require"
)

func TestUnpackFlags(t *testing.T) {
	tests := map[wsl.Flags]wsl.Configuration{
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
			got := wsl.Configuration{InteropEnabled: false, PathAppended: false, DriveMountingEnabled: false}
			got.UnpackFlags(flags)
			require.Equal(t, wants.InteropEnabled, got.InteropEnabled)
			require.Equal(t, wants.PathAppended, got.PathAppended)
			require.Equal(t, wants.DriveMountingEnabled, got.DriveMountingEnabled)
		})
	}
}

func TestPackFlags(t *testing.T) {
	tests := map[wsl.Flags]wsl.Configuration{
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

// Overrides the values in baseline with the ones passed as arguments
// The arguments are the mutable values in wsl.Instance.Configure
func overrideMutableConfig(baseline wsl.Configuration, DefaultUID uint32, InteropEnabled bool, PathAppended bool, DriveMountingEnabled bool) wsl.Configuration {
	baseline.DefaultUID = DefaultUID
	baseline.InteropEnabled = InteropEnabled
	baseline.PathAppended = PathAppended
	baseline.DriveMountingEnabled = DriveMountingEnabled
	return baseline
}

func TestConfigure(tst *testing.T) {
	t := NewTester(tst)

	inst := t.NewWslInstance("jammy")
	t.RegisterFromPowershell(inst, jammyRootFs)

	err := inst.LaunchInteractive("useradd testuser", false)
	require.NoError(t, err)

	default_config, err := inst.GetConfiguration()
	require.NoError(t, err)

	tests := map[string]wsl.Configuration{
		"root000": overrideMutableConfig(default_config, 0, false, false, false),
		"root001": overrideMutableConfig(default_config, 0, false, false, true),
		"root010": overrideMutableConfig(default_config, 0, false, true, false),
		"root011": overrideMutableConfig(default_config, 0, false, true, true),
		"root100": overrideMutableConfig(default_config, 0, true, false, false),
		"root101": overrideMutableConfig(default_config, 0, true, false, true),
		"root110": overrideMutableConfig(default_config, 0, true, true, false),
		"root111": overrideMutableConfig(default_config, 0, true, true, true),
		"user000": overrideMutableConfig(default_config, 1000, false, false, false),
		"user001": overrideMutableConfig(default_config, 1000, false, false, true),
		"user010": overrideMutableConfig(default_config, 1000, false, true, false),
		"user011": overrideMutableConfig(default_config, 1000, false, true, true),
		"user100": overrideMutableConfig(default_config, 1000, true, false, false),
		"user101": overrideMutableConfig(default_config, 1000, true, false, true),
		"user110": overrideMutableConfig(default_config, 1000, true, true, false),
		"user111": overrideMutableConfig(default_config, 1000, true, true, true),
	}

	for name, wants := range tests {
		tst.Run(name, func(tst *testing.T) {
			defer inst.Configure(default_config) // Reseting to default state
			t := NewTester(tst)

			err = inst.Configure(wants)
			require.NoError(t, err)

			got, err := inst.GetConfiguration()
			require.NoError(t, err)

			// Config test
			require.Equal(t, wants.DefaultUID, got.DefaultUID)
			require.Equal(t, wants.InteropEnabled, got.InteropEnabled)
			require.Equal(t, wants.PathAppended, got.PathAppended)
			require.Equal(t, wants.DriveMountingEnabled, got.DriveMountingEnabled)

			// TODO: behaviour tests
		})
	}
}
