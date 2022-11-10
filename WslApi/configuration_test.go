package WslApi_test

import (
	"WslApi"
	"fmt"
	"testing"
	"time"

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
