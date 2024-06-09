// Chat-GPT generated
package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Create a channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	// Notify the channel when a SIGINT or SIGTERM is received
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create a channel to signal when the program should stop
	done := make(chan bool, 1)

	// Run a goroutine to wait for the SIGINT signal
	go func() {
		sig := <-sigChan
		fmt.Printf("\nReceived signal: %s\n", sig)
		done <- true
	}()

	fmt.Println("Press Ctrl+C to exit...")
	
	// Simulate some work
	for i := 1; i <= 10; i++ {
		fmt.Printf("Working... %d\n", i)
		time.Sleep(1 * time.Second)
	}

	// Wait for the signal handling goroutine to complete
	<-done
	fmt.Println("Exiting program.")
}
