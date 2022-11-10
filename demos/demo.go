package main

import (
	"WslApi"
	"errors"
	"fmt"
	"syscall"
)

func main() {
	distro := WslApi.Distro{Name: "Ubuntu-24.04"}

	// Registering a new distro
	fmt.Println("Registering a new distro...")
	if err := distro.Register(`C:\Users\edu19\Work\images\jammy.tar.gz`); err != nil {
		panic(err)
	}

	// Ensuring the distro is unregistered at the end
	defer func() {
		if err := distro.Unregister(); err != nil {
			panic(err)
		}
	}()

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
	process, err := distro.Launch(`sleep 3 && echo "Good morning!"`, false, 0, syscall.Stdout, syscall.Stdout)
	if err != nil {
		panic(err)
	}
	defer process.Close()

	// Launching a regular command (should fail as per config change)
	err = distro.LaunchInteractive("notepad.exe", false)
	if err != nil {
		fmt.Printf("Sync command unsuccesful: %v\n", err)
	}
	fmt.Printf("Sync command succesful\n")

	// Managing the result of the async command
	err = process.Wait()
	if err != nil && !errors.Is(err, &WslApi.ExitError{}) {
		panic(err)
	}

	if err != nil {
		fmt.Printf("Unsuccesful async command: %v\n", err)
		return
	}
	fmt.Printf("Succesful async command!\n")
}
