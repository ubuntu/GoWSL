package main

import (
	wsl "github.com/EduardGomezEscandell/GoWSL"

	"context"
	"errors"
	"fmt"
	"time"
)

func main() {
	distro := wsl.Distro{Name: "Ubuntu-22.04-test"}

	// Registering a new distro
	fmt.Printf("Registering a new distro %q\n", distro.Name)
	if err := distro.Register(`.\images\jammy.tar.gz`); err != nil {
		panic(err)
	}

	// Ensuring the distro is unregistered at the end
	defer distro.Unregister()

	// Getting config and printing it
	fmt.Println("\nPrinting distro information:")
	fmt.Printf("%v", distro)

	// Setting config
	fmt.Println("\nDisable windows paths and fail to run notepad:")
	distro.PathAppended(false)

	// Launching an interactive command (should fail as per config change)
	out, err := distro.Command(context.Background(), "sh -c 'notepad.exe'").CombinedOutput()
	if err != nil {
		fmt.Printf("%v\n%s", err, out)
	} else {
		fmt.Printf("Unexpected success:\n%s", out)
	}

	// Launching async command 1
	fmt.Println("\nRunning async commands")
	cmd1 := distro.Command(context.Background(), `sleep 3 && cat goodmorning.txt`)
	cmd1.Start()

	// Launching and waiting for command 2
	distro.Command(context.Background(), `echo "Hello, world from WSL!" > "goodmorning.txt"`).Run()

	// Waiting for command 1
	err = cmd1.Wait()
	if err != nil && !errors.Is(err, &wsl.ExitError{}) {
		panic(err)
	}
	if err != nil {
		fmt.Printf("Unsuccesful async command: %v\n", err)
		return
	}
	fmt.Printf("Succesful async command!\n")

	// Showing CommandContext
	fmt.Println("\nCancelling a command that takes too long")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = distro.Command(ctx, "sleep 5 && echo 'Woke up!'").Run()
	if err != nil {
		fmt.Printf("Process with timeout failed: %v\n", err)
	} else {
		fmt.Println("Process with timeout succeeded!")
	}

	// Starting Shell
	fmt.Println("\nStarting a shell in Ubuntu. Feel free to exit to continue the demo")
	fmt.Println()
	err = distro.Shell()
	if err != nil {
		fmt.Printf("Shell exited with error:\n%s", err)
	} else {
		fmt.Println("Shell exited with exit code 0")
	}
}
