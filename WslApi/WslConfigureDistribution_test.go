package WslApi_test

import (
	"WslApi"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigure(tst *testing.T) {
	tst.Skip("This test is not yet ready")

	tests := map[string]WslApi.Configuration{
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
		t := NewTester(tst)
		time.Sleep(15 * time.Second)
		t.Logf("Running test case %s\n", name)

		distro := setUpDistro(&t, name)
		conf, err := distro.GetConfiguration()
		require.NoError(t, err)

		conf.DefaultUID = wants.DefaultUID
		conf.InteropEnabled = wants.InteropEnabled
		conf.PathAppended = wants.PathAppended
		conf.DriveMountingEnabled = wants.DriveMountingEnabled

		err = distro.Configure(conf)
		require.NoError(t, err)

		got, err := distro.GetConfiguration()
		require.NoError(t, err)

		// Config test
		require.Equal(t, wants.DefaultUID, got.DefaultUID)
		require.Equal(t, wants.InteropEnabled, got.InteropEnabled)
		require.Equal(t, wants.PathAppended, got.PathAppended)
		require.Equal(t, wants.DriveMountingEnabled, got.DriveMountingEnabled)

		// TODO: behviour tests
	}

}

// setUpDistros registers and creates a user
func setUpDistro(t *Tester, name string) WslApi.Distro {
	distro := t.NewDistro(name)
	err := distro.Register(jammyRootFs)
	require.NoError(t, err)

	reg, err := distro.IsRegistered()
	require.NoError(t, err)
	require.True(t, reg)

	exitCode, err := distro.LaunchInteractive("useradd testuser", false)
	require.NoError(t, err)
	require.Equal(t, exitCode, WslApi.ExitCode(0))

	err = distro.Terminate()
	require.NoError(t, err)

	return distro
}
