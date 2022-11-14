package wsl_test

import (
	"testing"
	"wsl"

	"github.com/stretchr/testify/require"
)

func TestRegister(tst *testing.T) {
	t := NewTester(tst)

	instance1 := t.NewWslInstance("Ubuntu")
	instance2 := t.NewWslInstance("Señor Ubuntu")

	err := instance1.Register(jammyRootFs)
	require.NoError(t, err)

	err = instance2.Register(jammyRootFs)
	require.Error(t, err) // Space not allowed in name

	err = instance1.Register(jammyRootFs)
	require.Error(t, err) // Double registration disallowed

	testInstances, err := RegisteredTestWslInstances()
	require.NoError(t, err)
	require.Contains(t, testInstances, instance1)
	require.NotContains(t, testInstances, instance2)
}

func TestRegisteredIntances(tst *testing.T) {
	t := NewTester(tst)
	d1 := t.NewWslInstance("Ubuntu")
	d2 := t.NewWslInstance("Ubuntu.Again")
	d3 := t.NewWslInstance("NotRegistered")

	t.RegisterFromPowershell(d1, emptyRootFs)
	t.RegisterFromPowershell(d2, emptyRootFs)

	list, err := wsl.RegisteredIntances()
	require.NoError(t, err)

	require.Contains(t, list, d1)
	require.Contains(t, list, d2)
	require.NotContains(t, list, d3)
}

func TestIsRegistered(tst *testing.T) {
	tests := map[string]struct {
		instanceName   string
		register       bool
		wantError      bool
		wantRegistered bool
	}{
		"nominal":    {instanceName: "UbuntuNominal", register: true, wantError: false, wantRegistered: true},
		"inexistent": {instanceName: "UbuntuInexistent", register: false, wantError: false, wantRegistered: false},
		"wrong_name": {instanceName: "Ubuntu Wrong Name", register: false, wantError: false, wantRegistered: false},
	}

	for name, config := range tests {
		name := name
		config := config

		tst.Run(name, func(tst *testing.T) {
			t := NewTester(tst)
			instance := t.NewWslInstance(config.instanceName)

			if config.register {
				t.RegisterFromPowershell(instance, emptyRootFs)
			}

			reg, err := instance.IsRegistered()
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

	inst1 := t.NewWslInstance("Ubuntu")
	inst2 := t.NewWslInstance("ThisDistroDoesNotExist")
	inst3 := t.NewWslInstance("This Distro Is Not Valid")

	t.RegisterFromPowershell(inst1, emptyRootFs)

	testInstances, err := RegisteredTestWslInstances()
	require.NoError(t, err)
	require.Contains(t, testInstances, inst1)
	require.NotContains(t, testInstances, inst2)
	require.NotContains(t, testInstances, inst3)

	err = inst1.Unregister()
	require.NoError(t, err)

	err = inst2.Unregister()
	require.Error(t, err)

	err = inst3.Unregister()
	require.Error(t, err)

	testInstances, err = RegisteredTestWslInstances()
	require.NoError(t, err)
	require.NotContains(t, testInstances, inst1)
	require.NotContains(t, testInstances, inst2)
	require.NotContains(t, testInstances, inst3)
}
