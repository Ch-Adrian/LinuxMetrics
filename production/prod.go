package main

import (
    "fmt"
    "os/exec"
    "bufio"
    "github.com/segmentio/kafka-go"
    "context"
    "time"
    "log"
    "sync"
    "os"
    "io/ioutil"
    "encoding/json"
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

    stdout, err := cmd.StdoutPipe()
    if err != nil {
        fmt.Printf("Error obtaining stdout pipe: %s\n", err)
        return
    }

    stderr, err := cmd.StderrPipe()
    if err != nil {
        fmt.Printf("Error obtaining stderr pipe: %s\n", err)
        return
    }

    w := &kafka.Writer{
		Addr:     kafka.TCP(config.Kafka_address),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

    if err := cmd.Start(); err != nil {
        fmt.Printf("Error starting command: %s\n", err)
        return
    }

    stdoutScanner := bufio.NewScanner(stdout)
    stderrScanner := bufio.NewScanner(stderr)

    go func() {
        for stdoutScanner.Scan() {
            text := stdoutScanner.Text()
            fmt.Printf("stdout: %s\n", text)

            if text == config.Tools[toolIdx].Output_description {
                fmt.Println("Skipping header")
                continue
            }

            if len(config.Tools[toolIdx].Noise) != 0 && text == config.Tools[toolIdx].Noise[0] {
                continue 
            }

            dt := time.Now().Format("2006-01-02 15:04:05")

            err := w.WriteMessages(context.Background(),
                kafka.Message{
                    Key:   []byte(dt),
                    Value: []byte(text),
                },
            )

            if err != nil {
                log.Fatal("failed to write messages:", err)
            }

        }

        if err := w.Close(); err != nil {
            log.Fatal("failed to close writer:", err)
        }
    }()

    go func() {

        for stderrScanner.Scan() {
            text := stderrScanner.Text()
            fmt.Printf("stderr: %s\n", text)
        }

    }()

    if err := cmd.Wait(); err != nil {
        fmt.Printf("Command finished with error: %s\n", err)
    }

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

    // wg.Add(1)
    // go runCommand("tcpconnect","tcpconnect", config)

    // wg.Add(1)
    // go runCommand("gethostlatency","gethostlatency", config)

    // wg.Add(1)
    // go runCommand("tcpconnlat", "tcpconnlat", config)

    wg.Add(1)
    go runCommand("tcplife", "tcplife", config)

    wg.Wait()
}
