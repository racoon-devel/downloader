package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/racoon-devel/downloader/internal/client"
	"github.com/racoon-devel/downloader/internal/protocol"
	"github.com/racoon-devel/downloader/internal/server"
	"log"
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
		runAsyncCommand(done, ctx, protocol.MakeCommand(protocol.Task, args...))

	case "status":
		runAsyncCommand(done, ctx, protocol.Status)

	case "stop":
		runAsyncCommand(done, ctx, protocol.Stop)

	case "done":
		runAsyncCommand(done, ctx, protocol.Done)

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

func runAsyncCommand(done chan<- bool, ctx context.Context, command protocol.Command) {
	runAsync(done, func() error {
		r, err := client.Execute(ctx, command)
		if err == nil {
			log.Println("[Server]", r)
		}
		return err
	})
}
