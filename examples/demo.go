package main

import (
	"os"

	wsl "github.com/EduardGomezEscandell/GoWSL"

	"context"
	"errors"
	"fmt"
	"time"
)

func main() {
	distro := wsl.Distro{Name: "Ubuntu-GoWSL-demo"}

	// Registering a new distro
	fmt.Printf("Registering a new distro %q\n", distro.Name)
	if err := distro.Register(`.\images\rootfs.tar.gz`); err != nil {
		fmt.Fprintf(os.Stderr, "Unexpected error: %v\n", err)
		return
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
	target := &wsl.ExitError{}
	switch {
	case err == nil:
		fmt.Printf("Succesful async command!\n")
	case errors.As(err, &target):
		fmt.Printf("Unsuccesful async command: %v\n", err)
	default:
		fmt.Fprintf(os.Stderr, "Unexpected error: %v\n", err)
		return
	}

	// Showing CommandContext
	fmt.Println("\nCancelling a command that takes too long")
	// We call 'sleep 5' but cancel after only one second.
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = distro.Command(ctx, "sleep 5 && echo 'Woke up!'").Run()
	if err != nil {
		fmt.Printf("Process with timeout failed: %v\n", err)
	} else {
		fmt.Println("Process with timeout succeeded!")
	}

	// Showing CombinedOutput
	fmt.Println("\rRunning a command with redirected output")
	fmt.Println()
	// Useful so the next shell command is less verbose
	out, err = distro.Command(context.Background(), "touch /root/.hushlogin").CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unexpected error: %v\nError message: %s", err, out)
	}

	// Starting Shell
	fmt.Println("\r\nStarting a shell in Ubuntu. Feel free to `exit <NUMBER>` to continue the demo")

	fmt.Println("\r")
	err = distro.Shell()
	switch {
	case err == nil:
		fmt.Printf("\rShell exited with exit code 0\n")
	case errors.As(err, &target):
		fmt.Printf("\rShell exited with exit code %d\n", target.Code)
	default:
		fmt.Fprintf(os.Stderr, "Unexpected error: %v\n", err)
		return
	}
	fmt.Println("\r")
}
