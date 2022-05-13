package server

import (
	"fmt"
	"github.com/racoon-devel/downloader/internal/task"
	"log"
	"sort"
	"sync"
)

// StatDictionary is a type of collection which consists of statistic variables
type StatDictionary map[string]uint32

func (d StatDictionary) String() string {
	out := ""
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out += fmt.Sprintf("%s: %d ", k, d[k])
	}

	return out
}

type statistic struct {
	mutex  sync.Mutex
	values StatDictionary
}

func (s *statistic) set(values StatDictionary) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.values = values
}

func (s *statistic) get() StatDictionary {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	res := StatDictionary{}
	for k, v := range s.values {
		res[k] = v
	}
	return res
}

func (s *server) updateStatistic() {
	var active uint32
	var failed uint32
	var pending uint32
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

	s.stat.set(StatDictionary{"active": active, "failed": failed, "pending": pending})
}

func (s *server) printStatistic() {
	log.Println("[Status]", s.stat.get())
}
