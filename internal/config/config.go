package config

import (
	"errors"
	"flag"
	"fmt"
)

// Default values for configuration
const (
	DefaultPort   = 5555
	DefaultPoints = 10
)

// Config holds the application configuration
type Config struct {
	IsServer    bool
	ServerAddr  string
	Port        int
	PointsToWin int
	PlayerName  string
}

// ParseArgs parses command line arguments and returns a Config
func ParseArgs(args []string) (*Config, error) {
	fs := flag.NewFlagSet("pixpong", flag.ContinueOnError)

	server := fs.Bool("server", false, "run as server")
	join := fs.String("join", "", "server address to join")
	port := fs.Int("port", DefaultPort, "port number (1-65535)")
	points := fs.Int("points", DefaultPoints, "points to win (>=1)")
	name := fs.String("name", "", "player name")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// Validate: cannot have both --server and --join
	if *server && *join != "" {
		return nil, errors.New("cannot specify both --server and --join")
	}

	// Validate: must have either --server or --join
	if !*server && *join == "" {
		return nil, errors.New("must specify either --server or --join")
	}

	// Validate port range
	if *port < 1 || *port > 65535 {
		return nil, fmt.Errorf("port must be between 1 and 65535, got %d", *port)
	}

	// Validate points
	if *points < 1 {
		return nil, fmt.Errorf("points must be at least 1, got %d", *points)
	}

	cfg := &Config{
		IsServer:    *server,
		ServerAddr:  *join,
		Port:        *port,
		PointsToWin: *points,
		PlayerName:  *name,
	}

	return cfg, nil
}
