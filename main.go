package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var usage = `Usage:
    appmig [options...]

Example:
    appmig --project=mytest --service=default --version=v2 --rate=1,5,10,25,50,75,100 --interval=30

Options:
    --project=PROJECT   (required)    Project ID
    --service=SERVICE   (required)    Service ID
    --version=VERSION   (required)    Version
    --rate=RATE         (required)    Traffic rate(%) in each step, commma separated (ex: 1,5,10,25,50,75,100)
    --interval=INTERVAL               Interval Second (default: 10)
    --verbose                         Verbose Logging
    --quiet                           Disable all interactive prompts
`

func parseRate(rate string) ([]float64, error) {
	ratesStr := strings.Split(rate, ",")
	rates := make([]float64, 0)
	for _, rateStr := range ratesStr {
		rate, err := strconv.ParseUint(rateStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("can not parse %s as integer", rateStr)
		}
		if rate > 100 {
			return nil, fmt.Errorf("rate over 100")
		}
		rates = append(rates, float64(rate)/100.0)
	}
	return rates, nil
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
		fmt.Printf("invalid `--rate` option: %s\n", err)
		os.Exit(1)
	}

	appmig := NewAppmig(project, service, verbose, quiet)
	err = appmig.Migrate(version, rates, interval)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
}
