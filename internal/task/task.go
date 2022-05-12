package task

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"
)

const readBufferSize = 65536

type Task struct {
	Timeout time.Duration
	url     string
	ctx     context.Context
	cancel  context.CancelFunc
	status  Status
}

func NewTask(ctx context.Context, url string) *Task {
	t := &Task{url: url}
	t.ctx, t.cancel = context.WithCancel(ctx)
	return t
}

func (t *Task) Run(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		t.process()
	}()
}

func (t Task) Status() Status {
	return t.status
}

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

	log.Printf("[%s] Connected", t.url)
	t.status = StatusActive

	defer resp.Body.Close()
	buffer := make([]byte, readBufferSize)
	for {
		_, err = resp.Body.Read(buffer)
		if err != nil {
			log.Printf("[%s] Read failed: %s", t.url, err)
			return
		}
	}

}
