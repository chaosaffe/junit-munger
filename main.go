package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"time"

	"github.com/bmatcuk/doublestar"
	"github.com/joshdk/go-junit"
	report "github.com/jstemmer/go-junit-report/v2/junit"
)

func main() {

	inPath := flag.String("in", "", "display colorized output")
	flag.Parse()

	suites := getSuitesFromJUnitXML(*inPath)

	junitSuites := buildSuitesFromFiles(suites)

	report := buildReportOutput(junitSuites)

	out, _ := xml.MarshalIndent(report, " ", "  ")
	fmt.Println(string(out))

}

func buildReportOutput(in []junit.Suite) (out report.Testsuites) {

	for _, suite := range in {
		s := buildSuite(suite)
		out.AddSuite(s)
	}

	sort.SliceStable(out.Suites, func(i, j int) bool {
		a, err := time.ParseDuration(out.Suites[i].Time + "s")
		if err != nil {
			log.Fatalf("could not parse duration: %s\n", err)
		}
		b, err := time.ParseDuration(out.Suites[j].Time + "s")
		if err != nil {
			log.Fatalf("could not parse duration: %s\n", err)
		}
		return a > b
	})

	return out

}

func buildSuite(in junit.Suite) (out report.Testsuite) {

	out = report.Testsuite{
		Name:    in.Name,
		Package: in.Package,
	}

	for k, v := range in.Properties {
		out.AddProperty(k, v)
	}

	var suiteDuration time.Duration

	for _, test := range in.Tests {
		tc := report.Testcase{
			Name:      test.Name,
			Classname: test.Classname,
			Time:      fmt.Sprintf("%.6f", test.Duration.Seconds()),
			Status:    string(test.Status),
		}

		if test.Status == junit.StatusError || test.Status == junit.StatusFailed {
			// append fail or error data
			junitError, ok := test.Error.(junit.Error)
			if !ok {
				panic("failed to typecase interface to junit.Error")
			}

			result := &report.Result{
				Message: junitError.Message,
				Type:    junitError.Type,
				Data:    junitError.Body,
			}

			switch test.Status {
			case junit.StatusFailed:
				tc.Failure = result
			case junit.StatusError:
				tc.Error = result
			}

		}

		if test.Status == junit.StatusSkipped {
			tc.Skipped = &report.Result{}
		}

		suiteDuration += test.Duration
		out.AddTestcase(tc)
	}

	sort.SliceStable(out.Testcases, func(i, j int) bool {
		a, err := time.ParseDuration(out.Testcases[i].Time + "s")
		if err != nil {
			log.Fatalf("could not parse duration: %s\n", err)
		}
		b, err := time.ParseDuration(out.Testcases[j].Time + "s")
		if err != nil {
			log.Fatalf("could not parse duration: %s\n", err)
		}
		return a > b
	})

	out.Time = fmt.Sprintf("%.6f", suiteDuration.Seconds())

	return out
}

func loadJUnitXML(r io.Reader) []junit.Suite {

	data, err := io.ReadAll(r)
	if err != nil {
		log.Fatalf("failed to read data: %v\n", err)
	}

	suites, err := junit.Ingest(data)
	if err != nil {
		log.Fatalf("failed to ingest JUnit xml: %v\n", err)
	}

	return suites
}

func getSuitesFromJUnitXML(path string) []junit.Suite {
	filenames, err := doublestar.Glob(path)
	var suites []junit.Suite
	if err != nil {
		log.Fatalf("failed to match jUnit filename pattern: %v", err)
	}
	for _, junitFilename := range filenames {
		log.Printf("loading file %s\n", junitFilename)
		f, err := os.Open(junitFilename)
		if err != nil {
			log.Fatalf("failed to open junit xml: %v\n", err)
		}
		defer f.Close()
		//log.Printf("using test times from JUnit report %s\n", junitFilename)
		suites = append(suites, loadJUnitXML(f)...)
	}
	return suites
}

func buildSuitesFromFiles(in []junit.Suite) []junit.Suite {

	var out []junit.Suite

	temp := make(map[string][]junit.Test)

	for _, suite := range in {
		for _, test := range suite.Tests {
			fileName := test.Properties["file"]
			temp[fileName] = append(temp[fileName], test)
		}
	}

	for k, v := range temp {
		fnSuite := junit.Suite{Name: k}
		fnSuite.Tests = v
		fnSuite.Aggregate()

		out = append(out, fnSuite)
	}
	return out

}
