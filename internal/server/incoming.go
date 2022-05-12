package server

import (
	"github.com/racoon-devel/downloader/internal/protocol"
	"github.com/racoon-devel/downloader/internal/task"
	"log"
	"net"
	"time"
)

const ioTimeout = 10 * time.Second

func (s *server) processIncomingConnection(conn net.Conn) {
	_ = conn.SetDeadline(time.Now().Add(ioTimeout))
	command, err := protocol.ReadCommand(conn)
	if err != nil {
		log.Printf("read from socket failed: %s", err)
		return
	}

	response := s.processCommand(command)
	_ = response.Write(conn)
}

func (s *server) processCommand(command protocol.Command) protocol.Response {
	if !command.IsValid() {
		return protocol.MakeBadResponse("Cannot recognize command")
	}

	switch command.Id() {
	case "done":
		s.cancel()
		return protocol.Ok
	case "status":
		return protocol.MakeResponse(s.stat.String())
	case "task":
		t := task.NewTask(s.ctx, command.Args()[0])
		s.taskCh <- t
	case "stop":
		s.taskCh <- nil // its meaning stop all tasks
	}

	return protocol.Ok
}
