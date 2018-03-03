package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var usage = `Usage:
    appmig [options...]

Example:
    appmig --project=mytest --service=default --version=v2 --rate=1,5,10,25,50,75,100 --interval=30

Options:
    --project=PROJECT   (required)    Project ID
    --service=SERVICE   (required)    Service ID
    --version=VERSION   (required)    Version
    --rate=RATE         (required)    Rate, commma separated (ex: 1,5,10,25,50,75,100)
    --interval=INTERVAL               Interval Second (default: 10)
    --verbose                         Verbose Logging
    --quiet                           Disable all interactive prompts
`

type ServingVersion struct {
	Id           string  `json:"id"`
	TrafficSplit float64 `json:"traffic_split"`
}

func (s ServingVersion) String() string {
	trafficPercent := uint(s.TrafficSplit * 100)
	return fmt.Sprintf("%s(%d%%)", s.Id, trafficPercent)
}

func parseRate(rate string) ([]float64, error) {
	ratesStr := strings.Split(rate, ",")
	rates := make([]float64, 0)
	for _, rateStr := range ratesStr {
		rate, err := strconv.ParseUint(rateStr, 10, 64)
		if err != nil {
			return nil, err
		}
		if rate > 100 {
			return nil, fmt.Errorf("rate over 100")
		}
		rates = append(rates, float64(rate)/100.0)
	}
	return rates, nil
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

func main() {
	var project string
	var service string
	var version string
	var rate string
	var interval uint
	var verbose bool
	var quiet bool

	flag.StringVar(&project, "project", "", "Project ID")
	flag.StringVar(&service, "service", "", "Service ID")
	flag.StringVar(&version, "version", "", "Version")
	flag.StringVar(&rate, "rate", "", "Rate (comma separated)")
	flag.UintVar(&interval, "interval", 10, "Interval Second")
	flag.BoolVar(&verbose, "verbose", false, "Verbose logging")
	flag.BoolVar(&quiet, "quiet", false, "Disable all interactive prompts")
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	if project == "" || service == "" || version == "" || rate == "" {
		flag.Usage()
		os.Exit(1)
	}

	rates, err := parseRate(rate)
	if err != nil {
		fmt.Printf("invalid value: --rate=%s\n", rate)
		os.Exit(1)
	}

	// check version existence
	stdout, stderr, err := execCommandWithMessage(fmt.Sprintf("Checking existence of version %s...", version), verbose,
		"gcloud",
		"app",
		"versions",
		"describe",
		"--project="+project,
		"--service="+service,
		"--format=value(id)",
		version,
	)
	if err != nil {
		fmt.Printf(" %s", stderr)
		os.Exit(1)
	}
	fmt.Printf(" : OK\n")

	// check serving version
	stdout, stderr, err = execCommandWithMessage("Checking current serving version...", verbose,
		"gcloud",
		"app",
		"versions",
		"list",
		"--project="+project,
		"--service="+service,
		"--filter=version.servingStatus=SERVING AND traffic_split>0",
		"--format=json",
	)
	if err != nil {
		fmt.Printf(" %s", stderr)
		os.Exit(1)
	}

	var servingVersions []ServingVersion
	if err = json.Unmarshal([]byte(stdout), &servingVersions); err != nil {
		fmt.Printf("failed to parse current serving version: %s", err)
		os.Exit(1)
	}
	var servingVersionStrings []string
	for _, s := range servingVersions {
		servingVersionStrings = append(servingVersionStrings, s.String())
	}
	fmt.Printf(" : %s\n", strings.Join(servingVersionStrings, ", "))

	// validate serving versions
	if len(servingVersions) == 0 {
		fmt.Println("No serving version found")
		os.Exit(1)
	}
	if len(servingVersions) == 1 && servingVersions[0].Id == version {
		fmt.Printf("Already %s is serving\n", version)
		os.Exit(0)
	}
	if len(servingVersions) == 2 && servingVersions[0].Id != version && servingVersions[1].Id != version {
		fmt.Printf("Multiple versions are serving\n")
		os.Exit(1)
	}
	if len(servingVersions) > 2 {
		fmt.Printf("Multiple versions are serving\n")
		os.Exit(1)
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
	fmt.Printf("Migrate traffic: project = %s, service = %s, from = %s, to = %s\n", project, service, currentVersion.Id, targetVersion.Id)
	if !quiet {
		if proceed := prompt("Do you want to continue?"); !proceed {
			os.Exit(0)
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
		_, stderr, err := execCommandWithMessage(fmt.Sprintf("Migrating from %s to %s...", currentVersion.String(), targetVersion.String()), verbose,
			"gcloud",
			"--project="+project,
			"app",
			"services",
			"set-traffic",
			service,
			"--splits="+splits,
			"--split-by=ip",
			"--quiet",
		)
		if err != nil {
			fmt.Printf("failed to set traffic: rate=%d, error=%s", uint(nextRate)*100, stderr)
			os.Exit(1)
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
}
