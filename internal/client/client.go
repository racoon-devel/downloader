package client

import (
	"context"
	"github.com/racoon-devel/downloader/internal/protocol"
	"github.com/racoon-devel/downloader/internal/server"
	"net"
	"time"
)

const ioTimeout = 5 * time.Second

func Execute(ctx context.Context, command protocol.Command) (response protocol.Response, err error) {
	command = command + "\n"

	var conn net.Conn
	conn, err = net.DialTimeout("unix", server.SocketAddr, ioTimeout)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	errCh := make(chan error)
	dataCh := make(chan protocol.Response)

	_ = conn.SetDeadline(time.Now().Add(ioTimeout))

	go func() {

		if err := command.Write(conn); err != nil {
			errCh <- err
			return
		}

		resp, err := protocol.ReadResponse(conn)
		if err != nil {
			errCh <- err
			return
		}
		dataCh <- resp
	}()

	select {
	case <-ctx.Done():
		err = ctx.Err()
	case err = <-errCh:
	case response = <-dataCh:
	}

	return
}
