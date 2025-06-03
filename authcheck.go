package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Suppress HTTP/2 protocol errors
func init() {
	log.SetOutput(ioutil.Discard)
}

const version = "1.0.0"

// Banner text
const banner = `
 █████╗ ██╗   ██╗████████╗██╗  ██╗ ██████╗██╗  ██╗███████╗ ██████╗██╗  ██╗
██╔══██╗██║   ██║╚══██╔══╝██║  ██║██╔════╝██║  ██║██╔════╝██╔════╝██║ ██╔╝
███████║██║   ██║   ██║   ███████║██║     ███████║█████╗  ██║     █████╔╝ 
██╔══██║██║   ██║   ██║   ██╔══██║██║     ██╔══██║██╔══╝  ██║     ██╔═██╗ 
██║  ██║╚██████╔╝   ██║   ██║  ██║╚██████╗██║  ██║███████╗╚██████╗██║  ██╗
╚═╝  ╚═╝ ╚═════╝    ╚═╝   ╚═╝  ╚═╝ ╚═════╝╚═╝  ╚═╝╚══════╝ ╚═════╝╚═╝  ╚═╝
                                                                  v%s
Authentication Testing Tool
`

// Help text
const helpText = `
DESCRIPTION:
  Auth Check is a specialized tool for comparing HTTP responses with different authentication methods.
  It helps identify potential authentication bypass vulnerabilities by comparing responses between:
  1. Cookies and No Cookies
  2. Two different Cookie headers
  3. Bearer token and No Bearer token
  4. Two different Bearer tokens

FEATURES:
  - Tests both GET and POST methods
  - Concurrent request processing
  - Progress bar visualization
  - Filters out static files (.js, .map, .svg)
  - Automatic retry mechanism for failed requests
  - Detailed response comparison including:
    • Response status codes
    • Response body sizes
    • Side-by-side comparison

USAGE:
  auth_check [options] -f <file_with_endpoints>

OPTIONS:
  -f <file>        File containing endpoints (one per line)
  -version         Show version information
  -mode <number>   Operation mode (1-4):
                   1: Cookies -> No Cookies
                   2: Compare Two Cookies
                   3: Bearer Token -> No Bearer Token
                   4: Compare Two Bearer Tokens
  -c1 <cookie>     First cookie header
  -c2 <cookie>     Second cookie header (for mode 2)
  -t1 <token>      First bearer token
  -t2 <token>      Second bearer token (for mode 4)

EXAMPLES:
  Compare with/without cookie:
    auth_check -f endpoints.txt -mode 1 -c1 "session=abc123"

  Compare two different cookies:
    auth_check -f endpoints.txt -mode 2 -c1 "session=abc123" -c2 "session=xyz789"

  Compare with/without bearer token:
    auth_check -f endpoints.txt -mode 3 -t1 "eyJ0eXAi..."

  Compare two different bearer tokens:
    auth_check -f endpoints.txt -mode 4 -t1 "eyJ0eXAi..." -t2 "eyKhbGci..."

OUTPUT:
  The tool reports endpoints where both requests return HTTP 200 status codes,
  showing the differences in response sizes that might indicate potential issues.
`

type Result struct {
	Endpoint     string
	Method       string
	StatusCode1  int
	StatusCode2  int
	Size1        int64
	Size2        int64
	Error        error
	Description1 string
	Description2 string
}

var (
	client = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  true,
			MaxConnsPerHost:     10,
			MaxIdleConnsPerHost: 10,
		},
	}
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
)

func printColored(color, text string) {
	fmt.Printf("%s%s%s", color, text, colorReset)
}

func makeRequest(url, method string, headers map[string]string) (int, int64, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return 0, 0, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, nil // Ignore all errors
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, nil // Ignore all errors
	}

	return resp.StatusCode, int64(len(body)), nil
}

func processEndpoint(endpoint, method string, headers1, headers2 map[string]string, desc1, desc2 string) Result {
	if strings.HasSuffix(endpoint, ".js") || strings.HasSuffix(endpoint, ".map") || strings.HasSuffix(endpoint, ".svg") {
		return Result{Error: fmt.Errorf("skipped static file")}
	}

	status1, size1, err := makeRequest(endpoint, method, headers1)
	if err != nil {
		return Result{Error: err}
	}

	status2, size2, err := makeRequest(endpoint, method, headers2)
	if err != nil {
		return Result{Error: err}
	}

	return Result{
		Endpoint:     endpoint,
		Method:       method,
		StatusCode1:  status1,
		StatusCode2:  status2,
		Size1:        size1,
		Size2:        size2,
		Description1: desc1,
		Description2: desc2,
	}
}

func processFile(filename string, headers1, headers2 map[string]string, desc1, desc2 string) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	var endpoints []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		endpoints = append(endpoints, strings.TrimSpace(scanner.Text()))
	}

	results := make(chan Result)
	var wg sync.WaitGroup

	// Process endpoints concurrently
	for _, endpoint := range endpoints {
		wg.Add(2) // One for GET, one for POST
		go func(ep string) {
			defer wg.Done()
			results <- processEndpoint(ep, "GET", headers1, headers2, desc1, desc2)
		}(endpoint)
		go func(ep string) {
			defer wg.Done()
			results <- processEndpoint(ep, "POST", headers1, headers2, desc1, desc2)
		}(endpoint)
	}

	// Close results channel when all goroutines are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process results as they come in
	total := len(endpoints) * 2
	count := 0
	for result := range results {
		count++
		printProgress(count, total)

		if result.Error != nil {
			continue
		}

		// Only report if both status codes are 200 AND response sizes are identical
		if result.StatusCode1 == 200 && result.StatusCode2 == 200 && result.Size1 == result.Size2 {
			// Clear the current line and print the match
			fmt.Printf("\r\033[K") // Clear the entire line
			printColored(colorGreen, fmt.Sprintf("Potential Auth Bypass Found!\n"))
			printColored(colorGreen, fmt.Sprintf("Endpoint: %s [%s]\n", result.Endpoint, result.Method))
			printColored(colorYellow, fmt.Sprintf("%s: %d (%d bytes)\n", result.Description1, result.StatusCode1, result.Size1))
			printColored(colorYellow, fmt.Sprintf("%s: %d (%d bytes)\n", result.Description2, result.StatusCode2, result.Size2))
			fmt.Printf("\n") // Add spacing between matches
			// Print a new progress bar
			printProgress(count, total)
		}
	}
	fmt.Printf("\r\033[K") // Clear the final progress bar
	fmt.Printf("Done.\n")
}

func printProgress(current, total int) {
	width := 50
	percentage := float64(current) / float64(total)
	filled := int(float64(width) * percentage)

	fmt.Printf("\rProgress: [")
	for i := 0; i < width; i++ {
		if i < filled {
			fmt.Print("#")
		} else {
			fmt.Print("-")
		}
	}
	fmt.Printf("] %.1f%%", percentage*100)
}

func main() {
	// Parse command line flags
	fileFlag := flag.String("f", "", "File containing endpoints (one per line)")
	modeFlag := flag.Int("mode", 0, "Operation mode (1-4)")
	cookie1Flag := flag.String("c1", "", "First cookie header")
	cookie2Flag := flag.String("c2", "", "Second cookie header")
	token1Flag := flag.String("t1", "", "First bearer token")
	token2Flag := flag.String("t2", "", "Second bearer token")
	versionFlag := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *versionFlag {
		fmt.Printf(banner, version)
		return
	}

	if *fileFlag == "" || *modeFlag == 0 {
		fmt.Printf(banner, version)
		fmt.Println(helpText)
		return
	}

	var headers1, headers2 map[string]string
	var desc1, desc2 string

	switch *modeFlag {
	case 1:
		if *cookie1Flag == "" {
			fmt.Println("Error: Cookie (-c1) is required for mode 1")
			return
		}
		headers1 = map[string]string{"Cookie": *cookie1Flag}
		headers2 = map[string]string{}
		desc1 = "With Cookie"
		desc2 = "Without Cookie"

	case 2:
		if *cookie1Flag == "" || *cookie2Flag == "" {
			fmt.Println("Error: Both cookies (-c1 and -c2) are required for mode 2")
			return
		}
		headers1 = map[string]string{"Cookie": *cookie1Flag}
		headers2 = map[string]string{"Cookie": *cookie2Flag}
		desc1 = "Cookie 1"
		desc2 = "Cookie 2"

	case 3:
		if *token1Flag == "" {
			fmt.Println("Error: Bearer token (-t1) is required for mode 3")
			return
		}
		headers1 = map[string]string{"Authorization": "Bearer " + *token1Flag}
		headers2 = map[string]string{}
		desc1 = "With Token"
		desc2 = "Without Token"

	case 4:
		if *token1Flag == "" || *token2Flag == "" {
			fmt.Println("Error: Both tokens (-t1 and -t2) are required for mode 4")
			return
		}
		headers1 = map[string]string{"Authorization": "Bearer " + *token1Flag}
		headers2 = map[string]string{"Authorization": "Bearer " + *token2Flag}
		desc1 = "Token 1"
		desc2 = "Token 2"

	default:
		fmt.Println("Error: Invalid mode. Must be 1-4")
		return
	}

	processFile(*fileFlag, headers1, headers2, desc1, desc2)
}
