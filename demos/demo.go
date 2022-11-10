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
	defer func() {
		if err := instance.Unregister(); err != nil {
			panic(err)
		}
	}()

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
	process, err := instance.Launch(`sleep 3 && echo "Good morning!"`, false, 0, syscall.Stdout, syscall.Stdout)
	if err != nil {
		panic(err)
	}
	defer process.Close()

	// Launching a regular command (should fail as per config change)
	err = instance.LaunchInteractive("notepad.exe", false)
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
