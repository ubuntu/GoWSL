package WslApi_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsRegistered(tst *testing.T) {
	NewTester(tst) // defering cleanup

	tests := map[string]struct {
		distroName     string
		register       bool
		wantError      bool
		wantRegistered bool
	}{
		"nominal":    {distroName: "UbuntuNominal", register: true, wantError: false, wantRegistered: true},
		"inexistant": {distroName: "Ubuntu.inexistant", register: false, wantError: false, wantRegistered: false},
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
