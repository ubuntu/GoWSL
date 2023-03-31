// Package state defines the state enum so that both backends can use it
package state

import "fmt"

// State is the state of a particular distro as seen in `wsl.exe -l -v`.
type State int

// The states here reported are the ones obtained via `wsl.exe -l -v`,
// with the addition of NotRegistered.
const (
	Error State = iota
	Stopped
	Running
	Installing
	Uninstalling
	NotRegistered
)

// NewFromString parses the name of a state as printed in `wsl.exe -l -v`
// and returns its `State` enum value.
func NewFromString(s string) (State, error) {
	switch s {
	case "Stopped":
		return Stopped, nil
	case "Running":
		return Running, nil
	case "Installing":
		return Installing, nil
	case "Uninstalling":
		return Uninstalling, nil
	}

	return Error, fmt.Errorf("could not parse state %q", s)
}

func (s State) String() string {
	switch s {
	case Error:
		return "Error"
	case Stopped:
		return "Stopped"
	case Running:
		return "Running"
	case Installing:
		return "Installing"
	case NotRegistered:
		return "NotRegistered"
	case Uninstalling:
		return "Uninstalling"
	}

	return fmt.Sprintf("Unknown state %d", s)
}
