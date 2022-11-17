package wsl_test

import (
	"context"
	"testing"
	"wsl"

	"github.com/stretchr/testify/require"
)

func TestConfiguration(tst *testing.T) {
	t := NewTester(tst)

	distro := t.NewWslDistro("jammy")
	t.RegisterFromPowershell(distro, jammyRootFs)

	cmd := distro.Command(context.Background(), "useradd testuser")
	err := cmd.Run()
	require.NoError(t, err)

	default_config, err := distro.GetConfiguration()
	require.NoError(t, err)

	tests := map[string]wsl.Configuration{
		"root000": {DefaultUID: 0, InteropEnabled: false, PathAppended: false, DriveMountingEnabled: false},
		"root001": {DefaultUID: 0, InteropEnabled: false, PathAppended: false, DriveMountingEnabled: true},
		"root010": {DefaultUID: 0, InteropEnabled: false, PathAppended: true, DriveMountingEnabled: false},
		"root011": {DefaultUID: 0, InteropEnabled: false, PathAppended: true, DriveMountingEnabled: true},
		"root100": {DefaultUID: 0, InteropEnabled: true, PathAppended: false, DriveMountingEnabled: false},
		"root101": {DefaultUID: 0, InteropEnabled: true, PathAppended: false, DriveMountingEnabled: true},
		"root110": {DefaultUID: 0, InteropEnabled: true, PathAppended: true, DriveMountingEnabled: false},
		"root111": {DefaultUID: 0, InteropEnabled: true, PathAppended: true, DriveMountingEnabled: true},
		"user000": {DefaultUID: 1000, InteropEnabled: false, PathAppended: false, DriveMountingEnabled: false},
		"user001": {DefaultUID: 1000, InteropEnabled: false, PathAppended: false, DriveMountingEnabled: true},
		"user010": {DefaultUID: 1000, InteropEnabled: false, PathAppended: true, DriveMountingEnabled: false},
		"user011": {DefaultUID: 1000, InteropEnabled: false, PathAppended: true, DriveMountingEnabled: true},
		"user100": {DefaultUID: 1000, InteropEnabled: true, PathAppended: false, DriveMountingEnabled: false},
		"user101": {DefaultUID: 1000, InteropEnabled: true, PathAppended: false, DriveMountingEnabled: true},
		"user110": {DefaultUID: 1000, InteropEnabled: true, PathAppended: true, DriveMountingEnabled: false},
		"user111": {DefaultUID: 1000, InteropEnabled: true, PathAppended: true, DriveMountingEnabled: true},
	}

	for name, wants := range tests {
		tst.Run(name, func(tst *testing.T) {
			defer func() { // Reseting to default state
				distro.DefaultUID(default_config.DefaultUID)
				distro.InteropEnabled(default_config.InteropEnabled)
				distro.PathAppended(default_config.PathAppended)
				distro.DriveMountingEnabled(default_config.DriveMountingEnabled)
			}()

			t := NewTester(tst)

			distro.DefaultUID(wants.DefaultUID)
			require.NoError(t, err)

			distro.InteropEnabled(wants.InteropEnabled)
			require.NoError(t, err)

			distro.PathAppended(wants.PathAppended)
			require.NoError(t, err)

			distro.DriveMountingEnabled(wants.DriveMountingEnabled)
			require.NoError(t, err)

			got, err := distro.GetConfiguration()
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
