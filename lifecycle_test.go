package wsl_test

import (
	"fmt"
	"testing"
	"wsl"

	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	t.Skip("The WslRegisterDistribution API call is very flaky for some reason")

	d1 := wsl.Distro{Name: fmt.Sprintf("%s_%s_%s", namePrefix, "_nameIsValid", uniqueId())}
	t.Cleanup(func() { cleanUpWslInstance(d1) })

	d2 := wsl.Distro{Name: fmt.Sprintf("%s_%s_%s", namePrefix, "_name Not Valid", uniqueId())}
	t.Cleanup(func() { cleanUpWslInstance(d2) })

	err := d1.Register(emptyRootFs)
	require.NoError(t, err)

	err = d2.Register(emptyRootFs)
	require.Error(t, err) // Space not allowed in name

	err = d1.Register(emptyRootFs)
	require.Error(t, err) // Double registration disallowed

	testInstances, err := registeredTestWslInstances()
	require.NoError(t, err)
	require.Contains(t, testInstances, d1)
	require.NotContains(t, testInstances, d2)
}

func TestRegisteredDistros(t *testing.T) {
	d1 := newDistro(t, emptyRootFs)
	d2 := newDistro(t, emptyRootFs)
	d3 := wsl.Distro{Name: "NotRegistered"}

	list, err := wsl.RegisteredDistros()
	require.NoError(t, err)

	require.Contains(t, list, d1)
	require.Contains(t, list, d2)
	require.NotContains(t, list, d3)
}

func TestIsRegistered(t *testing.T) {
	tests := map[string]struct {
		distroName     string
		register       bool
		wantError      bool
		wantRegistered bool
	}{
		"nominal":    {register: true, wantError: false, wantRegistered: true},
		"inexistent": {register: false, wantError: false, wantRegistered: false},
	}

	for name, config := range tests {
		name := name
		config := config

		t.Run(name, func(t *testing.T) {

			var distro wsl.Distro
			if config.register {
				distro = newDistro(t, emptyRootFs)
			} else {
				distro = wsl.Distro{Name: "IAmNotRegistered"}
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

func TestUnRegister(t *testing.T) {
	distro1 := newDistro(t, emptyRootFs)
	distro2 := wsl.Distro{Name: "ThisDistroDoesNotExist"}
	distro3 := wsl.Distro{Name: "This Distro Is Not Valid"}

	err := distro1.Unregister()
	require.NoError(t, err)

	err = distro2.Unregister()
	require.Error(t, err)

	err = distro3.Unregister()
	require.Error(t, err)

	testInstances, err := registeredTestWslInstances()
	require.NoError(t, err)
	require.NotContains(t, testInstances, distro1)
	require.NotContains(t, testInstances, distro2)
	require.NotContains(t, testInstances, distro3)
}
