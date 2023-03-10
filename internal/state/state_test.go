package state_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ubuntu/gowsl/internal/state"
)

func TestStateFromString(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input string

		want    state.State
		wantErr bool
	}{
		"Stopped":    {input: "Stopped", want: state.Stopped},
		"Running":    {input: "Running", want: state.Running},
		"Installing": {input: "Installing", want: state.Installing},

		// Error cases
		"Error with made-up state": {input: "Discombobulating", wantErr: true},
		"Error with empty string":  {input: "", wantErr: true},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := state.NewFromString(tc.input)
			if tc.wantErr {
				require.Error(t, err, "Unexpected success parsing wrong input")
				return
			}
			require.NoError(t, err, "NewFromString should not fail with valid inputs")

			require.Equal(t, tc.want, got, "Unexpected state returned by NewFromString")
		})
	}
}

func TestString(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input state.State
		want  string
	}{
		"Stopped":       {input: state.Stopped, want: "Stopped"},
		"Running":       {input: state.Running, want: "Running"},
		"Installing":    {input: state.Installing, want: "Installing"},
		"NotRegistered": {input: state.NotRegistered, want: "NotRegistered"},

		// Error case
		"Error with made-up state": {input: 35, want: "Unknown state 35"},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := tc.input.String()
			require.Equal(t, tc.want, got, "Unexpected text returned by state.String()")
		})
	}
}
