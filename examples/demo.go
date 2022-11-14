package main

import (
	"errors"
	"fmt"
	"wsl"
)

func main() {
	distro := wsl.Distro{Name: "Ubuntu-22.04-test"}

	// Registering a new distro
	fmt.Println("Registering a new WSL distro...")
	if err := distro.Register(`C:\Users\edu19\Work\images\jammy.tar.gz`); err != nil {
		panic(err)
	}

	// Ensuring the distro is unregistered at the end
	defer distro.Unregister()

	// Getting config and printing it
	config, err := distro.GetConfiguration()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v", config)

	// Setting config
	distro.PathAppended(false)

	// Launching async command
	process1 := distro.Command(`sleep 3 && cat goodmorning.txt`)
	process1.Stdin = 0 // (nullptr) TODO: Make this more Go-like with readers and writers
	process1.Start()

	// Launching async command
	process2 := distro.Command(`echo "Hello, world from WSL!" > "goodmorning.txt"`)
	process2.Stdin = 0 // (nullptr) TODO: Make this more Go-like with readers and writers
	process2.Run()

	// Launching an interactive command (should fail as per config change)
	err = distro.Shell(wsl.WithCommand("sh -c 'notepad.exe'"))
	if err != nil {
		fmt.Printf("Sync command unsuccesful: %v\n", err)
	} else {
		fmt.Printf("Sync command succesful\n")
	}

	// Managing the result of the async commands
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
