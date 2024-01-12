// how to run this application?
// go run main.go -source=./test.http -output=SD-83212 -retry=5 -sleep=1
// or
// you can build the binary and run it
// go build -o httpclient main.go
// ./httpclient -source=./test.http -output=SD-83212 -retry=5 -sleep=1
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// RequestData holds the parsed request information
type RequestData struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    string
}

// ReadHTTPFile parses the .HTTP file and returns RequestData
func ReadHTTPFile(filePath string) (RequestData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return RequestData{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	reqData := RequestData{
		Headers: make(map[string]string),
	}

	// Read the first line for method and URL
	if scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			return RequestData{}, fmt.Errorf("invalid request line")
		}
		reqData.Method, reqData.URL = parts[0], parts[1]
	}

	// Read headers
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue // Skip invalid header
		}
		reqData.Headers[parts[0]] = parts[1]
	}

	// Read body (if any)
	var bodyLines []string
	for scanner.Scan() {
		bodyLines = append(bodyLines, scanner.Text())
	}
	reqData.Body = strings.Join(bodyLines, "\n")

	if err := scanner.Err(); err != nil {
		return RequestData{}, err
	}

	return reqData, nil
}

// SendRequest sends an HTTP request based on RequestData
func SendRequest(reqData RequestData, retryCount int, sleepSec int) (*http.Response, error) {
	client := &http.Client{}
	var resp *http.Response
	var err error
	b := bytes.NewBufferString(reqData.Body)

	req, err := http.NewRequest(reqData.Method, reqData.URL, b)
	if err != nil {
		return nil, err
	}

	for k, v := range reqData.Headers {
		req.Header.Set(k, v)
	}

	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, err
}

// GenerateReport creates a report of the request and response
func GenerateReport(outputPath string, reqData RequestData, response *http.Response) error {
	file, err := os.Create(outputPath + "|" + fmt.Sprintf("%v", time.Now().Unix()) + "-status:" + fmt.Sprintf("%v", response.StatusCode) + ".txt")
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("Request Method: %s\nRequest URL: %s\n\n", reqData.Method, reqData.URL))
	if err != nil {
		return err
	}

	_, err = file.WriteString("Request Headers:\n")

	if err != nil {
		return err
	}

	for k, v := range reqData.Headers {
		_, err = file.WriteString(fmt.Sprintf("%s: %s\n", k, v))
		if err != nil {
			return err
		}
	}

	_, err = file.WriteString(fmt.Sprintf("\nRequest Body:\n%s\n\n", reqData.Body))
	if err != nil {
		return err
	}

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	response.Body.Close()

	_, err = file.WriteString(fmt.Sprintf("Response Status: %s\nResponse Body:\n%s\n", response.Status, string(responseBody)))
	return err
}

var (
	defaultRetry = 1
	defaultSleep = 0
)

func main() {
	source := flag.String("source", "", "Path to .http file")
	output := flag.String("output", "", "Path to output file")
	retry := flag.Int("retry", 0, "Number of retries")
	sleep := flag.Int("sleep", 0, "Sleep time between retries")

	flag.Parse()

	if *source == "" || *output == "" {
		fmt.Println("Usage: httpclient -source <path> -output <path>")
		return
	}

	if *retry == 0 {
		retry = &defaultRetry
	}

	if *sleep == 0 {
		sleep = &defaultSleep
	}

	reqData, err := ReadHTTPFile(*source)

	if err != nil {
		fmt.Println(err)
		return
	}

	for i := 0; i < *retry; i++ {
		response, err := SendRequest(reqData, *retry, *sleep)

		if err != nil {
			fmt.Println(err)
			return
		}

		err = GenerateReport(*output, reqData, response)

		if err != nil {
			fmt.Println(err)
			return
		}
		time.Sleep(time.Duration(*sleep) * time.Second)

	}

}
