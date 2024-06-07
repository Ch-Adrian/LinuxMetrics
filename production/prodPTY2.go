
	// package main

	// import (
	// 	"io"
	// 	"log"
	// 	"os"
	// 	"os/exec"
	// 	"syscall"
	// 	// "golang.org/x/sys/unix"
	// )
	
	// func main() {
	// 	// Command to run the Python script
	// 	cmd := exec.Command("sudo python /usr/share/bcc/tools/tcplife")
	
	// 	// Create a pseudo-terminal
	// 	masterFD, slaveFD, err := openPty()
	// 	if err != nil {
	// 		log.Fatalf("Failed to open pseudo terminal: %v", err)
	// 	}
	// 	defer masterFD.Close()
	// 	defer slaveFD.Close()
	
	// 	// Set the slave file descriptor as the standard input, output, and error for the command
	// 	cmd.Stdin = slaveFD
	// 	cmd.Stdout = slaveFD
	// 	cmd.Stderr = slaveFD
	
	// 	// Start the command
	// 	err = cmd.Start()
	// 	if err != nil {
	// 		log.Fatalf("Failed to start command: %v", err)
	// 	}
	
	// 	// Close the slave file descriptor in the parent process
	// 	slaveFD.Close()
	
	// 	// Create a goroutine to copy the master file descriptor output to the standard output
	// 	go func() {
	// 		_, err := io.Copy(os.Stdout, masterFD)
	// 		if err != nil {
	// 			log.Fatalf("Failed to read from master file descriptor: %v", err)
	// 		}
	// 	}()
	
	// 	// Wait for the command to finish
	// 	err = cmd.Wait()
	// 	if err != nil {
	// 		log.Fatalf("Command finished with error: %v", err)
	// 	}
	// }
	
	// func openPty() (*os.File, *os.File, error) {
	// 	// Use syscall to open a pseudo-terminal
	// 	masterFD, slaveFD, err := syscall.Openpty(nil, nil)
	// 	if err != nil {
	// 		return nil, nil, err
	// 	}
	
	// 	// Convert the file descriptors to os.File
	// 	master := os.NewFile(uintptr(masterFD), "/dev/ptmx")
	// 	slave := os.NewFile(uintptr(slaveFD), "/dev/pts")
	
	// 	return master, slave, nil
	// }

package main

import (
    "fmt"
    "os"
    "os/exec"
    "github.com/creack/pty"
    "golang.org/x/term"
)

func main() {
    // Define the command to be run in the pseudo-terminal
    command := "sudo python /usr/share/bcc/tools/tcplife" // Replace this with your actual command
	// cmd := exec.Command("sudo python /usr/share/bcc/tools/tcplife")
    // Create a pseudo-terminal
	cmd := exec.Command("bash", "-c",command)
    ptmx, err := pty.Start(cmd)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to start command: %v\n", err)
        return
    }
    defer ptmx.Close()

    // Set the terminal to raw mode
    oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
    if err != nil {
        fmt.Fprintf(os.Stderr, "Failed to set terminal to raw mode: %v\n", err)
        return
    }
    defer term.Restore(int(os.Stdin.Fd()), oldState)

    // Read and print the output from the pseudo-terminal
    go func() {
        buf := make([]byte, 1024)
        for {
            n, err := ptmx.Read(buf)
            if err != nil {
                fmt.Fprintf(os.Stderr, "Failed to read from pseudo-terminal: %v\n", err)
                break
            }
            fmt.Print(string(buf[:n]))
        }
    }()

    // Copy input from stdin to the pseudo-terminal
    go func() {
        buf := make([]byte, 1024)
        for {
            n, err := os.Stdin.Read(buf)
            if err != nil {
                fmt.Fprintf(os.Stderr, "Failed to read from stdin: %v\n", err)
                break
            }
            if n > 0 {
                _, err := ptmx.Write(buf[:n])
                if err != nil {
                    fmt.Fprintf(os.Stderr, "Failed to write to pseudo-terminal: %v\n", err)
                    break
                }
            }
        }
    }()

    // Wait for the command to complete
    if err := cmd.Wait(); err != nil {
        fmt.Fprintf(os.Stderr, "Command exited with error: %v\n", err)
    }
}