package WslApi_test

import (
	"WslApi"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnRegister(tst *testing.T) {
	t := NewTester(tst)

	distro1 := WslApi.Distro{Name: mangleName("Ubuntu")}
	distro2 := WslApi.Distro{Name: mangleName("ThisDistroDoesNotExist")}
	distro3 := WslApi.Distro{Name: mangleName("This Distro Is Not Valid")}

	err := distro1.Register(`C:\Users\edu19\Work\images\jammy.tar.gz`)
	require.NoError(t, err)

	testDistros, err := findTestDistros()
	fmt.Printf("%v\n", testDistros)
	require.NoError(t, err)
	require.Contains(t, testDistros, distro1.Name)
	require.NotContains(t, testDistros, distro2.Name)
	require.NotContains(t, testDistros, distro3.Name)

	err = distro1.Unregister()
	require.NoError(t, err)

	err = distro2.Unregister()
	require.Error(t, err)

	err = distro3.Unregister()
	require.Error(t, err)

	testDistros, err = findTestDistros()
	require.NoError(t, err)
	require.NotContains(t, testDistros, distro1.Name)
	require.NotContains(t, testDistros, distro2.Name)
	require.NotContains(t, testDistros, distro3.Name)

}
