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
)

var (
	address []string
	unitno  []string
)

type ResponseMsg struct {
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

func crawl(url string, ch chan ResponseMsg) {
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

	numLines := len(lines)

	for i, line := range lines {
		seedUrls := "https://api.phila.gov/ais_ps/v1/addresses/"
		fmt.Println("line - ", i)

		addresses := strings.Split(line, ",")

		seedUrls = seedUrls + html.EscapeString(strings.TrimSpace(addresses[0])) + "?include_units=" + strings.TrimSpace(addresses[1]) + "&opa_only="
		fmt.Println("Feed URL - ", seedUrls)

		go crawl(seedUrls, chUrls)
  	}

	for i:=0; i < numLines; i++ {
		msg := <- chUrls
		fmt.Println("For - ", msg.UrlSent)
		fmt.Println(msg.Features)
	}
}
