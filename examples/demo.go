package main

import (
	"errors"
	"fmt"
	"syscall"
	"wsl"
)

func main() {
	distro := wsl.Distro{Name: "Ubuntu-24.04"}

	// Registering a new instance
	fmt.Println("Registering a new WSL instance...")
	if err := distro.Register(`C:\Users\edu19\Work\images\jammy.tar.gz`); err != nil {
		panic(err)
	}

	// Ensuring the instance is unregistered at the end
	defer distro.Unregister()

	// Getting config and printing it
	config, err := distro.GetConfiguration()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v", config)

	// Setting config
	config.PathAppended = false
	distro.Configure(config)

	// Launching async command (Win32 style)
	process1, err := distro.Launch(`sleep 3 && cat goodmorning.txt`, false, 0, syscall.Stdout, syscall.Stdout)
	if err != nil {
		panic(err)
	}
	defer process1.Close()

	// Launching async command (Go style)
	process2 := distro.NewWslProcess(`echo "Hello, world from WSL!" > "goodmorning.txt"`)
	process2.Stdin = 0 // (nullptr) TODO: Make this more Go-like with readers and writers
	process2.Start()

	// Launching a regular command (should fail as per config change)
	err = distro.LaunchInteractive("notepad.exe", false)
	if err != nil {
		fmt.Printf("Sync command unsuccesful: %v\n", err)
	} else {
		fmt.Printf("Sync command succesful\n")
	}

	// Managing the result of the async commands
	err = process2.Wait()
	if err != nil {
		panic(err)
	}

	err = process1.Wait()
	if err != nil && !errors.Is(err, &wsl.ExitError{}) {
		panic(err)
	}

	if err != nil {
		fmt.Printf("Unsuccesful async command: %v\n", err)
		return
	}
	fmt.Printf("Succesful async command!\n")
}
