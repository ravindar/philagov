package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

var (
	address []string
	unitno  []string
)

type Response struct {
	SearchAddress string
	APIEndpoint   string
	Features      []Feature `json:"features"`
}

type Feature struct {
	Properties Properties `json:"properties"`
}

type Properties struct {
	Address    string `json:"street_address"`
	UnitNumber string `json:"unit_num"`
	OPANumber  string `json:"opa_account_num"`
}

// readLines reads a whole file into memory
// and returns a slice of its lines.
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func call(apiEndpoint string, searchAddress string, jobsToPerform chan Response) {
	resp, err := http.Get(apiEndpoint)

	if err != nil {
		fmt.Println("ERROR: Failed to crawl \"" + apiEndpoint + "\"")
		close(jobsToPerform)
		return
	}

	defer resp.Body.Close()

	var response Response

	err = json.NewDecoder(resp.Body).Decode(&response)

	if err != nil {
		fmt.Println("Invalid JSON response", err)
		close(jobsToPerform)
	}
	response.APIEndpoint = apiEndpoint
	response.SearchAddress = searchAddress
	jobsToPerform <- response
}

func main() {
	lines, err := readLines("address.in.txt")
	if err != nil {
		log.Fatalf("readLines: %s", err)
	}

	// Channels
	jobsToPerform := make(chan Response)
	completedJobs := make(chan int)

	fout, err := os.Create("address.out.txt")

	if err != nil {
		panic("can't write to out file")
	}

	defer fout.Close()
	w := bufio.NewWriter(fout)
	fmt.Fprintln(w, "Addresss, UnitNo(Response), OPANumber(Response), API")
	lock := sync.RWMutex{}
	numJobsToPerform := len(lines)

	// launch a goroutine for each numWorker
	for i := 0; i < numJobsToPerform; i++ {
		go worker(jobsToPerform, w, &lock, completedJobs)
	}

	for _, line := range lines {
		apiEndpoint := "https://api.phila.gov/ais_ps/v1/addresses/"
		//fmt.Println("line - ", i)

		searchAddress := ""
		unitNoRead := ""

		addresses := strings.Split(line, ",")
		if len(addresses) == 2 {
			searchAddress = strings.TrimSpace(addresses[0])
			unitNoRead = strings.TrimSpace(addresses[1])
		} else if len(addresses) == 1 {
			searchAddress = strings.TrimSpace(addresses[0])
		}

		apiEndpoint = apiEndpoint + html.EscapeString(searchAddress) + "?include_units=" + unitNoRead + "&opa_only="
		//fmt.Println("Feed URL - ", apiEndpoint)

		go call(apiEndpoint, searchAddress, jobsToPerform)
	}

	numJobsCompleted := 0
	for {
		k := <- completedJobs
		numJobsCompleted += k
		if numJobsCompleted >= numJobsToPerform {
			w.Flush()
			return
		}
	}


}

func worker(jobsToPerform chan Response, w io.Writer, lock *sync.RWMutex, completedJobs chan int) {
	for {
		msg := <-jobsToPerform
		writtenMsg := ""
		for _, featureFound := range msg.Features {
			writtenMsg += msg.SearchAddress+","
			writtenMsg += featureFound.Properties.UnitNumber+","
			writtenMsg += featureFound.Properties.OPANumber+","
			writtenMsg += msg.APIEndpoint
		}

		lock.Lock()
		fmt.Fprintln(w, writtenMsg)
		lock.Unlock()
		completedJobs <- 1

	}
}
