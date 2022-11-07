package WslApi_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
