package WslApi_test

import (
	"WslApi"
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

	testDistros, err := RegisteredTestDistros()
	require.NoError(t, err)
	require.Contains(t, testDistros, distro1)
	require.NotContains(t, testDistros, distro2)
}

func TestRegisteredDistros(tst *testing.T) {
	t := NewTester(tst)
	d1 := t.NewDistro("Ubuntu")
	d2 := t.NewDistro("Ubuntu.Again")
	d3 := t.NewDistro("NotRegistered")

	t.RegisterFromPowershell(d1, emptyRootFs)
	t.RegisterFromPowershell(d2, emptyRootFs)

	list, err := WslApi.RegisteredDistros()
	require.NoError(t, err)

	require.Contains(t, list, d1)
	require.Contains(t, list, d2)
	require.NotContains(t, list, d3)
}

func TestIsRegistered(tst *testing.T) {
	tests := map[string]struct {
		distroName     string
		register       bool
		wantError      bool
		wantRegistered bool
	}{
		"nominal":    {distroName: "UbuntuNominal", register: true, wantError: false, wantRegistered: true},
		"inexistent": {distroName: "UbuntuInexistent", register: false, wantError: false, wantRegistered: false},
		"wrong_name": {distroName: "Ubuntu Wrong Name", register: false, wantError: false, wantRegistered: false},
	}

	for name, config := range tests {
		name := name
		config := config

		tst.Run(name, func(tst *testing.T) {
			t := NewTester(tst)
			distro := t.NewDistro(config.distroName)

			if config.register {
				t.RegisterFromPowershell(distro, emptyRootFs)
			}

			reg, err := distro.IsRegistered()
			if config.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if config.wantRegistered {
				require.True(t, reg)
			} else {
				require.False(t, reg)
			}
		})
	}
}

func TestUnRegister(tst *testing.T) {
	t := NewTester(tst)

	distro1 := t.NewDistro("Ubuntu")
	distro2 := t.NewDistro("ThisDistroDoesNotExist")
	distro3 := t.NewDistro("This Distro Is Not Valid")

	t.RegisterFromPowershell(distro1, emptyRootFs)

	testDistros, err := RegisteredTestDistros()
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

	testDistros, err = RegisteredTestDistros()
	require.NoError(t, err)
	require.NotContains(t, testDistros, distro1)
	require.NotContains(t, testDistros, distro2)
	require.NotContains(t, testDistros, distro3)
}
