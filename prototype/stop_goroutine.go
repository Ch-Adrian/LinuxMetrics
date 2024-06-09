// Chat-GPT generated
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Worker function that runs in a goroutine
func worker(stopChan <-chan struct{}) {
	for {
		select {
		case <-stopChan:
			fmt.Println("Worker received stop signal")
			return
		default:
			fmt.Println("Worker is working...")
			time.Sleep(1 * time.Second)
		}
	}
}

func main() {
	// Create a channel to signal the worker to stop
	stopChan := make(chan struct{})

	// Start the worker goroutine
	go worker(stopChan)

	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for an OS signal
	sig := <-sigChan
	fmt.Printf("\nReceived signal: %s\n", sig)

	// Signal the worker to stop
	close(stopChan)

	// Give the worker some time to stop gracefully
	time.Sleep(2 * time.Second)

	fmt.Println("Exiting program.")
}
