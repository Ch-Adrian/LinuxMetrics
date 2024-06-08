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
    "sync"
    // "os"
    "io/ioutil"
    "encoding/json"
	"strings"
	// "reflect"
	"errors"
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
	command := "sudo python "+config.Path+strings.Join(config.Tools[cmdIdx].Cmd[0:], " ")

	amtOfColumns := len(strings.Fields(config.Tools[cmdIdx].Output_description))
	fmt.Println("Running command: ", command, " and ", amtOfColumns)

	cmd := exec.Command("bash", "-c", command)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create stderr pipe: %v\n", err)
		return nil, nil, nil, 0, nil, nil
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start command: %v\n", err)
		return nil, nil, nil, 0, nil, nil
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set terminal to raw mode: %v\n", err)
		return nil, nil, nil, 0, nil, nil
	}

	w := &kafka.Writer{
		Addr:     kafka.TCP(config.Kafka_address),
		Topic:    tool,
		Balancer: &kafka.LeastBytes{},
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
		fmt.Fprintf(os.Stderr, "Failed to read from pseudo-terminal: %v\n\r", err)
		return nil, errors.New("Failed to read from pseudo-terminal")
	}
	output := string(buf[:n])
	output2 := strings.Split(output, "\n")
	return output2, nil
}

func filterForMessageToKafka(text string, config Config, cmdIdx int, amtOfColumns int) (string, error) {
	
	if text == config.Tools[cmdIdx].Output_description {
		fmt.Fprintf(os.Stdout, "Skipping header\n\r")
		return "", errors.New("Skipping header")
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
			return "", errors.New("Skipping noise")
		}
	}

	fields := strings.Fields(text)
	fmt.Fprintf(os.Stdout, "field[0]: %s\n\r", fields[0])
	if fields[0] == "[sudo]"{
		return "", errors.New("Skipping sudo")
	}

	if config.Tools[cmdIdx].Output_type == output_type_basic {
		amtOfColumnsInText := len(fields)
		fmt.Fprintf(os.Stdout, "amtOfColumnsInText: %d\n\r", amtOfColumnsInText)
		if amtOfColumns != amtOfColumnsInText {
			return "", errors.New("Skipping different amount of columns")
		}
	}
	return text, nil
}

func sendToKafka(output string, w *kafka.Writer){
	dt := time.Now().Format("2006-01-02 15:04:05")

	if 	err := w.WriteMessages(context.Background(),
			kafka.Message{
				Key:   []byte(dt),
				Value: []byte(output),
			},
		); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write messages: %v\n\r", err)
	}
}

func writeStdOut(ptmx *os.File, config Config, cmdIdx int, amtOfColumns int, w *kafka.Writer){
	buf := make([]byte, 1024)

	for {
		lines, err := readFromPTY(buf, ptmx)
		if err != nil { return }

		for _, line := range filterNonEmptyAndConvert(lines, func(s string) bool { return s != "" }) {

			fmt.Fprintf(os.Stdout, "%s\n\r", line)

			output, filterErr := filterForMessageToKafka(line, config, cmdIdx, amtOfColumns)
			if filterErr != nil { continue }

			sendToKafka(output, w)

		}
	}
}

func printErrors(stderrPipe *os.File){
	scanner := bufio.NewScanner(stderrPipe)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Fprintf(os.Stderr, "stderr: %s\n\r", line)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading from stderr: %v\n\r", err)
	}
}

func readStdIn(ptmx *os.File){
	buf := make([]byte, 1024)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read from stdin: %v\n", err)
			return
		}
		if n > 0 {
			_, err := ptmx.Write(buf[:n])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to write to pseudo-terminal: %v\n", err)
				return
			}
		}
	}
}

func runCommand(tool string, config Config, cmdIdx int){
	defer wg.Done()

	cmd, ptmx, stderrPipe, amtOfColumns, w, oldState := prepareParameters(tool, config, cmdIdx)
	defer ptmx.Close()
	defer term.Restore(int(os.Stdin.Fd()), oldState)
	if cmd == nil { return }

	go func() {
		writeStdOut(ptmx, config, cmdIdx, amtOfColumns, w)
	}()

	go func() {
		printErrors(stderrPipe)
	}()

	go func() {
		readStdIn(ptmx)
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

	cmdIdx := 0
	wg.Add(1)
	go runCommand(config.Tools[cmdIdx].Cmd[0], config, cmdIdx)

	// for cmdIdx, tool := range config.Tools {
		// wg.Add(1)
		// go runCommand(config.Tools[cmdIdx].Cmd[0], config, cmdIdx)
	// }

    wg.Wait()
}