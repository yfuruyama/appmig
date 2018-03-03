package main

import (
	"strings"
	"testing"
)

type TestExecutor struct {
	commands []string
}

func (e *TestExecutor) ExecCommand(name string, arg ...string) (string, string, error) {
	command := name + " " + strings.Join(arg, " ")
	e.commands = append(e.commands, command)
	return "", "", nil
}

func TestMigrate(t *testing.T) {
	executor := &TestExecutor{}
	appmig := NewAppmig("myproject", "myservice", false, true, executor)

	currentVersion := ServiceVersion{
		Id:   "v1",
		Rate: 1.0,
	}
	targetVersion := ServiceVersion{
		Id:   "v2",
		Rate: 0.0,
	}
	rates := []float64{0.1, 0.5, 1.0}

	err := appmig.migrate(currentVersion, targetVersion, rates, 1)
	if err != nil {
		t.Errorf("migrate got error: %s", err)
	}

	expected := []string{
		"gcloud --project=myproject app services set-traffic myservice --splits=v1=0.90,v2=0.10 --split-by=ip --quiet",
		"gcloud --project=myproject app services set-traffic myservice --splits=v1=0.50,v2=0.50 --split-by=ip --quiet",
		"gcloud --project=myproject app services set-traffic myservice --splits=v2=1.00 --split-by=ip --quiet",
	}
	if len(executor.commands) != len(expected) {
		t.Errorf("migrate failed: expected=%v, got=%v", expected, executor.commands)
	}
	for i, command := range executor.commands {
		if command != expected[i] {
			t.Errorf("migrate command failed: expected=%v, got=%v", expected[i], command)
		}
	}
}
