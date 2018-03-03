package main

import (
	"strings"
	"testing"
)

type CommandRecordExecutor struct {
	commands []string
}

func (e *CommandRecordExecutor) ExecCommand(name string, arg ...string) (string, string, error) {
	command := name + " " + strings.Join(arg, " ")
	e.commands = append(e.commands, command)
	return "", "", nil
}

type MockStdoutExecutor struct {
	stdout string
}

func (e *MockStdoutExecutor) ExecCommand(name string, arg ...string) (string, string, error) {
	return e.stdout, "", nil
}

func TestMigrate(t *testing.T) {
	executor := &CommandRecordExecutor{}
	appmig := NewAppmig("myproject", "myservice", false, true, executor)

	currentVersion := &ServiceVersion{
		Id:   "v1",
		Rate: 1.0,
	}
	targetVersion := &ServiceVersion{
		Id:   "v2",
		Rate: 0.0,
	}
	rates := []float64{0.1, 0.5, 1.0}

	err := appmig.migrate(currentVersion, targetVersion, rates, 0)
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

func TestGetVersionsForMigration(t *testing.T) {
	executor := &MockStdoutExecutor{}
	appmig := NewAppmig("myproject", "myservice", false, true, executor)

	t.Run("success case", func(t *testing.T) {
		tests := []struct {
			response               string
			version                string
			expectedCurrentVersion ServiceVersion
			expectedTargetVersion  ServiceVersion
		}{
			{
				response:               `[{ "id": "v1", "traffic_split": 1.0 }]`,
				version:                "v2",
				expectedCurrentVersion: ServiceVersion{Id: "v1", Rate: 1.0},
				expectedTargetVersion:  ServiceVersion{Id: "v2", Rate: 0.0},
			},
			{
				response:               `[{ "id": "v1", "traffic_split": 0.9 }, { "id": "v2", "traffic_split": 0.1 }]`,
				version:                "v2",
				expectedCurrentVersion: ServiceVersion{Id: "v1", Rate: 0.9},
				expectedTargetVersion:  ServiceVersion{Id: "v2", Rate: 0.1},
			},
		}

		for _, test := range tests {
			executor.stdout = test.response
			currentVersion, targetVersion, err := appmig.getVersionsForMigration(test.version)
			if err != nil {
				t.Errorf("getVersionsForMigration got error: ", err)
			}
			if currentVersion.Id != test.expectedCurrentVersion.Id || currentVersion.Rate != test.expectedCurrentVersion.Rate {
				t.Errorf("invalid version: expected=%v, got=%v", test.expectedCurrentVersion, currentVersion)
			}
			if targetVersion.Id != test.expectedTargetVersion.Id || targetVersion.Rate != test.expectedTargetVersion.Rate {
				t.Errorf("invalid version: expected=%v, got=%v", test.expectedTargetVersion, targetVersion)
			}
		}
	})

	t.Run("fail case", func(t *testing.T) {
		tests := []struct {
			response string
			version  string
		}{
			{
				response: `[]`,
				version:  "v2",
			},
			{
				response: `[{ "id": "v2", "traffic_split": 1.0 }]`,
				version:  "v2",
			},
			{
				response: `[{ "id": "v1", "traffic_split": 0.5 }, { "id": "v2", "traffic_split": 0.5 }]`,
				version:  "v3",
			},
			{
				response: `[{ "id": "v1", "traffic_split": 0.4 }, { "id": "v2", "traffic_split": 0.4 }, { "id": "v3", "traffic_split": 0.2 }]`,
				version:  "v3",
			},
		}

		for _, test := range tests {
			executor.stdout = test.response
			_, _, err := appmig.getVersionsForMigration(test.version)
			if err != nil {
				t.Log(err)
			} else {
				t.Errorf("getVersionsForMigration got no error")
			}
		}
	})
}
