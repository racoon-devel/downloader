package main

import (
	"bufio"
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/racoon-devel/downloader/internal/job"
)

type healthChecker struct {
	pool            *job.Pool
	accessible      chan string
	wg              sync.WaitGroup
	outputFile      *os.File
	timeout         time.Duration
	accessibleCount uint64
	totalCount      uint64
}

const defaultTimeout uint = 20
const bytesThreshold = 8192

func main() {
	input := flag.String("i", "", "File with list of http URLs")
	output := flag.String("o", "", "File with accessible URLs")
	timeout := flag.Uint("t", defaultTimeout, "Request timeout (sec)")
	flag.Parse()
	if *input == "" || *output == "" {
		log.Fatalln("Input and output must be presented in command line arguments")
	}

	inputFile, err := os.Open(*input)
	if err != nil {
		log.Fatalf("Open input file failed: %s", err)
	}
	defer inputFile.Close()

	outputFile, err := os.Create(*output)
	if err != nil {
		log.Fatalf("Open output file failed: %s", err)
	}
	defer outputFile.Close()

	checker := NewHealthChecker(*timeout, outputFile)
	defer checker.Close()

	scanner := bufio.NewScanner(inputFile)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		checker.Check(scanner.Text())
	}
	checker.Wait()

	log.Printf("Accessible URLs: %d/%d", checker.accessibleCount, checker.totalCount)
}

func NewHealthChecker(timeout uint, outputFile *os.File) *healthChecker {
	hc := &healthChecker{
		pool:       job.NewPool(10, 100),
		accessible: make(chan string, 100),
		outputFile: outputFile,
		timeout:    time.Duration(timeout) * time.Second,
	}

	hc.wg.Add(1)
	go (func() {
		defer hc.wg.Done()
		for url := range hc.accessible {
			hc.accessibleCount++
			if _, err := hc.outputFile.WriteString(url + "\n"); err != nil {
				log.Fatalf("cannot write URL to output file: %s", err)
			}
		}
	})()

	return hc
}

func (hc *healthChecker) Check(url string) {
	hc.totalCount++
	hc.pool.Run(func() {
		ctx, _ := context.WithTimeout(context.Background(), hc.timeout)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			log.Printf("Сannot create request for URL '%s': %s", url, err)
			return
		}
		var c http.Client
		resp, err := c.Do(req)
		if err != nil {
			log.Printf("Request URL '%s' failed: %s", url, err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Request URL '%s' failed: unexpected status code %d", url, resp.StatusCode)
			return
		}

		buffer := make([]byte, bytesThreshold)
		if _, err = resp.Body.Read(buffer); err != nil {
			log.Printf("Request URL '%s' failed: %s", url, err)
			return
		}

		hc.accessible <- url
	})
}

func (hc *healthChecker) Wait() {
	hc.pool.Wait()
	close(hc.accessible)
	hc.wg.Wait()
}

func (hc *healthChecker) Close() {
	hc.pool.Close()
}
