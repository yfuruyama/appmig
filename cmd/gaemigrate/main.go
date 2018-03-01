package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var usage = `Usage:
    gaemigrate [options...]
Options:
    --project  (required)    Project ID
    --service  (required)    Service ID
    --version  (required)    Version
    --rate     (required)    Rate
    --interval               Interval Second (default: 10)
    --dryrun                 Dry run
`

func parseRate(rate string) ([]uint64, error) {
	ratesStr := strings.Split(rate, ",")
	rates := make([]uint64, 0)
	for _, rateStr := range ratesStr {
		rate, err := strconv.ParseUint(rateStr, 10, 64)
		if err != nil {
			return nil, err
		}
		rates = append(rates, rate)
	}
	return rates, nil
}

func main() {
	var project string
	var service string
	var version string
	var rate string
	var interval uint
	var dryrun bool

	flag.StringVar(&project, "project", "", "Project ID")
	flag.StringVar(&service, "service", "", "Service ID")
	flag.StringVar(&version, "version", "", "Version")
	flag.StringVar(&rate, "rate", "", "Rate (comma separated)")
	flag.UintVar(&interval, "interval", 10, "Interval Second")
	flag.BoolVar(&dryrun, "dryrun", false, "Dry Run")
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	if project == "" || service == "" || version == "" || rate == "" {
		flag.Usage()
		os.Exit(1)
	}

	rates, err := parseRate(rate)
	if err != nil {
		flag.Usage()
		os.Exit(1)
	}

	log.Printf("project=%s, service=%s, version=%s, rates=%v, interval=%d\n", project, service, version, rates, interval)

	progressMarks := []string{"-", "\\", "|", "/"}
	go func() {
		i := 0
		for {
			mark := progressMarks[i%len(progressMarks)]
			fmt.Printf("Checking current serving version... %s\r", mark)
			time.Sleep(time.Millisecond * 100)
			i++
		}
	}()

	out, err := exec.Command("gcloud",
		"app",
		"versions",
		"list",
		"--project="+project,
		"--service="+service,
		"--filter=traffic_split=1.0 AND version.servingStatus=SERVING",
		"--format=value(version.id)",
	).Output()
	if err != nil {
		fmt.Printf("failed to get current serving version: project=%s, version=%s, error=%s", project, version, err)
		os.Exit(1)
	}
	currentVersion := strings.TrimSuffix(string(out), "\n")
	log.Printf("current version: %s\n", currentVersion)

	fmt.Printf("CURRENT: %s\n", currentVersion)
	fmt.Printf("TARGET: %s\n", version)

	for step := 0; step < len(rates); step++ {
		nextRate := rates[step]
		remainRate := 100 - nextRate
		if !dryrun {
			out, err := exec.Command("gcloud",
				"--project="+project,
				"app",
				"services",
				"set-traffic",
				service,
				fmt.Sprintf("--splits=%s=%d,%s=%d", currentVersion, remainRate, version, nextRate),
				"--split-by=ip",
				"--quiet",
			).CombinedOutput()
			if err != nil {
				// fmt.Printf("failed to set traffic: project=%s, version=%s, rate=%d, error=%s", project, version, nextRate, err)
				// os.Exit(1)
			}
			fmt.Printf("%s", out)
		}

		time.Sleep(time.Second * time.Duration(interval))
	}
}

// func
