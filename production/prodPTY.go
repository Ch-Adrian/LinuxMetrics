package main

import (
    "fmt"
    "os/exec"
    // "bufio"
    // "github.com/segmentio/kafka-go"
    // "context"
    // "time"
    // "log"
    "sync"
    "syscall"
    "os"
    "io/ioutil"
    "encoding/json"
    "github.com/creack/pty"
    "golang.org/x/sys/unix"
)

type Tool struct {
    Cmd []string `json:"cmd"`
    Output_description string `json:"output_description"`
    Noise []string `json:"noise"`
    Additional []string `json:"additional"`
}

type Config struct {
    Kafka_address string `json:"kafka_address"`
    Path string `json:"path"`
    Tools []Tool `json:"tools"`
}


var wg = sync.WaitGroup{}

func runCommand(command string, topic string, config Config){
    toolIdx := 0
    for idx, tool := range config.Tools {
        if tool.Cmd[0] == command {
           toolIdx = idx
           break 
        }
    }
    fmt.Println("Running command: ", command, " and ", config.Tools[toolIdx].Cmd)

    defer wg.Done()

    cmd := exec.Command("sudo", "python", config.Path+command)
    // cmd := exec.Command("ping", "127.0.0.1")

    // stdout, err := cmd.StdoutPipe()
    // if err != nil {
    //     fmt.Printf("Error obtaining stdout pipe: %s\n", err)
    //     return
    // }

    // stderr, err := cmd.StderrPipe()
    // if err != nil {
    //     fmt.Printf("Error obtaining stderr pipe: %s\n", err)
    //     return
    // }

    // Start the command with a pseudo-terminal
	ptmx, err := pty.Start(cmd)
	if err != nil {
		fmt.Printf("Error starting command: %s\n", err)
		return
	}
	defer func() { _ = ptmx.Close() }() // Best effort close

	// Create a scanner to read the output
	// scanner := bufio.NewScanner(ptmx)

    // if err := unix.SetNonblock(int(ptmx.Fd()), true); err != nil {
	// 	fmt.Printf("Error setting non-blocking mode: %s\n", err)
	// 	return
	// }

    // w := &kafka.Writer{
	// 	Addr:     kafka.TCP(config.Kafka_address),
	// 	Topic:    topic,
	// 	Balancer: &kafka.LeastBytes{},
	// }

    // if err := cmd.Start(); err != nil {
    //     fmt.Printf("Error starting command: %s\n", err)
    //     return
    // }

    // stdoutScanner := bufio.NewScanner(stdout)
    // stderrScanner := bufio.NewScanner(stderr)

   	// Create a buffer for reading
	buffer := make([]byte, 1024)

	// Continuously read from the pseudo-terminal in non-blocking mode
	for {
		// Use select to wait for the file descriptor to be ready
		var fds unix.FdSet
		fds.Set(int(ptmx.Fd()))

		timeout := unix.Timeval{Sec: 0, Usec: 100000} // 0.1 seconds
		_, err := unix.Select(int(ptmx.Fd()+1), &fds, nil, nil, &timeout)
		if err != nil && err != syscall.EINTR {
			fmt.Printf("Error with select: %s\n", err)
			break
		}

		// Read from the file descriptor
		n, err := syscall.Read(int(ptmx.Fd()), buffer)
		if err != nil && err != syscall.EAGAIN && err != syscall.EWOULDBLOCK {
			fmt.Printf("Error reading from pseudo-terminal: %s\n", err)
			break
		}

		if n > 0 {
			fmt.Print(string(buffer[:n]))
		}

		// Check if the process has completed
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			break
		}
	}

	// Wait for the command to complete
	if err := cmd.Wait(); err != nil {
		fmt.Printf("Command finished with error: %s\n", err)
	} else {
		fmt.Println("Command finished successfully")
	}

    // go func() {

    //     for scanner.Scan() {
    //         text := scanner.Text()
    //         fmt.Printf("stdout: %s\n", text)
    //     }

        // for stdoutScanner.Scan() {
        //     text := stdoutScanner.Text()
        //     fmt.Printf("stdout: %s\n", text)

        //     if text == config.Tools[toolIdx].Output_description {
        //         fmt.Println("Skipping header")
        //         continue
        //     }

        //     if len(config.Tools[toolIdx].Noise) != 0 && text == config.Tools[toolIdx].Noise[0] {
        //         continue 
        //     }

        //     dt := time.Now().Format("2006-01-02 15:04:05")

        //     err := w.WriteMessages(context.Background(),
        //         kafka.Message{
        //             Key:   []byte(dt),
        //             Value: []byte(text),
        //         },
        //     )

        //     if err != nil {
        //         log.Fatal("failed to write messages:", err)
        //     }

        // }

    //     if err := w.Close(); err != nil {
    //         log.Fatal("failed to close writer:", err)
    //     }
    // }()

    // go func() {

    //     for stderrScanner.Scan() {
    //         text := stderrScanner.Text()
    //         fmt.Printf("stderr: %s\n", text)
    //     }

    // }()

    // if err := cmd.Wait(); err != nil {
    //     fmt.Printf("Command finished with error: %s\n", err)
    // }

}

func main(){

    jsonFile, err := os.Open("config.json")
    if err != nil {
        fmt.Println(err)
    }
    defer jsonFile.Close()

    bytes, _ := ioutil.ReadAll(jsonFile)

    var config Config

    json.Unmarshal(bytes, &config)

    // for _, tool := range config.Tools {
    //     wg.Add(1)
    //     go runCommand(tool.cmd[0], tool.cmd[0], config)
    // }

    wg.Add(1)
    go runCommand("tcpconnect","tcpconnect", config)

    // wg.Add(1)
    // go runCommand("gethostlatency","gethostlatency", config)

    // wg.Add(1)
    // go runCommand("tcpconnlat", "tcpconnlat", config)

    // wg.Add(1)
    // go runCommand("tcplife", "tcplife", config)

    wg.Wait()
}
