package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Appmig struct {
	project string
	service string
	verbose bool
	quiet   bool
}

type ServingVersion struct {
	Id           string  `json:"id"`
	TrafficSplit float64 `json:"traffic_split"`
}

func (s ServingVersion) String() string {
	trafficPercent := uint(s.TrafficSplit * 100)
	return fmt.Sprintf("%s(%d%%)", s.Id, trafficPercent)
}

func execCommand(name string, arg ...string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(name, arg...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func execCommandWithMessage(msg string, verbose bool, name string, arg ...string) (string, string, error) {
	if verbose {
		command := name + " " + strings.Join(arg, " ")
		fmt.Println(command)
	}

	ticker := printProgressingMessage(msg)
	stdout, stderr, err := execCommand(name, arg...)
	ticker.Stop()
	time.Sleep(time.Microsecond * 500) // waiting for progressing print
	fmt.Print(msg)                     // print message without progressing mark
	return stdout, stderr, err
}

func execFuncWithMessage(msg string, fun func()) {
	ticker := printProgressingMessage(msg)
	fun()
	ticker.Stop()
	time.Sleep(time.Microsecond * 500) // waiting for progressing print
	fmt.Print(msg)                     // print message without progressing mark
}

func printProgressingMessage(msg string) *time.Ticker {
	progressMarks := []string{"-", "\\", "|", "/"}
	ticker := time.NewTicker(time.Millisecond * 100)
	go func() {
		i := 0
		for {
			<-ticker.C
			mark := progressMarks[i%len(progressMarks)]
			fmt.Printf("%s %s\r", msg, mark)
			i++
		}
	}()
	return ticker
}

func prompt(msg string) bool {
	fmt.Printf("%s [Y/n] ", msg)
	s := bufio.NewScanner(os.Stdin)
	s.Scan()
	input := s.Text()
	if input == "Y" || input == "" {
		return true
	} else {
		return false
	}
}

func NewAppmig(project, service string, verbose, quiet bool) *Appmig {
	return &Appmig{
		project: project,
		service: service,
		verbose: verbose,
		quiet:   quiet,
	}
}

func (a *Appmig) Migrate(version string, rates []float64, interval uint) error {
	// check version existence
	stdout, stderr, err := execCommandWithMessage(fmt.Sprintf("Checking existence of version %s...", version), a.verbose,
		"gcloud",
		"app",
		"versions",
		"describe",
		"--project="+a.project,
		"--service="+a.service,
		"--format=value(id)",
		version,
	)
	if err != nil {
		return fmt.Errorf(" %s", stderr)
	}
	fmt.Printf(" : OK\n")

	// check serving version
	stdout, stderr, err = execCommandWithMessage("Checking current serving version...", a.verbose,
		"gcloud",
		"app",
		"versions",
		"list",
		"--project="+a.project,
		"--service="+a.service,
		"--filter=version.servingStatus=SERVING AND traffic_split>0",
		"--format=json",
	)
	if err != nil {
		return fmt.Errorf(" %s", stderr)
	}

	var servingVersions []ServingVersion
	if err = json.Unmarshal([]byte(stdout), &servingVersions); err != nil {
		return fmt.Errorf("failed to parse current serving version: %s", err)
	}
	var servingVersionStrings []string
	for _, s := range servingVersions {
		servingVersionStrings = append(servingVersionStrings, s.String())
	}
	fmt.Printf(" : %s\n", strings.Join(servingVersionStrings, ", "))

	// validate serving versions
	if len(servingVersions) == 0 {
		return fmt.Errorf("serving version found\n")
	}
	if len(servingVersions) == 1 && servingVersions[0].Id == version {
		return fmt.Errorf("Already %s is serving\n", version)
	}
	if len(servingVersions) == 2 && servingVersions[0].Id != version && servingVersions[1].Id != version {
		return fmt.Errorf("Multiple versions are serving\n")
	}
	if len(servingVersions) > 2 {
		return fmt.Errorf("Multiple versions are serving\n")
	}

	var currentVersion ServingVersion
	var targetVersion ServingVersion
	if len(servingVersions) == 1 {
		currentVersion = servingVersions[0]
		targetVersion = ServingVersion{
			Id:           version,
			TrafficSplit: 0.0,
		}
	} else {
		if servingVersions[0].Id != version {
			currentVersion = servingVersions[0]
			targetVersion = servingVersions[1]
		} else {
			currentVersion = servingVersions[1]
			targetVersion = servingVersions[0]
		}
	}

	// confirm user
	fmt.Printf("\n")
	fmt.Printf("Migrate traffic: project = %s, service = %s, from = %s, to = %s\n", a.project, a.service, currentVersion.Id, targetVersion.Id)
	if !a.quiet {
		if proceed := prompt("Do you want to continue?"); !proceed {
			return nil
		}
	}
	fmt.Printf("\n")

	for step := 0; step < len(rates); step++ {
		nextRate := rates[step]
		remainRate := 1.0 - nextRate
		if nextRate <= targetVersion.TrafficSplit {
			continue
		}
		currentVersion.TrafficSplit = remainRate
		targetVersion.TrafficSplit = nextRate

		var splits string
		if nextRate == 1.0 {
			splits = fmt.Sprintf("%s=1.0", targetVersion.Id)
		} else {
			splits = fmt.Sprintf("%s=%f,%s=%f", currentVersion.Id, remainRate, targetVersion.Id, nextRate)
		}
		_, stderr, err := execCommandWithMessage(fmt.Sprintf("Migrating from %s to %s...", currentVersion.String(), targetVersion.String()), a.verbose,
			"gcloud",
			"--project="+a.project,
			"app",
			"services",
			"set-traffic",
			a.service,
			"--splits="+splits,
			"--split-by=ip",
			"--quiet",
		)
		if err != nil {
			return fmt.Errorf("failed to set traffic: rate=%d, error=%s", uint(nextRate)*100, stderr)
		}
		fmt.Printf(" : DONE\n")

		if step != len(rates)-1 {
			execFuncWithMessage("Waiting...", func() {
				time.Sleep(time.Second * time.Duration(interval))
			})
			fmt.Printf("  \n")
		}
	}

	fmt.Printf("\n")
	fmt.Println("Finish migration!")

	return nil
}
