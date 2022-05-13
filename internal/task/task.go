package task

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"
)

const readBufferSize = 65536

// Task implements HTTP stream download session
type Task struct {
	// Timeout is a socket read timeout
	Timeout time.Duration

	url    string
	ctx    context.Context
	cancel context.CancelFunc
	status Status
}

// NewTask creates initialized task
func NewTask(ctx context.Context, url string) *Task {
	t := &Task{url: url}
	t.ctx, t.cancel = context.WithCancel(ctx)
	return t
}

// Run starts async stream receiving
func (t *Task) Run(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		t.process()
	}()
}

// Status gets task state
func (t Task) Status() Status {
	return t.status
}

// Stop immediately stops task
func (t *Task) Stop() {
	t.cancel()
}

func (t *Task) process() {
	defer func() {
		t.status = StatusError
	}()

	req, err := http.NewRequestWithContext(t.ctx, http.MethodGet, t.url, nil)
	if err != nil {
		log.Printf("[%s] Create request failed: %s", t.url, err)
		return
	}

	log.Printf("[%s] Connecting...", t.url)

	// TODO: properly set timeout
	var c http.Client
	resp, err := c.Do(req)
	if err != nil {
		log.Printf("[%s] Perform request failed: %s", t.url, err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[%s] Unexpected status code: %d", t.url, resp.StatusCode)
		return
	}

	/*
		Так как мы не можем для http.Response выставить таймаут для сокета, то
		используем такой костыль для контролирования, что данные вообще приходят
	*/
	notifyCh := make(chan bool)
	if t.Timeout != 0 {
		go func() {
			for {
				select {
				case <-t.ctx.Done():
					return
				case <-notifyCh:
				case <-time.After(t.Timeout):
					log.Printf("[%s] Read timeout expired", t.url)
					t.cancel()
					return
				}
			}
		}()
	}

	log.Printf("[%s] Connected", t.url)
	t.status = StatusActive

	buffer := make([]byte, readBufferSize)
	for {
		_, err = resp.Body.Read(buffer)
		if err != nil {
			log.Printf("[%s] Read failed: %s", t.url, err)
			t.cancel()
			return
		}
		if t.Timeout != 0 {
			notifyCh <- true
		}
	}
}
