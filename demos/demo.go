package main

import (
	"WslApi"
	"errors"
	"fmt"
	"syscall"
)

func main() {
	instance := WslApi.Instance{Name: "Ubuntu-24.04"}

	// Registering a new instance
	fmt.Println("Registering a new WSL instance...")
	if err := instance.Register(`C:\Users\edu19\Work\images\jammy.tar.gz`); err != nil {
		panic(err)
	}

	// Ensuring the instance is unregistered at the end
	defer instance.Unregister()

	// Getting config and printing it
	config, err := instance.GetConfiguration()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v", config)

	// Setting config
	config.PathAppended = false
	instance.Configure(config)

	// Launching async command (Win32 style)
	process1, err := instance.Launch(`sleep 3 && cat goodmorning.txt`, false, 0, syscall.Stdout, syscall.Stdout)
	if err != nil {
		panic(err)
	}
	defer process1.Close()

	// Launching async command (Go style)
	process2 := instance.NewWslProcess(`echo "Hello, world from WSL!" > "goodmorning.txt"`)
	process2.Stdin = 0 // (nullptr) TODO: Make this more Go-like with readers and writers
	process2.Start()

	// Launching a regular command (should fail as per config change)
	err = instance.LaunchInteractive("notepad.exe", false)
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
	if err != nil && !errors.Is(err, &WslApi.ExitError{}) {
		panic(err)
	}

	if err != nil {
		fmt.Printf("Unsuccesful async command: %v\n", err)
		return
	}
	fmt.Printf("Succesful async command!\n")
}
