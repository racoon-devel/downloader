package client

import (
	"net"
	"time"

	"github.com/racoon-devel/downloader/internal/api/downloader"
	"github.com/racoon-devel/downloader/internal/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Settings struct {
	Network string
	Addr    string
}

type Client struct {
	conn *grpc.ClientConn
}

func (c *Client) Connect(settings Settings) (downloader.DownloaderClient, error) {
	conn, err := grpc.Dial(settings.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
		return net.DialTimeout(settings.Network, addr, timeout)
	}), grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(utils.MaxMessageSize)))

	if err != nil {
		return nil, err
	}

	c.conn = conn

	return downloader.NewDownloaderClient(c.conn), err
}

func (c *Client) Spawn() downloader.DownloaderClient {
	return downloader.NewDownloaderClient(c.conn)
}

func (c *Client) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}
