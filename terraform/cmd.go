package terraform

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

type Cmd struct {
	stderr       io.Writer
	outputBuffer io.Writer
	tfDataDir    string
}

func NewCmd(stderr, outputBuffer io.Writer, tfDataDir string) Cmd {
	return Cmd{
		stderr:       stderr,
		outputBuffer: outputBuffer,
		tfDataDir:    tfDataDir,
	}
}

func (c Cmd) Run(stdout io.Writer, args []string, debug bool) error {
	command := exec.Command("terraform", args...)
	command.Env = append(os.Environ(), fmt.Sprintf("TF_DATA_DIR=%s", c.tfDataDir))

	if debug {
		command.Stdout = io.MultiWriter(stdout, c.outputBuffer)
		command.Stderr = io.MultiWriter(c.stderr, c.outputBuffer)
	} else {
		command.Stdout = c.outputBuffer
		command.Stderr = c.outputBuffer
	}

	return command.Run()
}
