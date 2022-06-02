package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/racoon-devel/downloader/internal/api/downloader"
	"github.com/racoon-devel/downloader/internal/task"
	"google.golang.org/grpc"
)

const maxTasksPerMoment = 8192
const printStatisticInterval = 5 * time.Second
const updateStatisticInterval = 1 * time.Second

type Settings struct {
	Ctx     context.Context
	Timeout time.Duration
	Network string
	Addr    string
}

type server struct {
	downloader.UnimplementedDownloaderServer

	grpcServer *grpc.Server
	ctx        context.Context
	cancel     context.CancelFunc

	timeout time.Duration
	taskCh  chan *task.Task
	tasks   []*task.Task
	wg      sync.WaitGroup

	stat statistic
}

// Run starts gRPC server which handle user requests
func Run(settings Settings) error {
	srv := server{grpcServer: grpc.NewServer()}
	srv.ctx, srv.cancel = context.WithCancel(settings.Ctx)

	srv.tasks = make([]*task.Task, 0)
	srv.taskCh = make(chan *task.Task, maxTasksPerMoment)

	srv.timeout = settings.Timeout

	return srv.listenAndServe(&settings)
}

func (s *server) listenAndServe(settings *Settings) error {

	// bind and listen unix socket
	if settings.Network == "unix" {
		if err := os.RemoveAll(settings.Addr); err != nil {
			return fmt.Errorf("cannot recreate unix socket: %w", err)
		}
	}

	l, err := net.Listen(settings.Network, settings.Addr)
	if err != nil {
		return fmt.Errorf("listen failed: %w", err)
	}
	log.Println("Server started")

	// register callbacks for RPC server
	downloader.RegisterDownloaderServer(s.grpcServer, s)

	// start processing command channels
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.processEvents()
	}()
	defer s.wg.Wait()

	// run gRPC server
	if err = s.grpcServer.Serve(l); err != nil {
		return err
	}

	return nil
}

func (s *server) processEvents() {
	prevTime := time.Now()
	ticker := time.NewTicker(updateStatisticInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			s.grpcServer.Stop()
			return

		case t := <-s.taskCh:
			if t != nil {
				s.tasks = append(s.tasks, t)
				t.Timeout = s.timeout
				t.Run(&s.wg)
			} else {
				s.stopTasks()
			}

		case now := <-ticker.C:
			s.updateStatistic()
			if now.Sub(prevTime) >= printStatisticInterval {
				s.printStatistic()
				prevTime = now
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
