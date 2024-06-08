package main

import (
    "fmt"
    "os"
    "os/exec"
    "github.com/creack/pty"
    "golang.org/x/term"
    "bufio"
    "github.com/segmentio/kafka-go"
    "context"
    "time"
    "log"
    "sync"
    // "os"
    "io/ioutil"
    "encoding/json"
	"strings"
)

const output_type_basic = "basic"
const output_type_specific = "specific"

type Tool struct {
    Cmd []string `json:"cmd"`
    Output_description string `json:"output_description"`
	Output_type string `json:"output_type"`
    Noise []string `json:"noise"`
    Additional []string `json:"additional"`
}

type Config struct {
    Kafka_address string `json:"kafka_address"`
    Path string `json:"path"`
    Tools []Tool `json:"tools"`
}

var wg = sync.WaitGroup{}

func runCommand(tool string, topic string, config Config, cmdIdx int){
	defer wg.Done()

	command := "sudo python /usr/share/bcc/tools/"+tool
	if len(config.Tools[cmdIdx].Cmd) != 1 {
		command = "sudo python /usr/share/bcc/tools/"+tool+" "+strings.Join(config.Tools[cmdIdx].Cmd[1:], " ")
	}
	amtOfColumns := len(strings.Fields(config.Tools[cmdIdx].Output_description))
	fmt.Println("Running command: ", command, " and ", amtOfColumns)

	cmd := exec.Command("bash", "-c",command)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create stderr pipe: %v\n", err)
		return
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start command: %v\n", err)
		return
	}
	defer ptmx.Close()

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set terminal to raw mode: %v\n", err)
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	w := &kafka.Writer{
		Addr:     kafka.TCP(config.Kafka_address),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buf)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read from pseudo-terminal: %v\n\r", err)
				break
			}
			output := string(buf[:n])
			output2 := strings.Split(output, "\n")
			for _, line := range output2 {

				text := strings.Trim(line, " \t\n\r\f\v")
				if text == "" {
					continue
				}

				fmt.Fprintf(os.Stdout, "%s\n\r", text)

				if text == config.Tools[cmdIdx].Output_description {
					fmt.Fprintf(os.Stdout, "Skipping header\n\r")
					continue
				}

				if len(config.Tools[cmdIdx].Noise) != 0 {
					cont := false
					for _, noise := range config.Tools[cmdIdx].Noise {
						if text == noise {
							fmt.Fprintf(os.Stdout, "Skipping noise\n\r")
							cont = true
							break;
						}
					}
					if cont {
						continue
					}
				}

				fields := strings.Fields(text)
				fmt.Fprintf(os.Stdout, "field[0]: %s\n\r", fields[0])
				if fields[0] == "[sudo]"{
					continue
				}

				if config.Tools[cmdIdx].Output_type == output_type_basic {
					amtOfColumnsInText := len(fields)
					fmt.Fprintf(os.Stdout, "amtOfColumnsInText: %d\n\r", amtOfColumnsInText)
					if amtOfColumns != amtOfColumnsInText {
						continue
					}
				}

				dt := time.Now().Format("2006-01-02 15:04:05")

				err2 := w.WriteMessages(context.Background(),
					kafka.Message{
						Key:   []byte(dt),
						Value: []byte(text),
					},
				)

				if err2 != nil {
					log.Fatal("failed to write messages:", err2)
				}
			}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintf(os.Stderr, "stderr: %s\n\r", line)
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stderr: %v\n\r", err)
		}
	}()

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

	if err := cmd.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "Command exited with error: %v\n\r", err)
	}

}

func main() {
    jsonFile, err := os.Open("config.json")
    if err != nil {
        fmt.Println(err)
    }
    defer jsonFile.Close()

    bytes, _ := ioutil.ReadAll(jsonFile)

    var config Config

    json.Unmarshal(bytes, &config)

	cmdIdx := 17

	wg.Add(1)
	go runCommand(config.Tools[cmdIdx].Cmd[0], config.Tools[cmdIdx].Cmd[0], config, cmdIdx)

    wg.Wait()
}