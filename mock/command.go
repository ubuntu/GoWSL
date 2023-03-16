package mock

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// mockedCommand is in charge of creating processes that behave the same way
// than the ones used in tests.
//
// A few notes about Windows:
//
//   - I use cmd.exe instead of powershell.exe because Powershell does not
//     allow writing freely to stderr: It becomes an exception so the text
//     does not match the one in tests.
//
//   - cmd.exe's version of "sleep 1" is "TIMEOUT /t 1 /nobreak >NUL", however
//     this command crashes if it cannot read from stdin. The standard alternative
//     is, shockingly, to ping some address and rely on the fact that it retries
//     once per second: "sleep 5" becomes "PING localhost -n 6 >NUL" The first attempt
//     is instantaneous so you perform `t+1` attempts, for a wait of `t` seconds
//
//   - In cmd.exe, "echo Hello >&2" will print "Hello " to stderr. Instead, you have
//     to do "(ECHO Hello) >&2" to avoid the trailing space.
type mockedCommand struct {
	linux, windows string
}

func newMockedCommand(cmd string) mockedCommand {
	m, ok := translateCommand[cmd]
	if !ok {
		panic(fmt.Sprintf("WslLaunch command not supported: %s", cmd))
	}

	if m.linux == "" {
		m.linux = cmd
	}
	if m.windows == "" {
		m.windows = cmd
	}

	return m
}

var translateCommand = map[string]mockedCommand{
	// Exit x
	"":        {linux: "exit 0", windows: "EXIT 0"},
	"exit 0":  {},
	"exit 42": {},

	// Sleep x
	"sleep 5":        {windows: "PING localhost -n 6 >NUL"},
	"sleep 10":       {windows: "PING localhost -n 11 >NUL"},
	"sleep infinity": {windows: "PING localhost -n -1 >NUL"},

	// Echo
	"echo 'Hello!'":     {windows: "(ECHO Hello!)"},
	"echo 'Error!' >&2": {windows: "(ECHO Error!) >&2"},

	// Combinations
	"echo 'Error!' >&2 && exit 42":                             {windows: "(ECHO Error!) >&2 && EXIT 42"},
	"echo 'Hello!' && sleep 1 && echo 'Error!' >&2":            {windows: "(ECHO Hello!) && (PING localhost -n 2) >NUL && (ECHO Error!) >&2"},
	"echo 'Hello!' && sleep 1 && echo 'Error!' >&2 && exit 42": {windows: "(ECHO Hello!) && (PING localhost -n 2) >NUL && (ECHO Error!) >&2 && EXIT 42"},

	// Other
	"useradd testuser": {linux: "exit 0", windows: "EXIT 0"},
}

// newCommandProcess starts a process of type:
//
//	windows: powershell.exe -Command <c.windows>
//	linux:   bash -c <c.linux>
//
// If cmd is empty, an interactive shell is started instead.
func (c mockedCommand) start(stdin, stdout, stderr *os.File) (*os.Process, error) {
	executable := "bash"
	argv := []string{executable, "-c", c.linux}
	if runtime.GOOS == "windows" {
		executable = "cmd.exe"
		argv = []string{executable, "/c", c.windows}
	}

	exec, err := exec.LookPath(executable)
	if err != nil {
		panic(fmt.Sprintf("could not find executable %q", executable))
	}

	p, err := os.StartProcess(exec, argv, &os.ProcAttr{
		Files: []*os.File{stdin, stdout, stderr},
	})

	if err != nil {
		return nil, fmt.Errorf("could not start mock process: %v", err)
	}

	return p, nil
}
