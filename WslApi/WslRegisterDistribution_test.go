package WslApi_test

import (
	"WslApi"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegister(tst *testing.T) {
	t := NewTester(tst)

	distro1 := WslApi.Distro{Name: mangleName("Ubuntu")}
	distro2 := WslApi.Distro{Name: mangleName("Ubuntu but Better")}

	err := distro1.Register(`C:\Users\edu19\Work\images\jammy.tar.gz`)
	require.NoError(t, err)

	err = distro2.Register(`C:\Users\edu19\Work\images\jammy.tar.gz`)
	require.Error(t, err) // Space not allowed in name

	err = distro1.Register(`C:\Users\edu19\Work\images\jammy.tar.gz`)
	require.Error(t, err) // Double registration disallowed

	testDistros, err := findTestDistros()
	require.NoError(t, err)
	require.Contains(t, testDistros, distro1.Name)
	require.NotContains(t, testDistros, distro2.Name)
}
