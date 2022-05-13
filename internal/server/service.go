package server

import (
	"context"
	"github.com/racoon-devel/downloader/internal/api/downloader"
	"github.com/racoon-devel/downloader/internal/task"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *server) AddTask(ctx context.Context, request *downloader.AddTaskRequest) (*emptypb.Empty, error) {
	t := task.NewTask(s.ctx, request.Url)
	s.taskCh <- t
	return &emptypb.Empty{}, nil
}

func (s *server) Status(ctx context.Context, empty *emptypb.Empty) (*downloader.StatusResponse, error) {
	resp := downloader.StatusResponse{
		Stat: s.stat.get(),
	}
	return &resp, nil
}

func (s *server) Stop(ctx context.Context, empty *emptypb.Empty) (*emptypb.Empty, error) {
	s.taskCh <- nil // its meaning stop all tasks
	return &emptypb.Empty{}, nil
}

func (s *server) Done(ctx context.Context, empty *emptypb.Empty) (*emptypb.Empty, error) {
	s.cancel()
	return &emptypb.Empty{}, nil
}
