package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type ServiceVersion struct {
	Id   string  `json:"id"`
	Rate float64 `json:"traffic_split"`
}

func (s ServiceVersion) String() string {
	ratePercent := uint(s.Rate * 100)
	return fmt.Sprintf("%s(%d%%)", s.Id, ratePercent)
}

type Appmig struct {
	project  string
	service  string
	verbose  bool
	quiet    bool
	executor Executor
}

func NewAppmig(project, service string, verbose, quiet bool, executor Executor) *Appmig {
	return &Appmig{
		project:  project,
		service:  service,
		verbose:  verbose,
		quiet:    quiet,
		executor: executor,
	}
}

func (a *Appmig) Run(version string, rates []float64, interval uint) error {
	if err := a.checkVersionExistence(version); err != nil {
		return err
	}

	currentVersion, targetVersion, err := a.getVersionsForMigration(version)
	if err != nil {
		return err
	}

	// confirm user
	fmt.Printf("\n")
	fmt.Printf("Migrate traffic: project=%s, service=%s, from=%s, to=%s\n", a.project, a.service, currentVersion.Id, targetVersion.Id)
	if !a.quiet {
		if proceed := prompt("Do you want to continue?"); !proceed {
			return nil
		}
	}
	fmt.Printf("\n")

	// migrate traffic step by step
	if err := a.migrate(currentVersion, targetVersion, rates, interval); err != nil {
		return err
	}
	fmt.Printf("\n")
	fmt.Println("Finish migration!")

	return nil
}

func (a *Appmig) checkVersionExistence(version string) error {
	_, stderr, err := a.execCommandWithMessage(fmt.Sprintf("Checking existence of version %s... ", version),
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
		return errors.New(stderr)
	}
	fmt.Printf(": OK\n")
	return nil
}

func (a *Appmig) getVersionsForMigration(version string) (*ServiceVersion, *ServiceVersion, error) {
	stdout, stderr, err := a.execCommandWithMessage("Checking current serving version... ",
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
		return nil, nil, errors.New(stderr)
	}

	var servingVersions []ServiceVersion
	if err = json.Unmarshal([]byte(stdout), &servingVersions); err != nil {
		return nil, nil, fmt.Errorf("failed to parse current serving version: %s", err)
	}
	var servingVersionStrings []string
	for _, s := range servingVersions {
		servingVersionStrings = append(servingVersionStrings, s.String())
	}
	fmt.Printf(": %s\n", strings.Join(servingVersionStrings, ", "))

	// validate serving versions
	if len(servingVersions) == 0 {
		return nil, nil, fmt.Errorf("No serving version found\n")
	}
	if len(servingVersions) == 1 && servingVersions[0].Id == version {
		return nil, nil, fmt.Errorf("Already %s is served\n", version)
	}
	if len(servingVersions) == 2 && servingVersions[0].Id != version && servingVersions[1].Id != version {
		return nil, nil, fmt.Errorf("Multiple versions are served\n")
	}
	if len(servingVersions) > 2 {
		return nil, nil, fmt.Errorf("Multiple versions are served\n")
	}

	var currentVersion ServiceVersion
	var targetVersion ServiceVersion
	if len(servingVersions) == 1 {
		currentVersion = servingVersions[0]
		targetVersion = ServiceVersion{
			Id:   version,
			Rate: 0.0,
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

	return &currentVersion, &targetVersion, nil
}

func (a *Appmig) migrate(currentVersion, targetVersion *ServiceVersion, rates []float64, interval uint) error {
	for i, rate := range rates {
		remainRate := 1.0 - rate
		if rate <= targetVersion.Rate {
			continue
		}
		currentVersion.Rate = remainRate
		targetVersion.Rate = rate

		var splits string
		if int(targetVersion.Rate) == 1 {
			splits = fmt.Sprintf("%s=1.00", targetVersion.Id)
		} else {
			splits = fmt.Sprintf("%s=%0.2f,%s=%0.2f", currentVersion.Id, currentVersion.Rate, targetVersion.Id, targetVersion.Rate)
		}
		_, stderr, err := a.execCommandWithMessage(fmt.Sprintf("Migrating from %s to %s... ", currentVersion.String(), targetVersion.String()),
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
			return fmt.Errorf("%s", stderr)
		}
		fmt.Printf(": DONE\n")

		// sleep until next step
		if i != len(rates)-1 {
			a.execFuncWithMessage(fmt.Sprintf("Waiting %d seconds... ", interval), func() {
				time.Sleep(time.Second * time.Duration(interval))
			})
			fmt.Printf("  \n")
			fmt.Printf("\n")
		}
	}
	return nil
}

func (a *Appmig) execCommandWithMessage(msg string, name string, arg ...string) (string, string, error) {
	if a.verbose {
		command := name + " " + strings.Join(arg, " ")
		fmt.Println(command)
	}

	ticker := printProgressingMessage(msg)
	stdout, stderr, err := a.executor.ExecCommand(name, arg...)
	ticker.Stop()
	time.Sleep(time.Microsecond * 500) // waiting for progressing print
	fmt.Printf("\r%s", msg)            // print message without progressing mark
	return stdout, stderr, err
}

func (a *Appmig) execFuncWithMessage(msg string, fun func()) {
	ticker := printProgressingMessage(msg)
	fun()
	ticker.Stop()
	time.Sleep(time.Microsecond * 500) // waiting for progressing print
	fmt.Printf("\r%s", msg)            // print message without progressing mark
}

// Helper
func printProgressingMessage(msg string) *time.Ticker {
	progressMarks := []string{"-", "\\", "|", "/"}
	ticker := time.NewTicker(time.Millisecond * 100)
	go func() {
		i := 0
		for {
			<-ticker.C
			mark := progressMarks[i%len(progressMarks)]
			fmt.Printf("\r%s%s", msg, mark)
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
