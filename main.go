package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"golang.org/x/net/html"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"encoding/csv"
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

func call(apiEndpoint string, searchAddress string, jobs chan Response) {
	resp, err := http.Get(apiEndpoint)

	if err != nil {
		fmt.Println("ERROR: Failed to crawl \"" + apiEndpoint + "\"")
		close(jobs)
		return
	}

	defer resp.Body.Close()

	var response Response

	err = json.NewDecoder(resp.Body).Decode(&response)

	if err != nil {
		fmt.Println("Invalid JSON response", err)
		close(jobs)
	}
	response.APIEndpoint = apiEndpoint
	response.SearchAddress = searchAddress
	jobs <- response
}

func main() {
	lines, err := readLines("address.in.txt")
	if err != nil {
		log.Fatalf("readLines: %s", err)
	}

	// Channels
	jobs := make(chan Response)

	fout, err := os.Create("address.out.txt")

	if err != nil {
		panic("can't write to out file")
	}

	defer fout.Close()
	wg := sync.WaitGroup{}
	w := csv.NewWriter(bufio.NewWriter(fout))
	w.Write([]string{"Addresss, UnitNo(Response), OPANumber(Response), API"})
	lock := sync.RWMutex{}
	numJobsToPerform := len(lines)
	wg.Add(numJobsToPerform)

	// launch a goroutine for each numWorker
	for i := 0; i < numJobsToPerform; i++ {
		go worker(jobs, w, &lock, &wg)
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

		go call(apiEndpoint, searchAddress, jobs)
	}

	wg.Wait()
	w.Flush()
}

func worker(jobs chan Response, w *csv.Writer, lock *sync.RWMutex, wg *sync.WaitGroup) {
	defer wg.Done()
	msg := <-jobs
	for _, featureFound := range msg.Features {
		lock.Lock()
		w.Write([]string{msg.SearchAddress, featureFound.Properties.UnitNumber, featureFound.Properties.OPANumber, msg.APIEndpoint})
		lock.Unlock()
	}
}
