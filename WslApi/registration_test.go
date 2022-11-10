package WslApi_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegister(tst *testing.T) {
	t := NewTester(tst)

	distro1 := t.NewDistro("Ubuntu")
	distro2 := t.NewDistro("Señor Ubuntu")

	err := distro1.Register(jammyRootFs)
	require.NoError(t, err)

	err = distro2.Register(jammyRootFs)
	require.Error(t, err) // Space not allowed in name

	err = distro1.Register(jammyRootFs)
	require.Error(t, err) // Double registration disallowed

	testDistros, err := findTestDistros()
	require.NoError(t, err)
	require.Contains(t, testDistros, distro1)
	require.NotContains(t, testDistros, distro2)
}

func TestIsRegistered(tst *testing.T) {
	tests := map[string]struct {
		distroName     string
		register       bool
		wantError      bool
		wantRegistered bool
	}{
		"nominal":    {distroName: "UbuntuNominal", register: true, wantError: false, wantRegistered: true},
		"inexistent": {distroName: "Ubuntu.inexistent", register: false, wantError: false, wantRegistered: false},
		"invalid":    {distroName: "Ubuntu . invalid", register: false, wantError: false, wantRegistered: false},
	}

	for name, config := range tests {
		name := name
		config := config

		func(tst *testing.T) {
			t := NewTester(tst)
			distro := t.NewDistro(config.distroName)

			if config.register {
				err := distro.Register(jammyRootFs)
				require.NoError(t, err, "Failure for subtest %s", name)
				defer distro.Unregister()
			}

			reg, err := distro.IsRegistered()
			if config.wantError {
				require.Error(t, err, "Failure for subtest %s", name)
			} else {
				require.NoError(t, err, "Failure for subtest %s", name)
			}

			if config.wantRegistered {
				require.True(t, reg, "Failure for subtest %s", name)
			} else {
				require.False(t, reg, "Failure for subtest %s", name)
			}
		}(tst)
	}
}

func TestUnRegister(tst *testing.T) {
	t := NewTester(tst)

	distro1 := t.NewDistro("Ubuntu")
	distro2 := t.NewDistro("ThisDistroDoesNotExist")
	distro3 := t.NewDistro("This Distro Is Not Valid")

	t.Logf(distro1.Name)
	err := distro1.Register(jammyRootFs)
	require.NoError(t, err)

	testDistros, err := findTestDistros()
	require.NoError(t, err)
	require.Contains(t, testDistros, distro1)
	require.NotContains(t, testDistros, distro2)
	require.NotContains(t, testDistros, distro3)

	err = distro1.Unregister()
	require.NoError(t, err)

	err = distro2.Unregister()
	require.Error(t, err)

	err = distro3.Unregister()
	require.Error(t, err)

	testDistros, err = findTestDistros()
	require.NoError(t, err)
	require.NotContains(t, testDistros, distro1)
	require.NotContains(t, testDistros, distro2)
	require.NotContains(t, testDistros, distro3)
}
