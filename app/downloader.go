package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/racoon-devel/downloader/internal/api/downloader"
	"github.com/racoon-devel/downloader/internal/client"
	"github.com/racoon-devel/downloader/internal/server"
	"google.golang.org/protobuf/types/known/emptypb"
)

const defaultTimeoutSec uint = 0
const defaultEndpoint = "unix:///tmp/downloader.sock"

type commandLineArgs struct {
	command        string
	serverSettings *server.Settings
	clientSettings *client.Settings
	taskURL        string
}

func main() {
	var args commandLineArgs
	if err := args.Parse(os.Args); err != nil {
		printUsage()
		log.Fatalf("Parse command line arguments failed: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan bool)

	command := os.Args[1]
	switch command {
	case "server":
		args.serverSettings.Ctx = ctx
		runAsync(done, func() error {
			return server.Run(*args.serverSettings)
		})

	case "task":
		runAsyncCommand(done, *args.clientSettings, func(client downloader.DownloaderClient) error {
			_, err := client.AddTask(ctx, &downloader.AddTaskRequest{Url: args.taskURL})
			return err
		})

	case "status":
		runAsyncCommand(done, *args.clientSettings, func(client downloader.DownloaderClient) error {
			resp, err := client.Status(ctx, &emptypb.Empty{})
			if err == nil {
				log.Println(server.StatDictionary(resp.Stat))
			}
			return err
		})

	case "stop":
		runAsyncCommand(done, *args.clientSettings, func(client downloader.DownloaderClient) error {
			_, err := client.Stop(ctx, &emptypb.Empty{})
			return err
		})

	case "done":
		runAsyncCommand(done, *args.clientSettings, func(client downloader.DownloaderClient) error {
			_, err := client.Done(ctx, &emptypb.Empty{})
			return err
		})

	default:
		printUsage()
		log.Fatalf("Unknown command: %s", command)
	}

	sig := make(chan os.Signal, 1)
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

func (c *commandLineArgs) Parse(args []string) error {
	if len(args) < 2 {
		return errors.New("command must be set")
	}
	c.command = args[1]

	switch c.command {
	case "server":
		return c.parseServerArgs(args[2:])
	case "task":
		if len(args) < 3 {
			return errors.New("task URL must be set")
		}
		c.taskURL = args[2]
		return c.parseClientArgs(args[3:])

	case "status":
		fallthrough
	case "stop":
		fallthrough
	case "done":
		return c.parseClientArgs(args[2:])
	default:
		return fmt.Errorf("unsupported command: %s", c.command)
	}
}

func (c *commandLineArgs) parseServerArgs(args []string) error {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	timeout := fs.Uint("timeout", defaultTimeoutSec, "read timeout for HTTP stream (sec)")
	endpoint := fs.String("endpoint", defaultEndpoint, "endpoint to listen clients")

	err := fs.Parse(args)
	if err != nil {
		return err
	}

	network, addr, err := parseEndpoint(*endpoint)
	if err != nil {
		return err
	}

	c.serverSettings = &server.Settings{
		Timeout: time.Duration(*timeout) * time.Second,
		Network: network,
		Addr:    addr,
	}

	return nil
}

func (c *commandLineArgs) parseClientArgs(args []string) error {
	fs := flag.NewFlagSet("client", flag.ContinueOnError)
	endpoint := fs.String("endpoint", defaultEndpoint, "endpoint of server")

	err := fs.Parse(args)
	if err != nil {
		return err
	}

	network, addr, err := parseEndpoint(*endpoint)
	if err != nil {
		return err
	}

	c.clientSettings = &client.Settings{
		Network: network,
		Addr:    addr,
	}

	return nil
}

func parseEndpoint(endpoint string) (schema, addr string, err error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return
	}

	if u.Scheme != "unix" && u.Scheme != "tcp" {
		err = fmt.Errorf("unsupported endpoint schema: %s", u.Scheme)
		return
	}

	schema = u.Scheme
	if schema == "unix" {
		addr = u.Path
	} else {
		addr = u.Host
	}

	return
}

func printUsage() {
	fmt.Println("Usage:\t./downloader server|task|status|stop|done [-endpoint <endpoint>")
	fmt.Println("Run server:\t\t./downloader server [-timeout N] [-endpoint <endpoint>]")
	fmt.Println("\t\t\t\t-timeout=N read timeout for HTTP stream (sec)")
	fmt.Println("Add download task:\t./downloader task <URL> [-endpoint <endpoint>]")
	fmt.Println("Print server statistic:\t./downloader status [-endpoint <endpoint>]")
	fmt.Println("Stop all tasks:\t\t./downloader stop [-endpoint <endpoint>]")
	fmt.Println("Teardown server:\t./downloader done [-endpoint <endpoint>]")

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

func runAsyncCommand(done chan<- bool, settings client.Settings, command func(client downloader.DownloaderClient) error) {
	runAsync(done, func() error {
		c := client.Client{}
		cli, err := c.Connect(settings)
		if err != nil {
			return fmt.Errorf("connect to the server failed: %w", err)
		}
		err = command(cli)
		if err != nil {
			return fmt.Errorf("cannot process command: %w", err)
		}
		log.Println("Ok")
		return nil
	})
}
