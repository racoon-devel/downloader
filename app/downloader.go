package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/racoon-devel/downloader/internal/api/downloader"
	"github.com/racoon-devel/downloader/internal/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const defaultTimeoutSec uint = 5

func main() {
	if len(os.Args) < 2 {
		printUsage()
		log.Fatal("Command must be set")
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan bool)

	command := os.Args[1]
	switch command {
	case "server":
		settings, err := parseServerArgs(os.Args[2:])
		if err != nil {
			log.Fatalf("parse server arguments failed: %s", err)
		}
		settings.Ctx = ctx
		runAsync(done, func() error {
			return server.Run(settings)
		})

	case "task":
		args := os.Args[2:]
		if len(args) != 1 {
			log.Fatalf("URL must be presented for task command")
		}
		runAsyncCommand(done, func(client downloader.DownloaderClient) error {
			_, err := client.AddTask(ctx, &downloader.AddTaskRequest{Url: args[0]})
			return err
		})

	case "status":
		runAsyncCommand(done, func(client downloader.DownloaderClient) error {
			resp, err := client.Status(ctx, &emptypb.Empty{})
			if err == nil {
				log.Println(server.StatDictionary(resp.Stat))
			}
			return err
		})

	case "stop":
		runAsyncCommand(done, func(client downloader.DownloaderClient) error {
			_, err := client.Stop(ctx, &emptypb.Empty{})
			return err
		})

	case "done":
		runAsyncCommand(done, func(client downloader.DownloaderClient) error {
			_, err := client.Done(ctx, &emptypb.Empty{})
			return err
		})

	default:
		printUsage()
		log.Fatalf("Unknown command: %s", command)
	}

	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	select {
	case <-sig:
		log.Println("Terminated")
		cancel()
		<-done
		os.Exit(2)
	case success := <-done:
		if success {
			os.Exit(0)
		} else {
			os.Exit(2)
		}
	}
}

func printUsage() {
	fmt.Println("Usage:\t./downloader server|task|status|stop|done")
	fmt.Println("Run server:\t\t./downloader server [-timeout N]")
	fmt.Println("\t\t\t\t-timeout=N read timeout for HTTP stream (sec)")
	fmt.Println("Add download task:\t./downloader task <URL>")
	fmt.Println("Print server statistic:\t./downloader status")
	fmt.Println("Stop all tasks:\t\t./downloader stop")
	fmt.Println("Teardown server:\t./downloader done")

}

func parseServerArgs(args []string) (server.Settings, error) {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	timeout := flag.Uint("timeout", defaultTimeoutSec, "read timeout for HTTP stream (sec)")

	err := fs.Parse(args)
	if err != nil {
		return server.Settings{}, err
	}

	return server.Settings{
		Timeout: time.Duration(*timeout) * time.Second,
	}, nil
}

func connect() (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(server.SocketAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
		return net.DialTimeout("unix", addr, timeout)
	}))
	return conn, err
}

func runAsync(done chan<- bool, f func() error) {
	go func() {
		if err := f(); err != nil {
			log.Println(err)
			done <- false
		} else {
			done <- true
		}
	}()
}

func runAsyncCommand(done chan<- bool, command func(client downloader.DownloaderClient) error) {
	runAsync(done, func() error {
		conn, err := connect()
		if err != nil {
			return fmt.Errorf("cannot connect to server: %w", err)
		}
		defer func() {
			_ = conn.Close()
		}()

		cli := downloader.NewDownloaderClient(conn)
		err = command(cli)
		if err != nil {
			return fmt.Errorf("cannot process command: %w", err)
		}
		log.Println("Ok")
		return nil
	})
}
