package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	wsl "github.com/ubuntu/gowsl"
)

func main() {
	distro := wsl.NewDistro("Ubuntu-GoWSL-demo")

	// Registering a new distro
	fmt.Printf("Registering a new distro %q\n", distro.Name())
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
	// Interactive commands are printed directly to console. If you call interactive
	// programs such as bash, python, etc. it'll will start an interactive session
	// that the user can interact with. This is presumably what wsl.exe uses.
	// It is a blocking call.
	if err := distro.Shell(wsl.WithCommand("sh -c 'powershell.exe'")); err != nil {
		fmt.Printf("Interactive session unsuccesful: %v\n", err)
	} else {
		fmt.Println("Interactive session succesful")
	}

	// Launching async command 1
	fmt.Println("\nRunning async commands")
	cmd1 := distro.Command(context.Background(), `sleep 3 && cat goodmorning.txt`)
	cmd1.Start()

	// Launching and waiting for command 2
	distro.Command(context.Background(), `echo "Hello, world from WSL!" > "goodmorning.txt"`).Run()

	// Waiting for command 1

	target := &exec.ExitError{}
	switch err := cmd1.Wait(); {
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

	if err := distro.Command(ctx, "sleep 5 && echo 'Woke up!'").Run(); err != nil {
		fmt.Printf("Process with timeout failed: %v\n", err)
	} else {
		fmt.Println("Process with timeout succeeded!")
	}

	// Showing CombinedOutput
	fmt.Println("Running a command with redirected output")
	fmt.Println()
	// Useful so the next shell command is less verbose
	out, err := distro.Command(context.Background(), "touch /root/.hushlogin").CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unexpected error: %v\nError message: %s", err, out)
	}

	// Starting Shell
	fmt.Println("\nStarting a shell in Ubuntu. Feel free to `exit <NUMBER>` to continue the demo")
	fmt.Println("")

	if err = distro.Shell(); err != nil {
		fmt.Printf("Shell exited with an error: %v\n", err)
	} else {
		fmt.Println("Shell exited with no errors")
	}
}
