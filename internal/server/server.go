package server

import (
	"context"
	"fmt"
	"github.com/racoon-devel/downloader/internal/task"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

const SocketAddr = "/tmp/downloader.sock"
const maxTasksPerMoment = 8192
const printStatisticInterval = 5 * time.Second
const updateStatisticInterval = 1 * time.Second

type Settings struct {
	Ctx     context.Context
	Timeout time.Duration
}

type server struct {
	ctx    context.Context
	cancel context.CancelFunc

	timeout time.Duration
	taskCh  chan *task.Task
	tasks   []*task.Task
	wg      sync.WaitGroup

	stat statistic
}

func Run(settings Settings) error {
	var srv server
	srv.ctx, srv.cancel = context.WithCancel(settings.Ctx)
	srv.tasks = make([]*task.Task, 0)
	srv.taskCh = make(chan *task.Task, maxTasksPerMoment)
	srv.timeout = settings.Timeout
	return srv.run()
}

func (s *server) run() error {
	if err := os.RemoveAll(SocketAddr); err != nil {
		return fmt.Errorf("cannot recreate unix socket: %w", err)
	}

	l, err := net.Listen("unix", SocketAddr)
	if err != nil {
		return fmt.Errorf("listen failed: %w", err)
	}
	defer l.Close()

	errChan := make(chan error)

	log.Println("Server started")

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				errChan <- err
				return
			}

			go s.processIncomingConnection(conn)
		}
	}()

	defer s.wg.Wait()

	now := time.Now()

	for {
		select {
		case err = <-errChan:
			return err

		case <-s.ctx.Done():
			return nil

		case t := <-s.taskCh:
			if t != nil {
				s.tasks = append(s.tasks, t)
				t.Timeout = s.timeout
				t.Run(&s.wg)
			} else {
				s.stopTasks()
			}

		case <-time.After(updateStatisticInterval):
			s.updateStatistic()
			if time.Since(now) >= printStatisticInterval {
				s.printStatistic()
				now = time.Now()
			}
		}
	}
}

func (s *server) stopTasks() {
	for _, t := range s.tasks {
		t.Stop()
	}
	s.tasks = make([]*task.Task, 0)
}
