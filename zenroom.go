package zenroom

import (
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

// maxString is zenroom defined buffer MAX_STRING size
const BUFSIZE = 2 * 1024 * 1024

type ZenResult struct {
	Output string
	Logs   string
}

// ZenroomExec is our primary public API method, and it is here that we call Zenroom's
// zenroom_exec_tobuf function. This method attempts to pass a required script,
// and some optional extra parameters to the Zenroom virtual machine, where
// cryptographic operations are performed with the result being returned to the
// caller. The method signature has been tweaked slightly from the original
// function defined by Zenroom; rather than making all parameters required,
// instead we have just included as a required parameter the input SCRIPT, while
// all other properties must be supplied via one of the previously defined
// Option helpers.
//
// Returns the output of the execution of the Zenroom virtual machine, or an
// error.
func ZenroomExec(script string, conf string, keys string, data string) (ZenResult, bool) {
	cmd := []string{"zenroom"}
	return wrapper(cmd, script, conf, keys, data)
}

func ZencodeExec(script string, conf string, keys string, data string) (ZenResult, bool) {
	cmd := []string{"zenroom", "-z"}
	return wrapper(cmd, script, conf, keys, data)
}

func wrapper(cmd []string, script string, conf string, keys string, data string) (ZenResult, bool) {
	if keys != "" {
		keysFile, _ := os.CreateTemp("", "tempKeys")
		defer keysFile.Close()
		defer os.Remove(keysFile.Name())
		keysFile.WriteString(keys)
		keysFile.Sync()
		cmd = append(cmd, "-k", keysFile.Name())
	}

	if data != "" {
		dataFile, _ := os.CreateTemp("", "tempData")
		defer dataFile.Close()
		defer os.Remove(dataFile.Name())
		dataFile.WriteString(data)
		dataFile.Sync()
		cmd = append(cmd, "-a", dataFile.Name())
	}

	scriptFile, _ := os.CreateTemp("", "tempScript")
	defer scriptFile.Close()
	defer os.Remove(scriptFile.Name())
	scriptFile.WriteString(script)
	scriptFile.Sync()
	cmd = append(cmd, scriptFile.Name())

	execCmd := exec.Command(cmd[0], cmd[1:]...)

	stdout, err := execCmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to create stdout pipe: %v", err)
	}

	stderr, err := execCmd.StderrPipe()
	if err != nil {
		log.Fatalf("Failed to create stderr pipe: %v", err)
	}

	err = execCmd.Start()
	if err != nil {
		log.Fatalf("Failed to start command: %v", err)
	}

	stdoutOutput := make(chan string)
	stderrOutput := make(chan string)
	go captureOutput(stdout, stdoutOutput)
	go captureOutput(stderr, stderrOutput)

	stdoutStr := <-stdoutOutput
	stderrStr := <-stderrOutput

	err = execCmd.Wait()

	return ZenResult{Output: stdoutStr, Logs: stderrStr}, err == nil

}

func captureOutput(pipe io.ReadCloser, output chan<- string) {
	defer close(output)

	buf := new(strings.Builder)
	_, err := io.Copy(buf, pipe)
	if err != nil {
		log.Printf("Failed to capture output: %v", err)
		return
	}

	output <- buf.String()
}
