package config

import (
	"testing"
)

func TestParseArgs_ServerMode(t *testing.T) {
	args := []string{"--server"}
	cfg, err := ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.IsServer {
		t.Error("expected IsServer to be true")
	}
	if cfg.Port != DefaultPort {
		t.Errorf("expected port %d, got %d", DefaultPort, cfg.Port)
	}
	if cfg.PointsToWin != DefaultPoints {
		t.Errorf("expected points %d, got %d", DefaultPoints, cfg.PointsToWin)
	}
}

func TestParseArgs_JoinMode(t *testing.T) {
	args := []string{"--join", "192.168.1.100"}
	cfg, err := ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.IsServer {
		t.Error("expected IsServer to be false")
	}
	if cfg.ServerAddr != "192.168.1.100" {
		t.Errorf("expected ServerAddr '192.168.1.100', got '%s'", cfg.ServerAddr)
	}
	if cfg.Port != DefaultPort {
		t.Errorf("expected port %d, got %d", DefaultPort, cfg.Port)
	}
}

func TestParseArgs_CustomOptions(t *testing.T) {
	args := []string{"--server", "--port", "8080", "--points", "21", "--name", "Alice"}
	cfg, err := ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.IsServer {
		t.Error("expected IsServer to be true")
	}
	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
	if cfg.PointsToWin != 21 {
		t.Errorf("expected points 21, got %d", cfg.PointsToWin)
	}
	if cfg.PlayerName != "Alice" {
		t.Errorf("expected name 'Alice', got '%s'", cfg.PlayerName)
	}
}

func TestParseArgs_JoinWithCustomOptions(t *testing.T) {
	args := []string{"--join", "localhost", "--port", "9999", "--points", "5", "--name", "Bob"}
	cfg, err := ParseArgs(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.IsServer {
		t.Error("expected IsServer to be false")
	}
	if cfg.ServerAddr != "localhost" {
		t.Errorf("expected ServerAddr 'localhost', got '%s'", cfg.ServerAddr)
	}
	if cfg.Port != 9999 {
		t.Errorf("expected port 9999, got %d", cfg.Port)
	}
	if cfg.PointsToWin != 5 {
		t.Errorf("expected points 5, got %d", cfg.PointsToWin)
	}
	if cfg.PlayerName != "Bob" {
		t.Errorf("expected name 'Bob', got '%s'", cfg.PlayerName)
	}
}

func TestParseArgs_RequiresMode(t *testing.T) {
	args := []string{"--port", "8080"}
	_, err := ParseArgs(args)
	if err == nil {
		t.Error("expected error when neither --server nor --join specified")
	}
}

func TestParseArgs_CannotBeBoth(t *testing.T) {
	args := []string{"--server", "--join", "localhost"}
	_, err := ParseArgs(args)
	if err == nil {
		t.Error("expected error when both --server and --join specified")
	}
}

func TestParseArgs_InvalidPortTooLow(t *testing.T) {
	args := []string{"--server", "--port", "0"}
	_, err := ParseArgs(args)
	if err == nil {
		t.Error("expected error for port 0")
	}
}

func TestParseArgs_InvalidPortTooHigh(t *testing.T) {
	args := []string{"--server", "--port", "65536"}
	_, err := ParseArgs(args)
	if err == nil {
		t.Error("expected error for port 65536")
	}
}

func TestParseArgs_InvalidPointsZero(t *testing.T) {
	args := []string{"--server", "--points", "0"}
	_, err := ParseArgs(args)
	if err == nil {
		t.Error("expected error for points 0")
	}
}

func TestParseArgs_InvalidPointsNegative(t *testing.T) {
	args := []string{"--server", "--points", "-5"}
	_, err := ParseArgs(args)
	if err == nil {
		t.Error("expected error for negative points")
	}
}

func TestParseArgs_ValidPortBoundaries(t *testing.T) {
	tests := []struct {
		name string
		port string
		want int
	}{
		{"minimum port", "1", 1},
		{"maximum port", "65535", 65535},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"--server", "--port", tt.port}
			cfg, err := ParseArgs(args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Port != tt.want {
				t.Errorf("expected port %d, got %d", tt.want, cfg.Port)
			}
		})
	}
}

func TestDefaultConstants(t *testing.T) {
	if DefaultPort != 5555 {
		t.Errorf("expected DefaultPort 5555, got %d", DefaultPort)
	}
	if DefaultPoints != 10 {
		t.Errorf("expected DefaultPoints 10, got %d", DefaultPoints)
	}
}
