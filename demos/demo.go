package main

import (
	"WslApi"
	"fmt"
	"syscall"
	"time"
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

	// Launching async command
	process, err := distro.Launch(`sleep 3 && echo "Good morning!"`, false, 0, syscall.Stdout, syscall.Stdout)
	if err != nil {
		panic(err)
	}
	defer process.Close()
	pStatus, pError := process.AsyncWait(time.Minute)

	// Launching a regular command (should fail as per config change)
	distro.LaunchInteractive("notepad.exe", false)

	// Managing the result of the async command
	if err = <-pError; err != nil {
		panic(err)
	}
	fmt.Printf("Exit status: %d", <-pStatus)
}
