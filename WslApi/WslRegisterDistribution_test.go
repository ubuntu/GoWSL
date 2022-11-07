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
	defer distro1.Unregister()

	err = distro2.Register(jammyRootFs)
	require.Error(t, err) // Space not allowed in name

	err = distro1.Register(jammyRootFs)
	require.Error(t, err) // Double registration disallowed

	testDistros, err := findTestDistros()
	require.NoError(t, err)
	require.Contains(t, testDistros, distro1)
	require.NotContains(t, testDistros, distro2)

}
