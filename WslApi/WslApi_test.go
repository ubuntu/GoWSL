package WslApi_test

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

const distroSuffix = "wsltesting"

type Tester struct {
	*testing.T
}

func TestMain(m *testing.M) {
	cleanUpDistros()
	exitVal := m.Run()
	os.Exit(exitVal)
}

func NewTester(tst *testing.T) (t Tester) {
	t = Tester{T: tst}
	t.Cleanup(cleanUpDistros)
	return t
}

// mangleName avoids name collisions with existing distros by adding a suffix
func mangleName(name string) string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		panic(fmt.Errorf("Failed to get caller name"))
	}
	funcname := runtime.FuncForPC(pc).Name()
	return fmt.Sprintf("%s_%s_%s", name, funcname, distroSuffix)
}

// cleanUpDistros finds all distros with mangled name and unregisters them
func cleanUpDistros() {
	testDistros, err := findTestDistros()
	if err != nil {
		panic(fmt.Errorf("failed to get test distros"))
	}

	for _, distroName := range testDistros {
		name, test := unmangleName(distroName)
		err := exec.Command("wsl.exe", "--unregister", distroName).Run()
		if err != nil {
			fmt.Printf("failed to clean up test distro (name=%s, test=%s)\n", name, test)
		}
	}
}

// unmangleName retrieves encoded info from distro name
func unmangleName(mangledName string) (name string, test string) {
	words := strings.Split(mangledName, "_")
	l := len(words)
	name = strings.Join(words[:l-2], "_")
	test = words[l-2]
	return name, test
}

// findTestDistros finds all distros with a mangled name
func findTestDistros() ([]string, error) {
	distros := []string{}

	outp, err := exec.Command("powershell.exe", "-command", "$env:WSL_UTF8=1 ; wsl.exe --list --quiet").CombinedOutput()
	if err != nil {
		return distros, err
	}

	for _, line := range strings.Fields(string(outp)) {
		if !strings.HasSuffix(line, distroSuffix) {
			continue
		}
		distros = append(distros, line)
	}

	return distros, nil
}
