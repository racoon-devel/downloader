package server

import (
	"fmt"
	"github.com/racoon-devel/downloader/internal/task"
	"log"
	"sync"
)

type statistic struct {
	mutex   sync.Mutex
	active  int
	failed  int
	pending int
}

func (s *statistic) set(active, failed, pending int) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.active = active
	s.failed = failed
	s.pending = pending
}

func (s *statistic) String() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return fmt.Sprintf("Active: %d, Failed: %d, Pending: %d", s.active, s.failed, s.pending)
}

func (s *server) updateStatistic() {
	active := 0
	failed := 0
	pending := 0
	for _, t := range s.tasks {
		switch t.Status() {
		case task.StatusActive:
			active++
		case task.StatusError:
			failed++
		case task.StatusConnecting:
			pending++
		}
	}

	s.stat.set(active, failed, pending)
}

func (s *server) printStatistic() {
	log.Println("[Status]", &s.stat)
}
