package main

import (
	"fmt"
	"net"
	"os"

	"github.com/diegok/pixpong/internal/app"
	"github.com/diegok/pixpong/internal/config"
)

func main() {
	cfg, err := config.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		printUsage()
		os.Exit(1)
	}

	if cfg.IsServer {
		showServerInfo(cfg.Port)
	}

	application := app.NewApp(cfg)
	if err := application.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  pixpong --server [options]       Start a game server")
	fmt.Fprintln(os.Stderr, "  pixpong --join <address>         Join a game server")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Options:")
	fmt.Fprintln(os.Stderr, "  --port <port>       Server port (default: 5555)")
	fmt.Fprintln(os.Stderr, "  --name <name>       Player name")
	fmt.Fprintln(os.Stderr, "  --points <n>        Points to win (default: 10)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Examples:")
	fmt.Fprintln(os.Stderr, "  pixpong --server --name Host")
	fmt.Fprintln(os.Stderr, "  pixpong --join 192.168.1.100 --name Player2")
	fmt.Fprintln(os.Stderr, "  pixpong --join localhost:5555 --name TestPlayer")
}

func showServerInfo(port int) {
	fmt.Printf("Starting PixPong server on port %d\n", port)
	fmt.Println("Players can connect using:")
	fmt.Println("")

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Printf("  pixpong --join localhost:%d\n", port)
		return
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}

		ip := ipNet.IP
		if ip.IsLoopback() || ip.To4() == nil {
			continue
		}

		fmt.Printf("  pixpong --join %s:%d\n", ip.String(), port)
	}

	fmt.Printf("  pixpong --join localhost:%d  (same machine)\n", port)
	fmt.Println("")
	fmt.Println("Press Ctrl+C to stop the server")
	fmt.Println("")
}
