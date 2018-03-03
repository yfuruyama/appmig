package main

import (
	"bytes"
	"os/exec"
)

type Executor interface {
	ExecCommand(name string, arg ...string) (string, string, error)
}

type DefaultExecutor struct {
}

func (e *DefaultExecutor) ExecCommand(name string, arg ...string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(name, arg...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}
