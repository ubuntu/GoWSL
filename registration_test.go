package wsl_test

import (
	"testing"
	"wsl"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	d1 := wsl.Distro{Name: UniqueDistroName(t)}
	t.Cleanup(func() { cleanUpWslInstance(d1) })

	d2 := wsl.Distro{Name: UniqueDistroName(t) + "_name not valid"}
	t.Cleanup(func() { cleanUpWslInstance(d2) })

	err := d1.Register(emptyRootFs)
	require.NoError(t, err)

	err = d2.Register(emptyRootFs)
	require.Error(t, err) // Space not allowed in name

	err = d1.Register(emptyRootFs)
	require.Error(t, err) // Double registration disallowed

	testInstances, err := registeredTestWslInstances()
	require.NoError(t, err)
	assert.Contains(t, testInstances, d1)
	assert.NotContains(t, testInstances, d2)
}

func TestRegisteredDistros(t *testing.T) {
	d1 := newTestDistro(t, emptyRootFs)
	d2 := newTestDistro(t, emptyRootFs)
	d3 := wsl.Distro{Name: UniqueDistroName(t)}

	list, err := wsl.RegisteredDistros()
	require.NoError(t, err)

	assert.Contains(t, list, d1)
	assert.Contains(t, list, d2)
	assert.NotContains(t, list, d3)
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
				distro = newTestDistro(t, emptyRootFs)
			} else {
				distro = wsl.Distro{Name: UniqueDistroName(t)}
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
	distro1 := newTestDistro(t, emptyRootFs)
	distro2 := wsl.Distro{Name: UniqueDistroName(t)}
	distro3 := wsl.Distro{Name: "This Distro Is Not Valid"}

	err := distro1.Unregister()
	require.NoError(t, err)

	err = distro2.Unregister()
	require.Error(t, err)

	err = distro3.Unregister()
	require.Error(t, err)

	testInstances, err := registeredTestWslInstances()
	require.NoError(t, err)
	assert.NotContains(t, testInstances, distro1)
	assert.NotContains(t, testInstances, distro2)
	assert.NotContains(t, testInstances, distro3)
}
