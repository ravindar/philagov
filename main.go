package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/html"
	"net/http"
	"bufio"
	"os"
	"log"
	"strings"
	"sync"
	"io"
)

var (
	address []string
	unitno  []string
)

type ResponseMsg struct {
	SearchAddress string
	UrlSent string
	Features []Feature `json:"features"`
}

type Feature struct {
	Properties Properties `json:"properties"`
}

type Properties struct {
	Address   string `json:"street_address"`
	UnitNumber string `json:"unit_num"`
	OPANumber string `json:"opa_account_num"`
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

func crawl(url string, searchAddress string, ch chan ResponseMsg) {
	fmt.Println("processing address ", searchAddress)
	resp, err := http.Get(url)

	if err != nil {
		fmt.Println("ERROR: Failed to crawl \"" + url + "\"")
		close(ch)
		return
	}

	defer resp.Body.Close()

	var response ResponseMsg

	err = json.NewDecoder(resp.Body).Decode(&response)

	if err != nil {
		fmt.Println("Invalid JSON response", err)
		close(ch)
	}
	response.UrlSent = url
	response.SearchAddress = searchAddress
	ch <- response
	return
}

func main() {
	lines, err := readLines("address.in.txt")
  	if err != nil {
    		log.Fatalf("readLines: %s", err)
  	}

	// Channels
	chUrls := make(chan ResponseMsg)

	fout, err := os.Create("address.out.txt")

	if err != nil {
		panic("can't write to out file")
	}

	defer fout.Close()
	w := bufio.NewWriter(fout)
	wg := sync.WaitGroup{}
	lock := sync.RWMutex{}
	numWorkers := 20
	wg.Add(numWorkers)

	// launch a goroutine for each numWorker
	for i:=0; i < numWorkers; i++ {
		go work(chUrls, w, &wg, &lock)
	}


	for i, line := range lines {
		seedUrls := "https://api.phila.gov/ais_ps/v1/addresses/"
		fmt.Println("line - ", i)

		addressRead := ""
		unitNoRead := ""

		addresses := strings.Split(line, ",")
		if len(addresses) == 2 {
			addressRead = strings.TrimSpace(addresses[0])
			unitNoRead = strings.TrimSpace(addresses[1])
		} else if len(addresses) == 1  {
			addressRead = strings.TrimSpace(addresses[0])
		}

		seedUrls = seedUrls + html.EscapeString(addressRead) + "?include_units=" + unitNoRead + "&opa_only="
		fmt.Println("Feed URL - ", seedUrls)

		go crawl(seedUrls, addressRead, chUrls)
  	}
	wg.Wait()
	w.Flush()

}

func work(ch chan ResponseMsg, w io.Writer, wg *sync.WaitGroup, lock *sync.RWMutex) {
	defer wg.Done()
	for {
		msg, ok := <- ch
		fmt.Println("WORK processing ", msg)
		if !ok {
			return
		}
		lock.Lock()
		fmt.Fprintln(w, "For Address - "+msg.SearchAddress)
		fmt.Fprintln(w, "API Call - "+msg.UrlSent)
		for n, featureFound := range msg.Features{
			fmt.Println("feature - ", n)
			fmt.Fprintln(w, "--- Response ", n)
			fmt.Fprintln(w, "Address - "+ featureFound.Properties.Address)
			fmt.Fprintln(w, "Unit Number - "+ featureFound.Properties.UnitNumber)
			fmt.Fprintln(w, "OPANumber - "+ featureFound.Properties.OPANumber)
		}
		fmt.Fprintln(w, "------")
		lock.Unlock()
	}
}