package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/creack/pty"
	"github.com/segmentio/kafka-go"
	"golang.org/x/term"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const output_type_basic = "basic"
const output_type_specific = "specific"

type Tool struct {
	Cmd               []string `json:"cmd"`
	OutputDescription string   `json:"output_description"`
	OutputType        string   `json:"output_type"`
	Noise             []string `json:"noise"`
	Additional        []string `json:"additional"`
}

type Config struct {
	KafkaConnected bool   `json:"kafka_connected"`
	Path           string `json:"path"`
	Tools          []Tool `json:"tools"`
}

var wg = sync.WaitGroup{}

var terminals = []*os.File{}

func filterNonEmptyAndConvert(output []string, test func(string) bool) (ret []string) {
	for _, line := range output {
		text := strings.Trim(line, " \t\n\r\f\v")
		if test(text) {
			ret = append(ret, text)
		}
	}
	return
}

func prepareParameters(tool string, config Config, cmdIdx int) (*exec.Cmd, *os.File, *os.File, int, *kafka.Writer, *term.State) {
	command := config.Path + config.Tools[cmdIdx].Cmd[0]
	var args []string = config.Tools[cmdIdx].Cmd[0:]
	args[0] = command

	kafkaAddress := os.Getenv("KAFKA_ADDRESS")
	instanceId := os.Getenv("EC2_INSTANCE_ID")

	amtOfColumns := len(strings.Fields(config.Tools[cmdIdx].OutputDescription))
	log.Printf("Running command: %s and %s\n\r", command, amtOfColumns)

	cmd := exec.Command("python3", args...)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("Failed to create stderr pipe: %v\n\r", err)
		return nil, nil, nil, 0, nil, nil
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Printf("Failed to start command: %v\n\r", err)
		return nil, nil, nil, 0, nil, nil
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		log.Printf("Failed to set terminal to raw mode: %v\n\r", err)
		return nil, nil, nil, 0, nil, nil
	}

	w := &kafka.Writer{
		Addr:                   kafka.TCP(kafkaAddress),
		Topic:                  fmt.Sprintf("%s-%s", instanceId, tool),
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
	}

	stderrPipeAsOsFile, ok := stderrPipe.(*os.File)
	if ok {
		return cmd, ptmx, stderrPipeAsOsFile, amtOfColumns, w, oldState
	}
	return nil, nil, nil, 0, nil, nil
}

func readFromPTY(buf []byte, ptmx *os.File) ([]string, error) {
	n, err := ptmx.Read(buf)
	if err != nil {
		log.Printf("Failed to read from pseudo-terminal: %v\n\r", err)
		return nil, errors.New("Failed to read from pseudo-terminal")
	}
	output := string(buf[:n])
	output2 := strings.Split(output, "\n")
	return output2, nil
}

func filterForMessageToKafka(text string, config Config, cmdIdx int, amtOfColumns int) (string, error) {

	if text == config.Tools[cmdIdx].OutputDescription {
		log.Printf("Skipping header\n\r")
		return "", errors.New("Skipping header")
	}

	if len(config.Tools[cmdIdx].Noise) != 0 {
		cont := false
		for _, noise := range config.Tools[cmdIdx].Noise {
			if text == noise {
				log.Printf("Skipping noise\n\r")
				cont = true
				break
			}
		}
		if cont {
			return "", errors.New("Skipping noise")
		}
	}

	fields := strings.Fields(text)

	if config.Tools[cmdIdx].OutputType == output_type_basic {
		amtOfColumnsInText := len(fields)
		log.Printf("amtOfColumnsInText: %d\n\r", amtOfColumnsInText)
		if amtOfColumns != amtOfColumnsInText {
			return "", errors.New("Skipping different amount of columns")
		}
	}
	return text, nil
}

func sendToKafka(output string, w *kafka.Writer) {
	dt := time.Now().Format("2006-01-02 15:04:05")
	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)

	if err := w.WriteMessages(ctx,
		kafka.Message{
			Key:   []byte(dt),
			Value: []byte(output),
		},
	); err != nil {
		log.Printf("Failed to write messages: %v\n\r", err)
	}
}

func writeStdOut(ptmx *os.File, config Config, cmdIdx int, amtOfColumns int, w *kafka.Writer) {
	buf := make([]byte, 1024)

	for {
		lines, err := readFromPTY(buf, ptmx)
		if err != nil {
			return
		}

		for _, line := range filterNonEmptyAndConvert(lines, func(s string) bool { return s != "" }) {

			fmt.Fprintf(os.Stdout, "%s\n\r", line)

			if config.KafkaConnected {
				output, filterErr := filterForMessageToKafka(line, config, cmdIdx, amtOfColumns)
				if filterErr != nil {
					continue
				}

				sendToKafka(output, w)
			}
		}
	}
}

func printErrors(stderrPipe *os.File) {
	scanner := bufio.NewScanner(stderrPipe)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintf(os.Stderr, "stderr: %s\n\r", line)
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from stderr: %v\n\r", err)
	}
}

func readStdIn() {
	buf := make([]byte, 1024)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			log.Printf("Failed to read from stdin: %v\n\r", err)
			return
		}
		if n > 0 {
			for _, ptmx := range terminals {
				_, err := ptmx.Write(buf[:n])
				if err != nil {
					log.Printf("Failed to write to pseudo-terminal: %v\n\r", err)
					return
				}
			}
		}
	}
}

func runCommand(tool string, config Config, cmdIdx int) {
	defer wg.Done()

	cmd, ptmx, stderrPipe, amtOfColumns, writer, oldState := prepareParameters(tool, config, cmdIdx)
	defer ptmx.Close()
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	if cmd == nil {
		return
	}

	go writeStdOut(ptmx, config, cmdIdx, amtOfColumns, writer)

	go printErrors(stderrPipe)

	terminals = append(terminals, ptmx)

	if err := cmd.Wait(); err != nil {
		log.Printf("Command exited with error: %v\n\r", err)
	}

}

func main() {
	logFile, err := os.OpenFile("logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0777)
	defer logFile.Close()
	if err != nil {
		log.Fatal(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	args := os.Args[1:]
	err = os.Setenv("KAFKA_ADDRESS", args[0])

	if err != nil || args[0] == "" {
		log.Fatal("Cannot set address of Kafka")
	}

	err = os.Setenv("EC2_INSTANCE_ID", args[1])

	if err != nil || args[1] == "" {
		log.Fatal("Cannot set ec2 instance id")
	}

	if args[2] == "" {
		log.Fatal("Path to config not provided")
	}

	log.Printf("Connecting to kafka on address: %s", os.Getenv("KAFKA_ADDRESS"))
	log.Printf("Instance-id: %s", os.Getenv("EC2_INSTANCE_ID"))

	jsonFile, err := os.Open(args[2])
	if err != nil {
		fmt.Println(err)
		return
	}
	defer jsonFile.Close()

	bytes, _ := ioutil.ReadAll(jsonFile)

	var config Config

	json.Unmarshal(bytes, &config)

	// commandsForTest := [...]int{0, 3, 6, 7, 8, 10, 13}

	for cmdIdx, _ := range config.Tools {
		log.Printf("cmdIdx: %d cmd: %s\n\r", cmdIdx, config.Tools[cmdIdx].Cmd[0])
		wg.Add(1)
		go runCommand(config.Tools[cmdIdx].Cmd[0], config, cmdIdx)
	}

	go readStdIn()
	wg.Wait()
}
