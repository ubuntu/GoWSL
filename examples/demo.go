package main

import (
	"context"
	"errors"
	"fmt"
	"time"
	"wsl"
)

func main() {
	distro := wsl.Distro{Name: "Ubuntu-22.04-test"}

	// Registering a new distro
	fmt.Println("Registering a new WSL distro...")
	if err := distro.Register(`.\images\jammy.tar.gz`); err != nil {
		panic(err)
	}

	// Ensuring the distro is unregistered at the end
	defer distro.Unregister()

	// Getting config and printing it
	fmt.Printf("%v", distro)

	// Setting config
	distro.PathAppended(false)

	// Launching async command
	process1 := distro.Command(context.Background(), `sleep 3 && cat goodmorning.txt`)
	process1.Stdin = 0 // (nullptr) TODO: Make this more Go-like with readers and writers
	process1.Start()

	// Launching async command
	process2 := distro.Command(context.Background(), `echo "Hello, world from WSL!" > "goodmorning.txt"`)
	process2.Stdin = 0 // (nullptr) TODO: Make this more Go-like with readers and writers
	process2.Run()

	// Launching an interactive command (should fail as per config change)
	err := distro.Shell(wsl.WithCommand("sh -c 'notepad.exe'"))
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

	// Showing CommandContext
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = distro.Command(ctx, "sleep 5 && echo 'Woke up!'").Run()
	if err != nil {
		fmt.Printf("Process with timeout failed: %v\n", err)
	} else {
		fmt.Println("Process with timeout succeeded!")
	}

}
