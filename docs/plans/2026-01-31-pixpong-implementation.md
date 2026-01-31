# pixpong Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a multiplayer terminal Pong game where teams compete to score points by getting a ball past defenders arranged in columns.

**Architecture:** Server-authoritative model with TCP/gob networking. Single binary runs as server or client. Server runs game loop at 60 ticks/sec, broadcasts state to clients. Clients send only up/down input.

**Tech Stack:** Go 1.21+, tcell v2 (terminal UI), encoding/gob (network codec)

---

## Task 1: Project Setup

**Files:**
- Create: `go.mod`
- Create: `cmd/pixpong/main.go`

**Step 1: Initialize Go module**

Run: `go mod init github.com/diegok/pixpong`

**Step 2: Create minimal main.go**

```go
package main

import "fmt"

func main() {
	fmt.Println("pixpong")
}
```

**Step 3: Verify build**

Run: `go build ./cmd/pixpong && ./pixpong`
Expected: Prints "pixpong"

**Step 4: Commit**

```bash
git add go.mod cmd/
git commit -m "feat: initialize pixpong project"
```

---

## Task 2: Config Package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Write the failing test**

```go
package config

import "testing"

func TestParseArgs_ServerMode(t *testing.T) {
	cfg, err := ParseArgs([]string{"--server"})
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
	cfg, err := ParseArgs([]string{"--join", "192.168.1.1:5555"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.IsServer {
		t.Error("expected IsServer to be false")
	}
	if cfg.ServerAddr != "192.168.1.1:5555" {
		t.Errorf("expected server addr '192.168.1.1:5555', got '%s'", cfg.ServerAddr)
	}
}

func TestParseArgs_CustomOptions(t *testing.T) {
	cfg, err := ParseArgs([]string{"--server", "--port", "9999", "--points", "15", "--name", "TestPlayer"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 9999 {
		t.Errorf("expected port 9999, got %d", cfg.Port)
	}
	if cfg.PointsToWin != 15 {
		t.Errorf("expected points 15, got %d", cfg.PointsToWin)
	}
	if cfg.PlayerName != "TestPlayer" {
		t.Errorf("expected name 'TestPlayer', got '%s'", cfg.PlayerName)
	}
}

func TestParseArgs_RequiresMode(t *testing.T) {
	_, err := ParseArgs([]string{})
	if err == nil {
		t.Error("expected error when no mode specified")
	}
}

func TestParseArgs_CannotBeBoth(t *testing.T) {
	_, err := ParseArgs([]string{"--server", "--join", "localhost"})
	if err == nil {
		t.Error("expected error when both server and join specified")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/config/...`
Expected: FAIL (package doesn't exist)

**Step 3: Write implementation**

```go
package config

import (
	"errors"
	"flag"
	"fmt"
)

const (
	DefaultPort   = 5555
	DefaultPoints = 10
)

type Config struct {
	IsServer    bool
	ServerAddr  string
	Port        int
	PointsToWin int
	PlayerName  string
}

func ParseArgs(args []string) (*Config, error) {
	cfg := &Config{
		Port:        DefaultPort,
		PointsToWin: DefaultPoints,
	}

	fs := flag.NewFlagSet("pixpong", flag.ContinueOnError)
	fs.BoolVar(&cfg.IsServer, "server", false, "Run as server")
	fs.StringVar(&cfg.ServerAddr, "join", "", "Server address to join")
	fs.IntVar(&cfg.Port, "port", DefaultPort, "Server port")
	fs.IntVar(&cfg.PointsToWin, "points", DefaultPoints, "Points to win")
	fs.StringVar(&cfg.PlayerName, "name", "", "Player name")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if cfg.IsServer && cfg.ServerAddr != "" {
		return nil, errors.New("cannot specify both --server and --join")
	}

	if !cfg.IsServer && cfg.ServerAddr == "" {
		return nil, errors.New("must specify --server or --join <address>")
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return nil, fmt.Errorf("port must be between 1 and 65535, got %d", cfg.Port)
	}

	if cfg.PointsToWin < 1 {
		return nil, fmt.Errorf("points must be at least 1, got %d", cfg.PointsToWin)
	}

	return cfg, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/config/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add config package with CLI parsing"
```

---

## Task 3: Protocol Package - Types

**Files:**
- Create: `internal/protocol/types.go`
- Create: `internal/protocol/types_test.go`

**Step 1: Write the test**

```go
package protocol

import (
	"bytes"
	"encoding/gob"
	"testing"
)

func TestGobRegistration(t *testing.T) {
	// Test that all types can be encoded/decoded via gob
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)

	// Test Message with various payloads
	testCases := []interface{}{
		JoinRequest{PlayerName: "Test", TerminalWidth: 80, TerminalHeight: 24},
		JoinResponse{PlayerID: 1, Accepted: true},
		PlayerInput{Direction: DirUp},
		GameState{
			Tick:       100,
			Ball:       BallState{X: 10.5, Y: 5.5, VX: 1.0, VY: 0.5},
			LeftScore:  3,
			RightScore: 2,
		},
		LobbyState{Players: []LobbyPlayer{{ID: 1, Name: "P1", Color: 0}}},
		GameOverState{WinningTeam: TeamLeft, LeftScore: 10, RightScore: 5},
	}

	for _, payload := range testCases {
		buf.Reset()
		msg := Message{Type: MsgGameState, Payload: payload}
		if err := enc.Encode(&msg); err != nil {
			t.Errorf("failed to encode %T: %v", payload, err)
			continue
		}
		var decoded Message
		if err := dec.Decode(&decoded); err != nil {
			t.Errorf("failed to decode %T: %v", payload, err)
		}
	}
}

func TestDirection(t *testing.T) {
	if DirNone != 0 {
		t.Errorf("DirNone should be 0")
	}
	if DirUp != 1 {
		t.Errorf("DirUp should be 1")
	}
	if DirDown != 2 {
		t.Errorf("DirDown should be 2")
	}
}

func TestTeam(t *testing.T) {
	if TeamLeft != 0 {
		t.Errorf("TeamLeft should be 0")
	}
	if TeamRight != 1 {
		t.Errorf("TeamRight should be 1")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/protocol/...`
Expected: FAIL

**Step 3: Write implementation**

```go
package protocol

import "encoding/gob"

// Direction represents paddle movement direction
type Direction int

const (
	DirNone Direction = iota
	DirUp
	DirDown
)

// Team represents which side a player is on
type Team int

const (
	TeamLeft Team = iota
	TeamRight
)

// MessageType identifies the kind of message
type MessageType int

const (
	MsgPlayerInput MessageType = iota
	MsgGameState
	MsgLobbyState
	MsgJoinRequest
	MsgJoinResponse
	MsgStartGame
	MsgGameOver
	MsgRematchReady
	MsgRematchState
	MsgCountdown
	MsgPauseState // For score pause
)

// Message is the envelope for all network communication
type Message struct {
	Type    MessageType
	Payload interface{}
}

// PlayerInput sent from client to server
type PlayerInput struct {
	Direction Direction
}

// JoinRequest sent when client connects
type JoinRequest struct {
	PlayerName     string
	TerminalWidth  int
	TerminalHeight int
}

// JoinResponse sent by server after join
type JoinResponse struct {
	PlayerID int
	Accepted bool
	Reason   string
}

// BallState represents the ball's position and velocity
type BallState struct {
	X, Y   float64 // Position
	VX, VY float64 // Velocity
}

// PaddleState represents a player's paddle
type PaddleState struct {
	ID     int
	Team   Team
	Column int     // X position (fixed per player)
	Y      float64 // Vertical position (center of paddle)
	Height int     // Paddle height in cells
	Color  int
}

// GameState broadcast by server each tick
type GameState struct {
	Tick        int
	Ball        BallState
	Paddles     []PaddleState
	LeftScore   int
	RightScore  int
	CourtWidth  int
	CourtHeight int
	PointsToWin int
}

// LobbyState broadcast while waiting
type LobbyState struct {
	Players     []LobbyPlayer
	IsHost      bool
	CanStart    bool
	ServerAddrs []string
	PointsToWin int
}

// LobbyPlayer in waiting room
type LobbyPlayer struct {
	ID    int
	Name  string
	Color int
}

// GameOverState sent when game ends
type GameOverState struct {
	WinningTeam Team
	LeftScore   int
	RightScore  int
}

// RematchState broadcast while waiting for rematch
type RematchState struct {
	Players  []RematchPlayer
	IsHost   bool
	AllReady bool
}

// RematchPlayer in rematch waiting room
type RematchPlayer struct {
	ID    int
	Name  string
	Color int
	Ready bool
}

// Countdown broadcast before game starts
type Countdown struct {
	Seconds int
}

// PauseState broadcast during score pause
type PauseState struct {
	SecondsLeft int
	LeftScore   int
	RightScore  int
	LastScorer  Team // Which team just scored
}

func init() {
	// Register types for gob encoding
	gob.Register(PlayerInput{})
	gob.Register(JoinRequest{})
	gob.Register(JoinResponse{})
	gob.Register(GameState{})
	gob.Register(BallState{})
	gob.Register(PaddleState{})
	gob.Register(LobbyState{})
	gob.Register(LobbyPlayer{})
	gob.Register(GameOverState{})
	gob.Register(RematchState{})
	gob.Register(RematchPlayer{})
	gob.Register(Countdown{})
	gob.Register(PauseState{})
}
```

**Step 4: Run tests**

Run: `go test ./internal/protocol/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/protocol/
git commit -m "feat: add protocol types for network messages"
```

---

## Task 4: Protocol Package - Codec

**Files:**
- Create: `internal/protocol/codec.go`
- Create: `internal/protocol/codec_test.go`

**Step 1: Write the test**

```go
package protocol

import (
	"bytes"
	"testing"
)

func TestCodec_EncodeDecodeRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	codec := NewCodec(&buf)

	original := &Message{
		Type: MsgGameState,
		Payload: GameState{
			Tick: 42,
			Ball: BallState{X: 10.5, Y: 20.3, VX: 1.0, VY: -0.5},
		},
	}

	if err := codec.Encode(original); err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	decoded, err := codec.Decode()
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if decoded.Type != original.Type {
		t.Errorf("type mismatch: got %v, want %v", decoded.Type, original.Type)
	}

	state, ok := decoded.Payload.(GameState)
	if !ok {
		t.Fatalf("payload type mismatch")
	}

	if state.Tick != 42 {
		t.Errorf("tick mismatch: got %d, want 42", state.Tick)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/protocol/... -run TestCodec`
Expected: FAIL

**Step 3: Write implementation**

```go
package protocol

import (
	"encoding/gob"
	"io"
)

// Codec handles message encoding/decoding
type Codec struct {
	enc *gob.Encoder
	dec *gob.Decoder
}

// NewCodec creates a codec for the given read/writer
func NewCodec(rw io.ReadWriter) *Codec {
	return &Codec{
		enc: gob.NewEncoder(rw),
		dec: gob.NewDecoder(rw),
	}
}

// NewEncoder creates an encoder-only codec
func NewEncoder(w io.Writer) *Codec {
	return &Codec{
		enc: gob.NewEncoder(w),
	}
}

// NewDecoder creates a decoder-only codec
func NewDecoder(r io.Reader) *Codec {
	return &Codec{
		dec: gob.NewDecoder(r),
	}
}

// Encode writes a message
func (c *Codec) Encode(msg *Message) error {
	return c.enc.Encode(msg)
}

// Decode reads a message
func (c *Codec) Decode() (*Message, error) {
	var msg Message
	if err := c.dec.Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/protocol/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/protocol/
git commit -m "feat: add protocol codec for message encoding"
```

---

## Task 5: Game Package - Ball

**Files:**
- Create: `internal/game/ball.go`
- Create: `internal/game/ball_test.go`

**Step 1: Write the test**

```go
package game

import (
	"math"
	"testing"
)

func TestBall_Move(t *testing.T) {
	b := NewBall(50, 25)
	b.VX = 1.0
	b.VY = 0.5

	b.Move()

	if b.X != 51.0 {
		t.Errorf("expected X=51.0, got %f", b.X)
	}
	if b.Y != 25.5 {
		t.Errorf("expected Y=25.5, got %f", b.Y)
	}
}

func TestBall_BounceVertical(t *testing.T) {
	b := NewBall(50, 25)
	b.VY = 1.0

	b.BounceVertical()

	if b.VY != -1.0 {
		t.Errorf("expected VY=-1.0, got %f", b.VY)
	}
}

func TestBall_BounceOffPaddle(t *testing.T) {
	b := NewBall(50, 25)
	b.VX = 1.0
	b.VY = 0.0

	// Hit paddle at center - should bounce straight back
	b.BounceOffPaddle(25, 10) // paddleY=25, paddleHeight=10

	if b.VX >= 0 {
		t.Errorf("expected VX<0 after bounce, got %f", b.VX)
	}
}

func TestBall_BounceOffPaddle_Edge(t *testing.T) {
	b := NewBall(50, 20)
	b.VX = 1.0
	b.VY = 0.0

	// Hit paddle near top edge - should bounce at angle
	b.BounceOffPaddle(25, 10) // paddleY=25 (center), height=10, ball at Y=20 (top edge)

	if b.VX >= 0 {
		t.Errorf("expected VX<0 after bounce, got %f", b.VX)
	}
	if b.VY >= 0 {
		t.Errorf("expected VY<0 (upward) when hitting top edge, got %f", b.VY)
	}
}

func TestBall_SpeedUp(t *testing.T) {
	b := NewBall(50, 25)
	b.VX = 1.0
	b.VY = 0.5
	initialSpeed := math.Sqrt(b.VX*b.VX + b.VY*b.VY)

	b.SpeedUp(1.1)

	newSpeed := math.Sqrt(b.VX*b.VX + b.VY*b.VY)
	expectedSpeed := initialSpeed * 1.1

	if math.Abs(newSpeed-expectedSpeed) > 0.001 {
		t.Errorf("expected speed %f, got %f", expectedSpeed, newSpeed)
	}
}

func TestBall_Reset(t *testing.T) {
	b := NewBall(50, 25)
	b.X = 100
	b.Y = 100
	b.VX = 5
	b.VY = 5

	b.Reset(50, 25, true) // Launch toward right

	if b.X != 50 {
		t.Errorf("expected X=50, got %f", b.X)
	}
	if b.Y != 25 {
		t.Errorf("expected Y=25, got %f", b.Y)
	}
	if b.VX <= 0 {
		t.Errorf("expected VX>0 when launching right, got %f", b.VX)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/game/... -run TestBall`
Expected: FAIL

**Step 3: Write implementation**

```go
package game

import (
	"math"
	"math/rand"
)

const (
	InitialBallSpeed = 0.5
	MaxBounceAngle   = math.Pi / 3 // 60 degrees max
)

// Ball represents the game ball
type Ball struct {
	X, Y   float64
	VX, VY float64
}

// NewBall creates a ball at the given position
func NewBall(x, y float64) *Ball {
	return &Ball{
		X: x,
		Y: y,
	}
}

// Move advances the ball by its velocity
func (b *Ball) Move() {
	b.X += b.VX
	b.Y += b.VY
}

// BounceVertical reverses vertical direction (wall bounce)
func (b *Ball) BounceVertical() {
	b.VY = -b.VY
}

// BounceOffPaddle bounces the ball off a paddle, calculating angle based on hit position
// paddleY is the center Y of the paddle, paddleHeight is total height
func (b *Ball) BounceOffPaddle(paddleY float64, paddleHeight int) {
	// Calculate where on the paddle the ball hit (-1 to 1, 0 = center)
	relativeHit := (b.Y - paddleY) / (float64(paddleHeight) / 2)

	// Clamp to valid range
	if relativeHit < -1 {
		relativeHit = -1
	}
	if relativeHit > 1 {
		relativeHit = 1
	}

	// Calculate bounce angle based on hit position
	bounceAngle := relativeHit * MaxBounceAngle

	// Preserve speed
	speed := math.Sqrt(b.VX*b.VX + b.VY*b.VY)

	// Reverse horizontal direction and apply angle
	if b.VX > 0 {
		b.VX = -speed * math.Cos(bounceAngle)
	} else {
		b.VX = speed * math.Cos(bounceAngle)
	}
	b.VY = speed * math.Sin(bounceAngle)
}

// SpeedUp multiplies the ball's speed by the given factor
func (b *Ball) SpeedUp(factor float64) {
	b.VX *= factor
	b.VY *= factor
}

// Speed returns the current speed of the ball
func (b *Ball) Speed() float64 {
	return math.Sqrt(b.VX*b.VX + b.VY*b.VY)
}

// Reset places the ball at center and launches it in the specified direction
// launchRight: true = launch toward right team, false = launch toward left team
func (b *Ball) Reset(centerX, centerY float64, launchRight bool) {
	b.X = centerX
	b.Y = centerY

	// Random vertical angle between -30 and 30 degrees
	angle := (rand.Float64() - 0.5) * math.Pi / 3

	speed := InitialBallSpeed
	if launchRight {
		b.VX = speed * math.Cos(angle)
	} else {
		b.VX = -speed * math.Cos(angle)
	}
	b.VY = speed * math.Sin(angle)
}
```

**Step 4: Run tests**

Run: `go test ./internal/game/... -run TestBall`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/game/
git commit -m "feat: add ball physics with bouncing and speed control"
```

---

## Task 6: Game Package - Paddle

**Files:**
- Create: `internal/game/paddle.go`
- Create: `internal/game/paddle_test.go`

**Step 1: Write the test**

```go
package game

import (
	"testing"

	"github.com/diegok/pixpong/internal/protocol"
)

func TestPaddle_MoveUp(t *testing.T) {
	p := NewPaddle(1, protocol.TeamLeft, 5, 10)
	p.Y = 15.0
	p.CourtHeight = 30

	p.SetDirection(protocol.DirUp)
	p.Move()

	if p.Y >= 15.0 {
		t.Errorf("expected Y < 15.0 after moving up, got %f", p.Y)
	}
}

func TestPaddle_MoveDown(t *testing.T) {
	p := NewPaddle(1, protocol.TeamLeft, 5, 10)
	p.Y = 15.0
	p.CourtHeight = 30

	p.SetDirection(protocol.DirDown)
	p.Move()

	if p.Y <= 15.0 {
		t.Errorf("expected Y > 15.0 after moving down, got %f", p.Y)
	}
}

func TestPaddle_StaysInBounds_Top(t *testing.T) {
	p := NewPaddle(1, protocol.TeamLeft, 5, 10)
	p.Height = 6
	p.Y = 3.0 // Near top
	p.CourtHeight = 30

	p.SetDirection(protocol.DirUp)
	// Move many times
	for i := 0; i < 100; i++ {
		p.Move()
	}

	minY := float64(p.Height) / 2
	if p.Y < minY {
		t.Errorf("paddle went above bounds: Y=%f, minY=%f", p.Y, minY)
	}
}

func TestPaddle_StaysInBounds_Bottom(t *testing.T) {
	p := NewPaddle(1, protocol.TeamLeft, 5, 10)
	p.Height = 6
	p.Y = 27.0 // Near bottom
	p.CourtHeight = 30

	p.SetDirection(protocol.DirDown)
	// Move many times
	for i := 0; i < 100; i++ {
		p.Move()
	}

	maxY := float64(p.CourtHeight) - float64(p.Height)/2
	if p.Y > maxY {
		t.Errorf("paddle went below bounds: Y=%f, maxY=%f", p.Y, maxY)
	}
}

func TestPaddle_ContainsY(t *testing.T) {
	p := NewPaddle(1, protocol.TeamLeft, 5, 10)
	p.Y = 15.0
	p.Height = 6

	// Center should be contained
	if !p.ContainsY(15.0) {
		t.Error("center of paddle should be contained")
	}

	// Top edge should be contained
	if !p.ContainsY(12.0) {
		t.Error("top edge should be contained")
	}

	// Bottom edge should be contained
	if !p.ContainsY(18.0) {
		t.Error("bottom edge should be contained")
	}

	// Outside top should not be contained
	if p.ContainsY(11.0) {
		t.Error("above paddle should not be contained")
	}

	// Outside bottom should not be contained
	if p.ContainsY(19.0) {
		t.Error("below paddle should not be contained")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/game/... -run TestPaddle`
Expected: FAIL

**Step 3: Write implementation**

```go
package game

import "github.com/diegok/pixpong/internal/protocol"

const (
	PaddleSpeed = 0.8
)

// Paddle represents a player's paddle
type Paddle struct {
	ID          int
	Team        protocol.Team
	Column      int // X position (fixed)
	Y           float64
	Height      int
	Color       int
	Direction   protocol.Direction
	CourtHeight int
}

// NewPaddle creates a new paddle
func NewPaddle(id int, team protocol.Team, column int, color int) *Paddle {
	return &Paddle{
		ID:        id,
		Team:      team,
		Column:    column,
		Color:     color,
		Direction: protocol.DirNone,
	}
}

// SetDirection sets the paddle's movement direction
func (p *Paddle) SetDirection(dir protocol.Direction) {
	p.Direction = dir
}

// Move moves the paddle according to its direction, respecting bounds
func (p *Paddle) Move() {
	halfHeight := float64(p.Height) / 2

	switch p.Direction {
	case protocol.DirUp:
		p.Y -= PaddleSpeed
		// Clamp to top
		if p.Y < halfHeight {
			p.Y = halfHeight
		}
	case protocol.DirDown:
		p.Y += PaddleSpeed
		// Clamp to bottom
		maxY := float64(p.CourtHeight) - halfHeight
		if p.Y > maxY {
			p.Y = maxY
		}
	}
}

// ContainsY checks if the given Y coordinate is within the paddle's vertical range
func (p *Paddle) ContainsY(y float64) bool {
	halfHeight := float64(p.Height) / 2
	return y >= p.Y-halfHeight && y <= p.Y+halfHeight
}

// TopY returns the top Y coordinate of the paddle
func (p *Paddle) TopY() float64 {
	return p.Y - float64(p.Height)/2
}

// BottomY returns the bottom Y coordinate of the paddle
func (p *Paddle) BottomY() float64 {
	return p.Y + float64(p.Height)/2
}
```

**Step 4: Run tests**

Run: `go test ./internal/game/... -run TestPaddle`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/game/
git commit -m "feat: add paddle with movement and bounds checking"
```

---

## Task 7: Game Package - State

**Files:**
- Create: `internal/game/state.go`
- Create: `internal/game/state_test.go`

**Step 1: Write the test**

```go
package game

import (
	"testing"

	"github.com/diegok/pixpong/internal/protocol"
)

func TestGameState_AddPlayer(t *testing.T) {
	gs := NewGameState(80, 24, 10)

	p1 := gs.AddPlayer(1, "Player1")
	if p1 == nil {
		t.Fatal("failed to add player 1")
	}

	p2 := gs.AddPlayer(2, "Player2")
	if p2 == nil {
		t.Fatal("failed to add player 2")
	}

	if len(gs.Paddles) != 2 {
		t.Errorf("expected 2 paddles, got %d", len(gs.Paddles))
	}
}

func TestGameState_AssignTeams(t *testing.T) {
	gs := NewGameState(80, 24, 10)

	gs.AddPlayer(1, "P1")
	gs.AddPlayer(2, "P2")
	gs.AddPlayer(3, "P3")
	gs.AddPlayer(4, "P4")

	gs.AssignTeams()

	leftCount := 0
	rightCount := 0
	for _, p := range gs.Paddles {
		if p.Team == protocol.TeamLeft {
			leftCount++
		} else {
			rightCount++
		}
	}

	// Teams should be balanced (2 vs 2)
	if leftCount != 2 || rightCount != 2 {
		t.Errorf("expected 2v2, got left=%d right=%d", leftCount, rightCount)
	}
}

func TestGameState_CalculatePaddleHeight(t *testing.T) {
	gs := NewGameState(80, 24, 10)

	// 1 player per side: larger paddles
	height1 := gs.CalculatePaddleHeight(1)

	// 4 players per side: smaller paddles
	height4 := gs.CalculatePaddleHeight(4)

	if height1 <= height4 {
		t.Errorf("1 player should have larger paddle than 4: %d vs %d", height1, height4)
	}
}

func TestGameState_BallScoring(t *testing.T) {
	gs := NewGameState(80, 24, 10)

	// Ball past left edge = right team scores
	gs.Ball.X = -1
	gs.CheckScore()

	if gs.RightScore != 1 {
		t.Errorf("expected right score 1, got %d", gs.RightScore)
	}

	// Reset and test other side
	gs.Ball.Reset(40, 12, true)
	gs.Ball.X = 81
	gs.CheckScore()

	if gs.LeftScore != 1 {
		t.Errorf("expected left score 1, got %d", gs.LeftScore)
	}
}

func TestGameState_WinCondition(t *testing.T) {
	gs := NewGameState(80, 24, 5) // 5 points to win
	gs.LeftScore = 5

	if !gs.IsGameOver() {
		t.Error("game should be over when team reaches winning score")
	}

	winner := gs.GetWinner()
	if winner != protocol.TeamLeft {
		t.Errorf("expected left team to win, got %v", winner)
	}
}

func TestGameState_SpeedCap(t *testing.T) {
	gs := NewGameState(80, 24, 10)

	// With 1 player per side, cap should be lower
	cap1 := gs.GetSpeedCap(1)

	// With 4 players per side, cap should be higher
	cap4 := gs.GetSpeedCap(4)

	if cap1 >= cap4 {
		t.Errorf("4 players should allow higher speed cap: 1=%f, 4=%f", cap1, cap4)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/game/... -run TestGameState`
Expected: FAIL

**Step 3: Write implementation**

```go
package game

import (
	"math/rand"

	"github.com/diegok/pixpong/internal/protocol"
)

const (
	TickRate           = 60
	BaseSpeedCap       = 1.5
	SpeedCapPerPlayer  = 0.3
	SpeedIncrement     = 1.05 // 5% speed increase per hit
	BasePaddleHeight   = 8
	MinPaddleHeight    = 3
	PaddleHeightPerPlayer = 1
)

// GameState represents the complete game state
type GameState struct {
	Width       int
	Height      int
	Ball        *Ball
	Paddles     []*Paddle
	LeftScore   int
	RightScore  int
	PointsToWin int
	Tick        int
	Paused      bool
	PauseTicksLeft int
	LastScorer  protocol.Team
}

// NewGameState creates a new game state
func NewGameState(width, height, pointsToWin int) *GameState {
	gs := &GameState{
		Width:       width,
		Height:      height,
		Ball:        NewBall(float64(width)/2, float64(height)/2),
		Paddles:     make([]*Paddle, 0),
		PointsToWin: pointsToWin,
	}
	return gs
}

// AddPlayer adds a new player paddle
func (gs *GameState) AddPlayer(id int, name string) *Paddle {
	color := (id - 1) % 8
	paddle := NewPaddle(id, protocol.TeamLeft, 0, color) // Team assigned later
	paddle.CourtHeight = gs.Height
	gs.Paddles = append(gs.Paddles, paddle)
	return paddle
}

// GetPaddle returns the paddle with the given ID
func (gs *GameState) GetPaddle(id int) *Paddle {
	for _, p := range gs.Paddles {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// AssignTeams randomly assigns players to teams and positions columns
func (gs *GameState) AssignTeams() {
	// Shuffle paddles for random assignment
	rand.Shuffle(len(gs.Paddles), func(i, j int) {
		gs.Paddles[i], gs.Paddles[j] = gs.Paddles[j], gs.Paddles[i]
	})

	// Split into teams
	leftCount := len(gs.Paddles) / 2
	rightCount := len(gs.Paddles) - leftCount

	var leftPaddles, rightPaddles []*Paddle

	for i, p := range gs.Paddles {
		if i < leftCount {
			p.Team = protocol.TeamLeft
			leftPaddles = append(leftPaddles, p)
		} else {
			p.Team = protocol.TeamRight
			rightPaddles = append(rightPaddles, p)
		}
	}

	// Calculate paddle height based on team size
	maxPlayersPerSide := leftCount
	if rightCount > maxPlayersPerSide {
		maxPlayersPerSide = rightCount
	}
	paddleHeight := gs.CalculatePaddleHeight(maxPlayersPerSide)

	// Position left team columns (evenly distributed in left half)
	gs.positionTeamColumns(leftPaddles, 1, gs.Width/2-2, paddleHeight)

	// Position right team columns (evenly distributed in right half)
	gs.positionTeamColumns(rightPaddles, gs.Width/2+2, gs.Width-2, paddleHeight)
}

// positionTeamColumns distributes paddles evenly across the given X range
func (gs *GameState) positionTeamColumns(paddles []*Paddle, minX, maxX, height int) {
	if len(paddles) == 0 {
		return
	}

	spacing := (maxX - minX) / (len(paddles) + 1)

	for i, p := range paddles {
		p.Column = minX + spacing*(i+1)
		p.Height = height
		p.Y = float64(gs.Height) / 2
	}
}

// CalculatePaddleHeight returns paddle height based on players per side
func (gs *GameState) CalculatePaddleHeight(playersPerSide int) int {
	height := BasePaddleHeight - (playersPerSide-1)*PaddleHeightPerPlayer
	if height < MinPaddleHeight {
		height = MinPaddleHeight
	}
	return height
}

// GetSpeedCap returns the maximum ball speed based on players per side
func (gs *GameState) GetSpeedCap(playersPerSide int) float64 {
	return BaseSpeedCap + float64(playersPerSide-1)*SpeedCapPerPlayer
}

// ProcessInput handles a direction input from a player
func (gs *GameState) ProcessInput(playerID int, dir protocol.Direction) {
	paddle := gs.GetPaddle(playerID)
	if paddle != nil {
		paddle.SetDirection(dir)
	}
}

// Tick advances the game state by one tick
func (gs *GameState) Tick() {
	gs.Tick++

	if gs.Paused {
		gs.PauseTicksLeft--
		if gs.PauseTicksLeft <= 0 {
			gs.Paused = false
			// Launch ball toward the team that scored
			gs.Ball.Reset(float64(gs.Width)/2, float64(gs.Height)/2, gs.LastScorer == protocol.TeamLeft)
		}
		return
	}

	// Move paddles
	for _, p := range gs.Paddles {
		p.Move()
	}

	// Move ball
	gs.Ball.Move()

	// Check wall bounces (top/bottom)
	if gs.Ball.Y <= 0 || gs.Ball.Y >= float64(gs.Height)-1 {
		gs.Ball.BounceVertical()
		// Clamp to bounds
		if gs.Ball.Y <= 0 {
			gs.Ball.Y = 0.1
		}
		if gs.Ball.Y >= float64(gs.Height)-1 {
			gs.Ball.Y = float64(gs.Height) - 1.1
		}
	}

	// Check paddle collisions
	gs.checkPaddleCollisions()

	// Check scoring
	gs.CheckScore()
}

// checkPaddleCollisions checks and handles ball-paddle collisions
func (gs *GameState) checkPaddleCollisions() {
	for _, p := range gs.Paddles {
		// Check if ball is at paddle's column
		ballX := int(gs.Ball.X + 0.5)
		if ballX != p.Column {
			continue
		}

		// Check if ball is within paddle's vertical range
		if p.ContainsY(gs.Ball.Y) {
			gs.Ball.BounceOffPaddle(p.Y, p.Height)

			// Speed up ball
			maxPlayers := gs.getMaxPlayersPerSide()
			speedCap := gs.GetSpeedCap(maxPlayers)
			if gs.Ball.Speed() < speedCap {
				gs.Ball.SpeedUp(SpeedIncrement)
			}
			break
		}
	}
}

// getMaxPlayersPerSide returns the larger team size
func (gs *GameState) getMaxPlayersPerSide() int {
	left, right := 0, 0
	for _, p := range gs.Paddles {
		if p.Team == protocol.TeamLeft {
			left++
		} else {
			right++
		}
	}
	if left > right {
		return left
	}
	return right
}

// CheckScore checks if a team scored and handles the pause
func (gs *GameState) CheckScore() {
	if gs.Ball.X < 0 {
		// Right team scores
		gs.RightScore++
		gs.LastScorer = protocol.TeamRight
		gs.startPause()
	} else if gs.Ball.X >= float64(gs.Width) {
		// Left team scores
		gs.LeftScore++
		gs.LastScorer = protocol.TeamLeft
		gs.startPause()
	}
}

// startPause initiates the 2-second pause after a score
func (gs *GameState) startPause() {
	gs.Paused = true
	gs.PauseTicksLeft = 2 * TickRate // 2 seconds
}

// IsGameOver returns true if a team has won
func (gs *GameState) IsGameOver() bool {
	return gs.LeftScore >= gs.PointsToWin || gs.RightScore >= gs.PointsToWin
}

// GetWinner returns the winning team
func (gs *GameState) GetWinner() protocol.Team {
	if gs.LeftScore >= gs.PointsToWin {
		return protocol.TeamLeft
	}
	return protocol.TeamRight
}

// ToProtocolState converts to network-transmittable state
func (gs *GameState) ToProtocolState() protocol.GameState {
	paddles := make([]protocol.PaddleState, len(gs.Paddles))
	for i, p := range gs.Paddles {
		paddles[i] = protocol.PaddleState{
			ID:     p.ID,
			Team:   p.Team,
			Column: p.Column,
			Y:      p.Y,
			Height: p.Height,
			Color:  p.Color,
		}
	}

	return protocol.GameState{
		Tick: gs.Tick,
		Ball: protocol.BallState{
			X:  gs.Ball.X,
			Y:  gs.Ball.Y,
			VX: gs.Ball.VX,
			VY: gs.Ball.VY,
		},
		Paddles:     paddles,
		LeftScore:   gs.LeftScore,
		RightScore:  gs.RightScore,
		CourtWidth:  gs.Width,
		CourtHeight: gs.Height,
		PointsToWin: gs.PointsToWin,
	}
}
```

**Step 4: Run tests**

Run: `go test ./internal/game/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/game/
git commit -m "feat: add game state with scoring and team management"
```

---

## Task 8: UI Package - Screen

**Files:**
- Create: `internal/ui/screen.go`

**Step 1: Add tcell dependency**

Run: `go get github.com/gdamore/tcell/v2`

**Step 2: Write implementation**

```go
package ui

import (
	"github.com/gdamore/tcell/v2"
)

// Screen wraps tcell.Screen providing convenient drawing methods
type Screen struct {
	screen tcell.Screen
}

// PlayerColors defines the colors for up to 8 players
var PlayerColors = []tcell.Color{
	tcell.ColorRed,
	tcell.ColorBlue,
	tcell.ColorGreen,
	tcell.ColorYellow,
	tcell.ColorPurple,
	tcell.ColorOrange,
	tcell.ColorTeal,
	tcell.ColorFuchsia,
}

// NewScreen wraps an existing tcell.Screen
func NewScreen(s tcell.Screen) *Screen {
	return &Screen{screen: s}
}

// InitScreen creates and initializes a real terminal screen
func InitScreen() (*Screen, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := s.Init(); err != nil {
		return nil, err
	}
	return NewScreen(s), nil
}

// Size returns the terminal dimensions
func (s *Screen) Size() (int, int) {
	return s.screen.Size()
}

// Clear clears the screen
func (s *Screen) Clear() {
	s.screen.Clear()
}

// Show updates the terminal with buffered content
func (s *Screen) Show() {
	s.screen.Show()
}

// Fini releases the screen resources
func (s *Screen) Fini() {
	s.screen.Fini()
}

// SetCell sets a single cell on the screen
func (s *Screen) SetCell(x, y int, style tcell.Style, r rune) {
	s.screen.SetContent(x, y, r, nil, style)
}

// DrawText renders a string starting at position x, y
func (s *Screen) DrawText(x, y int, text string, style tcell.Style) {
	for i, r := range text {
		s.screen.SetContent(x+i, y, r, nil, style)
	}
}

// DrawBox draws a Unicode box with the given dimensions
func (s *Screen) DrawBox(x, y, w, h int, style tcell.Style) {
	const (
		topLeft     = '┌'
		topRight    = '┐'
		bottomLeft  = '└'
		bottomRight = '┘'
		horizontal  = '─'
		vertical    = '│'
	)

	s.screen.SetContent(x, y, topLeft, nil, style)
	s.screen.SetContent(x+w-1, y, topRight, nil, style)
	s.screen.SetContent(x, y+h-1, bottomLeft, nil, style)
	s.screen.SetContent(x+w-1, y+h-1, bottomRight, nil, style)

	for i := x + 1; i < x+w-1; i++ {
		s.screen.SetContent(i, y, horizontal, nil, style)
		s.screen.SetContent(i, y+h-1, horizontal, nil, style)
	}

	for j := y + 1; j < y+h-1; j++ {
		s.screen.SetContent(x, j, vertical, nil, style)
		s.screen.SetContent(x+w-1, j, vertical, nil, style)
	}
}

// FillRect fills a rectangle with a rune
func (s *Screen) FillRect(x, y, w, h int, style tcell.Style, r rune) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			s.screen.SetContent(x+dx, y+dy, r, nil, style)
		}
	}
}

// DrawVerticalLine draws a vertical line
func (s *Screen) DrawVerticalLine(x, y1, y2 int, style tcell.Style, r rune) {
	for y := y1; y <= y2; y++ {
		s.screen.SetContent(x, y, r, nil, style)
	}
}

// PollEvent waits for and returns the next event
func (s *Screen) PollEvent() tcell.Event {
	return s.screen.PollEvent()
}

// GetPlayerStyle returns a foreground style for a player color
func GetPlayerStyle(colorIndex int) tcell.Style {
	if colorIndex < 0 || colorIndex >= len(PlayerColors) {
		return tcell.StyleDefault
	}
	return tcell.StyleDefault.Foreground(PlayerColors[colorIndex])
}

// GetPlayerBgStyle returns a background style for a player color
func GetPlayerBgStyle(colorIndex int) tcell.Style {
	if colorIndex < 0 || colorIndex >= len(PlayerColors) {
		return tcell.StyleDefault
	}
	return tcell.StyleDefault.Background(PlayerColors[colorIndex])
}

// GetPlayerColor returns the tcell.Color for a player color index
func GetPlayerColor(colorIndex int) tcell.Color {
	if colorIndex < 0 || colorIndex >= len(PlayerColors) {
		return tcell.ColorWhite
	}
	return PlayerColors[colorIndex]
}
```

**Step 3: Commit**

```bash
git add internal/ui/ go.mod go.sum
git commit -m "feat: add UI screen wrapper with drawing helpers"
```

---

## Task 9: UI Package - Input

**Files:**
- Create: `internal/ui/input.go`
- Create: `internal/ui/input_test.go`

**Step 1: Write the test**

```go
package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/diegok/pixpong/internal/protocol"
)

func TestKeyToDirection(t *testing.T) {
	tests := []struct {
		key  tcell.Key
		rune rune
		want protocol.Direction
	}{
		{tcell.KeyUp, 0, protocol.DirUp},
		{tcell.KeyDown, 0, protocol.DirDown},
		{tcell.KeyRune, 'w', protocol.DirUp},
		{tcell.KeyRune, 'W', protocol.DirUp},
		{tcell.KeyRune, 's', protocol.DirDown},
		{tcell.KeyRune, 'S', protocol.DirDown},
		{tcell.KeyRune, 'x', protocol.DirNone},
	}

	for _, tt := range tests {
		got := KeyToDirection(tt.key, tt.rune)
		if got != tt.want {
			t.Errorf("KeyToDirection(%v, %c) = %v, want %v", tt.key, tt.rune, got, tt.want)
		}
	}
}

func TestIsQuitKey(t *testing.T) {
	if !IsQuitKey(tcell.KeyRune, 'q') {
		t.Error("'q' should be quit key")
	}
	if !IsQuitKey(tcell.KeyRune, 'Q') {
		t.Error("'Q' should be quit key")
	}
	if !IsQuitKey(tcell.KeyEscape, 0) {
		t.Error("Escape should be quit key")
	}
	if !IsQuitKey(tcell.KeyCtrlC, 0) {
		t.Error("Ctrl+C should be quit key")
	}
	if IsQuitKey(tcell.KeyRune, 'x') {
		t.Error("'x' should not be quit key")
	}
}

func TestIsStartKey(t *testing.T) {
	if !IsStartKey(tcell.KeyEnter) {
		t.Error("Enter should be start key")
	}
	if IsStartKey(tcell.KeyRune) {
		t.Error("other keys should not be start key")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/ui/... -run TestKey`
Expected: FAIL

**Step 3: Write implementation**

```go
package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/diegok/pixpong/internal/protocol"
)

// KeyToDirection converts a key event to a movement direction
// For Pong, only up/down movement is allowed
func KeyToDirection(key tcell.Key, r rune) protocol.Direction {
	switch key {
	case tcell.KeyUp:
		return protocol.DirUp
	case tcell.KeyDown:
		return protocol.DirDown
	case tcell.KeyRune:
		switch r {
		case 'w', 'W':
			return protocol.DirUp
		case 's', 'S':
			return protocol.DirDown
		}
	}
	return protocol.DirNone
}

// IsQuitKey returns true if the key should quit the application
func IsQuitKey(key tcell.Key, r rune) bool {
	if key == tcell.KeyEscape || key == tcell.KeyCtrlC {
		return true
	}
	if key == tcell.KeyRune && (r == 'q' || r == 'Q') {
		return true
	}
	return false
}

// IsStartKey returns true if the key should start/confirm
func IsStartKey(key tcell.Key) bool {
	return key == tcell.KeyEnter
}
```

**Step 4: Run tests**

Run: `go test ./internal/ui/...`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/
git commit -m "feat: add UI input handling for paddle movement"
```

---

## Task 10: UI Package - Renderer

**Files:**
- Create: `internal/ui/renderer.go`

**Step 1: Write implementation**

```go
package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/diegok/pixpong/internal/protocol"
)

const (
	BallChar   = '⬤'
	PaddleChar = '█'
)

// Renderer handles drawing game states to the screen
type Renderer struct {
	screen *Screen
}

// NewRenderer creates a new renderer
func NewRenderer(screen *Screen) *Renderer {
	return &Renderer{screen: screen}
}

// RenderLobby draws the lobby waiting room
func (r *Renderer) RenderLobby(state protocol.LobbyState) {
	r.screen.Clear()
	w, h := r.screen.Size()

	titleStyle := tcell.StyleDefault.Bold(true)
	defaultStyle := tcell.StyleDefault
	dimStyle := tcell.StyleDefault.Dim(true)

	// Title
	title := "=== PIXPONG LOBBY ==="
	r.screen.DrawText((w-len(title))/2, 1, title, titleStyle)

	// Draw box around player list
	boxW := 40
	boxH := len(state.Players) + 4
	if boxH < 6 {
		boxH = 6
	}
	boxX := (w - boxW) / 2
	boxY := 3
	r.screen.DrawBox(boxX, boxY, boxW, boxH, defaultStyle)

	// Header
	header := fmt.Sprintf("Players (need 2+ to start):")
	r.screen.DrawText(boxX+2, boxY+1, header, titleStyle)

	// Player list
	for i, p := range state.Players {
		playerStyle := GetPlayerStyle(p.Color)
		line := fmt.Sprintf("%d. %s", i+1, p.Name)
		r.screen.DrawText(boxX+2, boxY+2+i, line, playerStyle)
	}

	// Show server addresses for host
	addrY := boxY + boxH + 1
	if state.IsHost && len(state.ServerAddrs) > 0 {
		r.screen.DrawText(boxX, addrY, "Others can join with:", dimStyle)
		addrY++
		maxAddrs := 3
		for i, addr := range state.ServerAddrs {
			if i >= maxAddrs {
				r.screen.DrawText(boxX+2, addrY, fmt.Sprintf("... and %d more", len(state.ServerAddrs)-maxAddrs), dimStyle)
				addrY++
				break
			}
			r.screen.DrawText(boxX+2, addrY, fmt.Sprintf("pixpong --join %s", addr), dimStyle)
			addrY++
		}
		addrY++
	}

	// Game settings
	settingsY := addrY
	r.screen.DrawText(boxX, settingsY, fmt.Sprintf("Points to win: %d", state.PointsToWin), dimStyle)

	// Instructions
	var instructions string
	if state.IsHost && state.CanStart {
		instructions = "Press ENTER to start | Q to quit"
	} else if state.IsHost {
		instructions = "Waiting for players... (min 2) | Q to quit"
	} else {
		instructions = "Waiting for host to start... | Q to quit"
	}
	r.screen.DrawText((w-len(instructions))/2, h-2, instructions, defaultStyle)

	r.screen.Show()
}

// RenderGame draws the main game state
func (r *Renderer) RenderGame(state protocol.GameState) {
	r.screen.Clear()
	screenW, screenH := r.screen.Size()

	// Draw court background
	courtStyle := tcell.StyleDefault.Background(tcell.ColorBlack)
	r.screen.FillRect(0, 2, state.CourtWidth, state.CourtHeight, courtStyle, ' ')

	// Draw center line (dashed)
	centerX := state.CourtWidth / 2
	lineStyle := tcell.StyleDefault.Foreground(tcell.ColorDarkGray)
	for y := 2; y < state.CourtHeight+2; y += 2 {
		r.screen.SetCell(centerX, y, lineStyle, '│')
	}

	// Draw scoreboard at top center
	r.renderScoreboard(state, screenW)

	// Draw paddles
	for _, p := range state.Paddles {
		style := GetPlayerBgStyle(p.Color)
		topY := int(p.Y - float64(p.Height)/2)
		for dy := 0; dy < p.Height; dy++ {
			y := topY + dy + 2 // +2 for scoreboard offset
			if y >= 2 && y < state.CourtHeight+2 {
				r.screen.SetCell(p.Column, y, style, PaddleChar)
			}
		}
	}

	// Draw ball
	ballStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
	ballX := int(state.Ball.X + 0.5)
	ballY := int(state.Ball.Y + 0.5) + 2 // +2 for scoreboard offset
	if ballY >= 2 && ballY < state.CourtHeight+2 {
		r.screen.SetCell(ballX, ballY, ballStyle, BallChar)
	}

	// Draw status bar
	statusY := screenH - 1
	statusStyle := tcell.StyleDefault.Reverse(true)
	r.screen.FillRect(0, statusY, screenW, 1, statusStyle, ' ')
	statusText := " W/S or Up/Down to move | Q to quit"
	r.screen.DrawText(0, statusY, statusText, statusStyle)

	r.screen.Show()
}

// renderScoreboard draws the stadium-style scoreboard at the top
func (r *Renderer) renderScoreboard(state protocol.GameState, screenW int) {
	// Find team colors
	leftColor := tcell.ColorWhite
	rightColor := tcell.ColorWhite
	for _, p := range state.Paddles {
		if p.Team == protocol.TeamLeft && leftColor == tcell.ColorWhite {
			leftColor = GetPlayerColor(p.Color)
		} else if p.Team == protocol.TeamRight && rightColor == tcell.ColorWhite {
			rightColor = GetPlayerColor(p.Color)
		}
	}

	// Score display
	scoreText := fmt.Sprintf("  %d  -  %d  ", state.LeftScore, state.RightScore)
	scoreX := (screenW - len(scoreText)) / 2

	leftStyle := tcell.StyleDefault.Background(leftColor).Foreground(tcell.ColorWhite).Bold(true)
	rightStyle := tcell.StyleDefault.Background(rightColor).Foreground(tcell.ColorWhite).Bold(true)
	defaultStyle := tcell.StyleDefault.Bold(true)

	// Draw left score with team color
	r.screen.DrawText(scoreX, 0, fmt.Sprintf("  %d  ", state.LeftScore), leftStyle)

	// Draw separator
	sepX := scoreX + 6
	r.screen.DrawText(sepX, 0, " - ", defaultStyle)

	// Draw right score with team color
	r.screen.DrawText(sepX+3, 0, fmt.Sprintf("  %d  ", state.RightScore), rightStyle)

	// Points to win
	targetText := fmt.Sprintf("First to %d wins", state.PointsToWin)
	r.screen.DrawText((screenW-len(targetText))/2, 1, targetText, tcell.StyleDefault.Dim(true))
}

// RenderPause draws the pause screen after scoring
func (r *Renderer) RenderPause(state protocol.PauseState, screenW, screenH int) {
	r.screen.Clear()

	// Big score display
	scoreText := fmt.Sprintf("%d - %d", state.LeftScore, state.RightScore)
	r.screen.DrawText((screenW-len(scoreText))/2, screenH/2-2, scoreText, tcell.StyleDefault.Bold(true))

	// Who scored
	var scorerText string
	if state.LastScorer == protocol.TeamLeft {
		scorerText = "LEFT TEAM SCORES!"
	} else {
		scorerText = "RIGHT TEAM SCORES!"
	}
	r.screen.DrawText((screenW-len(scorerText))/2, screenH/2, scorerText, tcell.StyleDefault.Bold(true))

	// Countdown
	countText := fmt.Sprintf("%d", state.SecondsLeft)
	r.screen.DrawText(screenW/2, screenH/2+2, countText, tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true))

	r.screen.Show()
}

// RenderGameOver draws the game over screen
func (r *Renderer) RenderGameOver(state protocol.GameOverState) {
	r.screen.Clear()
	w, h := r.screen.Size()

	titleStyle := tcell.StyleDefault.Bold(true)
	defaultStyle := tcell.StyleDefault

	// Title
	title := "=== GAME OVER ==="
	r.screen.DrawText((w-len(title))/2, h/2-4, title, titleStyle)

	// Final score
	scoreText := fmt.Sprintf("Final Score: %d - %d", state.LeftScore, state.RightScore)
	r.screen.DrawText((w-len(scoreText))/2, h/2-2, scoreText, defaultStyle)

	// Winner
	var winnerText string
	winnerStyle := tcell.StyleDefault.Bold(true).Foreground(tcell.ColorGreen)
	if state.WinningTeam == protocol.TeamLeft {
		winnerText = "LEFT TEAM WINS!"
	} else {
		winnerText = "RIGHT TEAM WINS!"
	}
	r.screen.DrawText((w-len(winnerText))/2, h/2, winnerText, winnerStyle)

	// Instructions
	instructions := "Press ENTER for rematch | Q to quit"
	r.screen.DrawText((w-len(instructions))/2, h-2, instructions, defaultStyle)

	r.screen.Show()
}

// RenderRematch draws the rematch waiting screen
func (r *Renderer) RenderRematch(state protocol.RematchState) {
	r.screen.Clear()
	w, h := r.screen.Size()

	titleStyle := tcell.StyleDefault.Bold(true)
	defaultStyle := tcell.StyleDefault
	readyStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen)
	waitingStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow)

	// Title
	title := "=== REMATCH ==="
	r.screen.DrawText((w-len(title))/2, 1, title, titleStyle)

	// Player list
	boxW := 40
	boxH := len(state.Players) + 4
	boxX := (w - boxW) / 2
	boxY := 3
	r.screen.DrawBox(boxX, boxY, boxW, boxH, defaultStyle)

	r.screen.DrawText(boxX+2, boxY+1, "Players:", titleStyle)

	for i, p := range state.Players {
		playerStyle := GetPlayerStyle(p.Color)
		var statusStr string
		var statusStyle tcell.Style
		if p.Ready {
			statusStr = " [READY]"
			statusStyle = readyStyle
		} else {
			statusStr = " [WAITING...]"
			statusStyle = waitingStyle
		}
		line := fmt.Sprintf("%d. %s", i+1, p.Name)
		r.screen.DrawText(boxX+2, boxY+2+i, line, playerStyle)
		r.screen.DrawText(boxX+2+len(line), boxY+2+i, statusStr, statusStyle)
	}

	// Instructions
	var instructions string
	if state.IsHost {
		if state.AllReady {
			instructions = "All ready! Press ENTER to start | Q to quit"
		} else {
			instructions = "Waiting for all players... | Q to quit"
		}
	} else {
		instructions = "Press ENTER when ready | Q to quit"
	}
	r.screen.DrawText((w-len(instructions))/2, h-2, instructions, defaultStyle)

	r.screen.Show()
}

// RenderCountdown draws the countdown before game start
func (r *Renderer) RenderCountdown(seconds int) {
	r.screen.Clear()
	w, h := r.screen.Size()

	countStyle := tcell.StyleDefault.Bold(true).Foreground(tcell.ColorYellow)
	countStr := fmt.Sprintf("%d", seconds)
	r.screen.DrawText(w/2, h/2, countStr, countStyle)

	msgStyle := tcell.StyleDefault
	msg := "Get Ready!"
	r.screen.DrawText((w-len(msg))/2, h/2+2, msg, msgStyle)

	r.screen.Show()
}

// RenderConnecting draws the connecting screen
func (r *Renderer) RenderConnecting(addr string) {
	r.screen.Clear()
	w, h := r.screen.Size()

	msg := fmt.Sprintf("Connecting to %s...", addr)
	r.screen.DrawText((w-len(msg))/2, h/2, msg, tcell.StyleDefault)

	r.screen.Show()
}

// RenderError draws an error message
func (r *Renderer) RenderError(err string) {
	r.screen.Clear()
	w, h := r.screen.Size()

	errorStyle := tcell.StyleDefault.Foreground(tcell.ColorRed).Bold(true)

	title := "ERROR"
	r.screen.DrawText((w-len(title))/2, h/2-1, title, errorStyle)
	r.screen.DrawText((w-len(err))/2, h/2+1, err, tcell.StyleDefault)

	instructions := "Press any key to exit"
	r.screen.DrawText((w-len(instructions))/2, h-2, instructions, tcell.StyleDefault)

	r.screen.Show()
}
```

**Step 2: Commit**

```bash
git add internal/ui/
git commit -m "feat: add renderer for game, lobby, and UI screens"
```

---

## Task 11: Server Package - Client

**Files:**
- Create: `internal/server/client.go`

**Step 1: Write implementation**

```go
package server

import (
	"net"
	"sync"

	"github.com/diegok/pixpong/internal/protocol"
)

const sendBufferSize = 64

// Client represents a connected player on the server
type Client struct {
	ID       int
	Name     string
	Width    int
	Height   int
	PlayerID int
	conn     net.Conn
	Codec    *protocol.Codec
	sendCh   chan *protocol.Message
	done     chan struct{}
	mu       sync.Mutex
}

// NewClient creates a new client with the given connection
func NewClient(id int, conn net.Conn) *Client {
	return &Client{
		ID:       id,
		PlayerID: -1,
		conn:     conn,
		Codec:    protocol.NewCodec(conn),
		sendCh:   make(chan *protocol.Message, sendBufferSize),
		done:     make(chan struct{}),
	}
}

// StartWriter starts the goroutine that writes messages to the connection
func (c *Client) StartWriter() {
	go func() {
		for {
			select {
			case <-c.done:
				return
			case msg := <-c.sendCh:
				if err := c.Codec.Encode(msg); err != nil {
					return
				}
			}
		}
	}()
}

// Send queues a message to be sent to the client
func (c *Client) Send(msg *protocol.Message) {
	select {
	case c.sendCh <- msg:
	default:
		// Buffer full, drop message
	}
}

// SendDirect sends a message immediately (for handshake)
func (c *Client) SendDirect(msg *protocol.Message) error {
	return c.Codec.Encode(msg)
}

// Close closes the client connection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case <-c.done:
		return
	default:
		close(c.done)
	}

	if c.conn != nil {
		c.conn.Close()
	}
}
```

**Step 2: Commit**

```bash
git add internal/server/
git commit -m "feat: add server client connection handler"
```

---

## Task 12: Server Package - Server

**Files:**
- Create: `internal/server/server.go`

**Step 1: Write implementation**

```go
package server

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/diegok/pixpong/internal/config"
	"github.com/diegok/pixpong/internal/game"
	"github.com/diegok/pixpong/internal/protocol"
)

const (
	TickRate      = 60
	MinTermWidth  = 40
	MinTermHeight = 20
)

// Server manages the game and client connections
type Server struct {
	cfg          *config.Config
	listener     net.Listener
	mu           sync.RWMutex
	clients      map[int]*Client
	nextID       int
	gameState    *game.GameState
	inLobby      bool
	inRematch    bool
	rematchReady map[int]bool
	minWidth     int
	minHeight    int
	done         chan struct{}
}

// NewServer creates a new server with the specified configuration
func NewServer(cfg *config.Config) *Server {
	return &Server{
		cfg:          cfg,
		clients:      make(map[int]*Client),
		nextID:       1,
		inLobby:      true,
		rematchReady: make(map[int]bool),
		minWidth:     MinTermWidth,
		minHeight:    MinTermHeight,
		done:         make(chan struct{}),
	}
}

// Start begins listening for connections
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener

	go s.acceptLoop()

	return nil
}

// GetServerAddresses returns all IP addresses clients can use to connect
func (s *Server) GetServerAddresses() []string {
	var addresses []string

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return []string{fmt.Sprintf("localhost:%d", s.cfg.Port)}
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

		addresses = append(addresses, fmt.Sprintf("%s:%d", ip.String(), s.cfg.Port))
	}

	addresses = append(addresses, fmt.Sprintf("localhost:%d", s.cfg.Port))

	return addresses
}

// Stop gracefully shuts down the server
func (s *Server) Stop() {
	s.mu.Lock()
	select {
	case <-s.done:
		s.mu.Unlock()
		return
	default:
		close(s.done)
	}
	s.mu.Unlock()

	if s.listener != nil {
		s.listener.Close()
	}

	s.mu.RLock()
	for _, client := range s.clients {
		client.Close()
	}
	s.mu.RUnlock()
}

// acceptLoop accepts incoming connections
func (s *Server) acceptLoop() {
	for {
		select {
		case <-s.done:
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				continue
			}
		}

		go s.handleConnection(conn)
	}
}

// handleConnection processes a new client connection
func (s *Server) handleConnection(conn net.Conn) {
	s.mu.Lock()
	// Reject if game already in progress
	if !s.inLobby && !s.inRematch {
		s.mu.Unlock()
		conn.Close()
		return
	}

	clientID := s.nextID
	s.nextID++
	client := NewClient(clientID, conn)
	s.clients[clientID] = client
	s.mu.Unlock()

	client.StartWriter()

	// Wait for join request
	msg, err := client.Codec.Decode()
	if err != nil {
		s.removeClient(clientID)
		return
	}

	if msg.Type != protocol.MsgJoinRequest {
		s.removeClient(clientID)
		return
	}

	joinReq, ok := msg.Payload.(protocol.JoinRequest)
	if !ok {
		s.removeClient(clientID)
		return
	}

	// Validate terminal size
	if joinReq.TerminalWidth < MinTermWidth || joinReq.TerminalHeight < MinTermHeight {
		response := &protocol.Message{
			Type: protocol.MsgJoinResponse,
			Payload: protocol.JoinResponse{
				Accepted: false,
				Reason:   fmt.Sprintf("terminal too small (min %dx%d)", MinTermWidth, MinTermHeight),
			},
		}
		client.SendDirect(response)
		s.removeClient(clientID)
		return
	}

	client.Name = joinReq.PlayerName
	client.Width = joinReq.TerminalWidth
	client.Height = joinReq.TerminalHeight

	// Update minimum board size
	s.mu.Lock()
	if joinReq.TerminalWidth < s.minWidth {
		s.minWidth = joinReq.TerminalWidth
	}
	if joinReq.TerminalHeight < s.minHeight {
		s.minHeight = joinReq.TerminalHeight
	}
	s.mu.Unlock()

	// Send accept response
	response := &protocol.Message{
		Type: protocol.MsgJoinResponse,
		Payload: protocol.JoinResponse{
			PlayerID: clientID,
			Accepted: true,
		},
	}
	if err := client.SendDirect(response); err != nil {
		s.removeClient(clientID)
		return
	}

	client.PlayerID = clientID

	// Broadcast updated lobby state
	s.BroadcastLobbyState()

	// Handle messages from client
	for {
		select {
		case <-s.done:
			return
		case <-client.done:
			return
		default:
		}

		msg, err := client.Codec.Decode()
		if err != nil {
			s.handleDisconnect(clientID)
			return
		}

		s.handleMessage(client, msg)
	}
}

// handleDisconnect handles a client disconnecting
func (s *Server) handleDisconnect(clientID int) {
	s.mu.Lock()
	inGame := !s.inLobby && !s.inRematch
	s.mu.Unlock()

	// If game is in progress, end it
	if inGame {
		s.endGameDueToDisconnect()
	}

	s.removeClient(clientID)
	s.BroadcastLobbyState()
}

// endGameDueToDisconnect ends the game when a player disconnects
func (s *Server) endGameDueToDisconnect() {
	s.mu.Lock()
	s.gameState = nil
	s.inLobby = true
	s.inRematch = false
	s.mu.Unlock()

	// Notify all clients
	msg := &protocol.Message{
		Type: protocol.MsgGameOver,
		Payload: protocol.GameOverState{
			WinningTeam: protocol.TeamLeft, // Arbitrary
			LeftScore:   0,
			RightScore:  0,
		},
	}
	s.broadcast(msg)
}

// handleMessage processes a message from a client
func (s *Server) handleMessage(client *Client, msg *protocol.Message) {
	switch msg.Type {
	case protocol.MsgPlayerInput:
		s.mu.Lock()
		if s.gameState != nil && !s.inLobby && !s.inRematch {
			input, ok := msg.Payload.(protocol.PlayerInput)
			if ok {
				s.gameState.ProcessInput(client.PlayerID, input.Direction)
			}
		}
		s.mu.Unlock()

	case protocol.MsgRematchReady:
		if s.inRematch {
			s.mu.Lock()
			s.rematchReady[client.ID] = true
			s.mu.Unlock()
			s.BroadcastRematchState()
		}
	}
}

// StartGame begins the game from the lobby
func (s *Server) StartGame() {
	s.mu.Lock()
	if !s.inLobby {
		s.mu.Unlock()
		return
	}

	// Calculate board size
	boardWidth := s.minWidth - 2
	boardHeight := s.minHeight - 4

	s.gameState = game.NewGameState(boardWidth, boardHeight, s.cfg.PointsToWin)

	// Add players
	for _, client := range s.clients {
		if client.Name != "" {
			s.gameState.AddPlayer(client.ID, client.Name)
		}
	}

	// Assign teams and positions
	s.gameState.AssignTeams()

	// Initialize ball
	s.gameState.Ball.Reset(float64(boardWidth)/2, float64(boardHeight)/2, true)

	s.inLobby = false
	s.mu.Unlock()

	// Broadcast game start
	startMsg := &protocol.Message{
		Type:    protocol.MsgStartGame,
		Payload: nil,
	}
	s.broadcast(startMsg)

	// Start game loop
	go s.gameLoop()
}

// StartGameWithCountdown broadcasts countdown then starts game
func (s *Server) StartGameWithCountdown() {
	for i := 3; i >= 1; i-- {
		msg := &protocol.Message{
			Type:    protocol.MsgCountdown,
			Payload: protocol.Countdown{Seconds: i},
		}
		s.broadcast(msg)
		time.Sleep(time.Second)
	}

	s.mu.Lock()
	s.inRematch = false
	s.inLobby = true
	s.mu.Unlock()

	s.StartGame()
}

// gameLoop runs the main game tick loop
func (s *Server) gameLoop() {
	ticker := time.NewTicker(time.Second / TickRate)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			if s.gameState == nil {
				s.mu.Unlock()
				return
			}

			s.gameState.Tick()

			// Check for game over
			if s.gameState.IsGameOver() {
				s.broadcastGameOver()
				s.mu.Unlock()
				return
			}

			// Broadcast game state
			state := s.gameState.ToProtocolState()
			s.mu.Unlock()

			msg := &protocol.Message{
				Type:    protocol.MsgGameState,
				Payload: state,
			}
			s.broadcast(msg)
		}
	}
}

// broadcast sends a message to all connected clients
func (s *Server) broadcast(msg *protocol.Message) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.clients {
		client.Send(msg)
	}
}

// BroadcastLobbyState sends the current lobby state to all clients
func (s *Server) BroadcastLobbyState() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.inLobby {
		return
	}

	players := make([]protocol.LobbyPlayer, 0, len(s.clients))
	for _, client := range s.clients {
		if client.Name != "" {
			players = append(players, protocol.LobbyPlayer{
				ID:    client.ID,
				Name:  client.Name,
				Color: (client.ID - 1) % 8,
			})
		}
	}

	canStart := len(players) >= 2
	serverAddrs := s.GetServerAddresses()

	for _, client := range s.clients {
		isHost := client.ID == 1

		lobbyState := protocol.LobbyState{
			Players:     players,
			IsHost:      isHost,
			CanStart:    canStart,
			PointsToWin: s.cfg.PointsToWin,
		}

		if isHost {
			lobbyState.ServerAddrs = serverAddrs
		}

		msg := &protocol.Message{
			Type:    protocol.MsgLobbyState,
			Payload: lobbyState,
		}
		client.Send(msg)
	}
}

// ResetForRematch enters rematch waiting state
func (s *Server) ResetForRematch() {
	s.mu.Lock()
	s.gameState = nil
	s.inLobby = false
	s.inRematch = true
	s.rematchReady = make(map[int]bool)
	s.mu.Unlock()

	s.BroadcastRematchState()
}

// SetClientRematchReady marks a client as ready for rematch
func (s *Server) SetClientRematchReady(clientID int) {
	s.mu.Lock()
	s.rematchReady[clientID] = true
	s.mu.Unlock()
	s.BroadcastRematchState()
}

// BroadcastRematchState sends the current rematch state to all clients
func (s *Server) BroadcastRematchState() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.inRematch {
		return
	}

	players := make([]protocol.RematchPlayer, 0, len(s.clients))
	allReady := true
	for _, client := range s.clients {
		if client.Name != "" {
			ready := s.rematchReady[client.ID]
			if !ready {
				allReady = false
			}
			players = append(players, protocol.RematchPlayer{
				ID:    client.ID,
				Name:  client.Name,
				Color: (client.ID - 1) % 8,
				Ready: ready,
			})
		}
	}

	for _, client := range s.clients {
		isHost := client.ID == 1
		rematchState := protocol.RematchState{
			Players:  players,
			IsHost:   isHost,
			AllReady: allReady,
		}
		msg := &protocol.Message{
			Type:    protocol.MsgRematchState,
			Payload: rematchState,
		}
		client.Send(msg)
	}
}

// broadcastGameOver sends the game over state to all clients
func (s *Server) broadcastGameOver() {
	if s.gameState == nil {
		return
	}

	gameOver := protocol.GameOverState{
		WinningTeam: s.gameState.GetWinner(),
		LeftScore:   s.gameState.LeftScore,
		RightScore:  s.gameState.RightScore,
	}

	msg := &protocol.Message{
		Type:    protocol.MsgGameOver,
		Payload: gameOver,
	}

	for _, client := range s.clients {
		client.Send(msg)
	}
}

// removeClient removes a client from the server
func (s *Server) removeClient(clientID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	client, ok := s.clients[clientID]
	if !ok {
		return
	}

	client.Close()
	delete(s.clients, clientID)
}
```

**Step 2: Commit**

```bash
git add internal/server/
git commit -m "feat: add server with game loop and client management"
```

---

## Task 13: Client Package

**Files:**
- Create: `internal/client/client.go`

**Step 1: Write implementation**

```go
package client

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/diegok/pixpong/internal/protocol"
)

const (
	channelBufferSize = 16
	connectTimeout    = 5 * time.Second
)

// Client represents a connection to the game server
type Client struct {
	Name         string
	Width        int
	Height       int
	PlayerID     int
	conn         net.Conn
	codec        *protocol.Codec
	mu           sync.Mutex
	connected    bool
	GameState    chan protocol.GameState
	LobbyState   chan protocol.LobbyState
	GameOver     chan protocol.GameOverState
	RematchState chan protocol.RematchState
	Countdown    chan protocol.Countdown
	PauseState   chan protocol.PauseState
	GameStart    chan struct{}
	Error        chan error
	done         chan struct{}
}

// NewClient creates a new client
func NewClient(name string, width, height int) *Client {
	return &Client{
		Name:         name,
		Width:        width,
		Height:       height,
		PlayerID:     -1,
		GameState:    make(chan protocol.GameState, channelBufferSize),
		LobbyState:   make(chan protocol.LobbyState, channelBufferSize),
		GameOver:     make(chan protocol.GameOverState, channelBufferSize),
		RematchState: make(chan protocol.RematchState, channelBufferSize),
		Countdown:    make(chan protocol.Countdown, channelBufferSize),
		PauseState:   make(chan protocol.PauseState, channelBufferSize),
		GameStart:    make(chan struct{}, 1),
		Error:        make(chan error, channelBufferSize),
		done:         make(chan struct{}),
	}
}

// Connect establishes a connection to the server
func (c *Client) Connect(addr string) error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return fmt.Errorf("already connected")
	}
	c.mu.Unlock()

	conn, err := net.DialTimeout("tcp", addr, connectTimeout)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.codec = protocol.NewCodec(conn)
	c.mu.Unlock()

	// Send join request
	joinReq := &protocol.Message{
		Type: protocol.MsgJoinRequest,
		Payload: protocol.JoinRequest{
			PlayerName:     c.Name,
			TerminalWidth:  c.Width,
			TerminalHeight: c.Height,
		},
	}

	if err := c.codec.Encode(joinReq); err != nil {
		conn.Close()
		return fmt.Errorf("failed to send join request: %w", err)
	}

	// Wait for response
	conn.SetReadDeadline(time.Now().Add(connectTimeout))
	msg, err := c.codec.Decode()
	conn.SetReadDeadline(time.Time{})

	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to receive join response: %w", err)
	}

	if msg.Type != protocol.MsgJoinResponse {
		conn.Close()
		return fmt.Errorf("unexpected response type: %v", msg.Type)
	}

	response, ok := msg.Payload.(protocol.JoinResponse)
	if !ok {
		conn.Close()
		return fmt.Errorf("invalid join response payload")
	}

	if !response.Accepted {
		conn.Close()
		return fmt.Errorf("join rejected: %s", response.Reason)
	}

	c.mu.Lock()
	c.PlayerID = response.PlayerID
	c.connected = true
	c.mu.Unlock()

	go c.receiveLoop()

	return nil
}

// SendInput sends a direction input to the server
func (c *Client) SendInput(dir protocol.Direction) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return fmt.Errorf("not connected")
	}

	msg := &protocol.Message{
		Type: protocol.MsgPlayerInput,
		Payload: protocol.PlayerInput{
			Direction: dir,
		},
	}

	return c.codec.Encode(msg)
}

// SendRematchReady signals ready for rematch
func (c *Client) SendRematchReady() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return fmt.Errorf("not connected")
	}

	msg := &protocol.Message{
		Type:    protocol.MsgRematchReady,
		Payload: nil,
	}

	return c.codec.Encode(msg)
}

// Close closes the connection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return
	}

	select {
	case <-c.done:
		return
	default:
		close(c.done)
	}

	c.connected = false
	if c.conn != nil {
		c.conn.Close()
	}
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// receiveLoop reads messages from the server
func (c *Client) receiveLoop() {
	defer c.Close()

	for {
		select {
		case <-c.done:
			return
		default:
		}

		msg, err := c.codec.Decode()
		if err != nil {
			select {
			case <-c.done:
				return
			case c.Error <- fmt.Errorf("receive error: %w", err):
			default:
			}
			return
		}

		c.dispatchMessage(msg)
	}
}

// dispatchMessage routes messages to appropriate channels
func (c *Client) dispatchMessage(msg *protocol.Message) {
	switch msg.Type {
	case protocol.MsgGameState:
		if state, ok := msg.Payload.(protocol.GameState); ok {
			c.sendToChannel(c.GameState, state)
		}

	case protocol.MsgLobbyState:
		if state, ok := msg.Payload.(protocol.LobbyState); ok {
			c.sendToChannel(c.LobbyState, state)
		}

	case protocol.MsgGameOver:
		if state, ok := msg.Payload.(protocol.GameOverState); ok {
			select {
			case c.GameOver <- state:
			default:
			}
		}

	case protocol.MsgStartGame:
		select {
		case c.GameStart <- struct{}{}:
		default:
		}

	case protocol.MsgRematchState:
		if state, ok := msg.Payload.(protocol.RematchState); ok {
			c.sendToChannel(c.RematchState, state)
		}

	case protocol.MsgCountdown:
		if countdown, ok := msg.Payload.(protocol.Countdown); ok {
			c.sendToChannel(c.Countdown, countdown)
		}

	case protocol.MsgPauseState:
		if state, ok := msg.Payload.(protocol.PauseState); ok {
			c.sendToChannel(c.PauseState, state)
		}
	}
}

// sendToChannel sends to a channel, dropping old messages if full
func (c *Client) sendToChannel[T any](ch chan T, value T) {
	select {
	case ch <- value:
	default:
		select {
		case <-ch:
		default:
		}
		select {
		case ch <- value:
		default:
		}
	}
}
```

**Step 2: Commit**

```bash
git add internal/client/
git commit -m "feat: add client with server connection and message handling"
```

---

## Task 14: App Package

**Files:**
- Create: `internal/app/app.go`

**Step 1: Write implementation**

```go
package app

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/diegok/pixpong/internal/client"
	"github.com/diegok/pixpong/internal/config"
	"github.com/diegok/pixpong/internal/protocol"
	"github.com/diegok/pixpong/internal/server"
	"github.com/diegok/pixpong/internal/ui"
)

// App is the main application controller
type App struct {
	cfg      *config.Config
	screen   *ui.Screen
	renderer *ui.Renderer
	client   *client.Client
	server   *server.Server

	// State
	inLobby      bool
	inGame       bool
	gameOver     bool
	inRematch    bool
	inCountdown  bool
	lobbyState   protocol.LobbyState
	gameState    protocol.GameState
	overState    protocol.GameOverState
	rematchState protocol.RematchState
	countdown    int

	quit    chan struct{}
	sigChan chan os.Signal
}

// NewApp creates a new application
func NewApp(cfg *config.Config) *App {
	return &App{
		cfg:  cfg,
		quit: make(chan struct{}),
	}
}

// Run is the main entry point
func (a *App) Run() error {
	screen, err := ui.InitScreen()
	if err != nil {
		return fmt.Errorf("failed to initialize screen: %w", err)
	}
	a.screen = screen
	a.renderer = ui.NewRenderer(screen)

	a.sigChan = make(chan os.Signal, 1)
	signal.Notify(a.sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-a.sigChan
		close(a.quit)
	}()

	w, h := a.screen.Size()

	var runErr error
	if a.cfg.IsServer {
		runErr = a.runServer(w, h)
	} else {
		runErr = a.runClient(w, h)
	}

	a.cleanup()
	return runErr
}

// runServer starts the server and connects as a client
func (a *App) runServer(w, h int) error {
	a.server = server.NewServer(a.cfg)
	if err := a.server.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	addr := fmt.Sprintf("localhost:%d", a.cfg.Port)
	return a.connectAndRun(addr, w, h)
}

// runClient connects to a remote server
func (a *App) runClient(w, h int) error {
	addr := a.cfg.ServerAddr
	if !hasPort(addr) {
		addr = fmt.Sprintf("%s:%d", addr, a.cfg.Port)
	}
	return a.connectAndRun(addr, w, h)
}

// connectAndRun establishes connection and runs main loop
func (a *App) connectAndRun(addr string, w, h int) error {
	a.renderer.RenderConnecting(addr)

	name := a.cfg.PlayerName
	if name == "" {
		name = fmt.Sprintf("Player%d", time.Now().UnixNano()%1000)
	}

	boardHeight := h - 4
	a.client = client.NewClient(name, w, boardHeight)

	if err := a.client.Connect(addr); err != nil {
		a.renderer.RenderError(fmt.Sprintf("Connection failed: %v", err))
		a.screen.PollEvent()
		return fmt.Errorf("failed to connect: %w", err)
	}

	a.inLobby = true

	return a.mainLoop()
}

// mainLoop is the main event loop
func (a *App) mainLoop() error {
	eventChan := make(chan tcell.Event, 16)
	go func() {
		for {
			ev := a.screen.PollEvent()
			if ev == nil {
				return
			}
			select {
			case eventChan <- ev:
			case <-a.quit:
				return
			}
		}
	}()

	ticker := time.NewTicker(16 * time.Millisecond) // ~60fps
	defer ticker.Stop()

	for {
		select {
		case <-a.quit:
			return nil

		case ev := <-eventChan:
			if a.handleEvent(ev) {
				return nil
			}

		case state := <-a.client.LobbyState:
			a.lobbyState = state
			a.render()

		case <-a.client.GameStart:
			a.inLobby = false
			a.inGame = true
			a.gameOver = false
			a.inRematch = false
			a.inCountdown = false
			a.render()

		case state := <-a.client.GameState:
			a.gameState = state
			a.render()

		case state := <-a.client.GameOver:
			a.overState = state
			a.inGame = false
			a.gameOver = true
			a.render()

		case state := <-a.client.RematchState:
			a.rematchState = state
			a.inRematch = true
			a.gameOver = false
			a.inCountdown = false
			a.render()

		case countdown := <-a.client.Countdown:
			a.countdown = countdown.Seconds
			a.inCountdown = true
			a.inRematch = false
			a.render()

		case err := <-a.client.Error:
			a.renderer.RenderError(fmt.Sprintf("Connection error: %v", err))
			a.screen.PollEvent()
			return err

		case <-ticker.C:
			a.render()
		}
	}
}

// handleEvent processes input events
func (a *App) handleEvent(ev tcell.Event) bool {
	switch e := ev.(type) {
	case *tcell.EventKey:
		if ui.IsQuitKey(e.Key(), e.Rune()) {
			return true
		}

		if a.inLobby {
			if ui.IsStartKey(e.Key()) && a.lobbyState.IsHost && a.lobbyState.CanStart {
				if a.server != nil {
					a.server.StartGame()
				}
			}
		} else if a.inGame {
			dir := ui.KeyToDirection(e.Key(), e.Rune())
			if dir != protocol.DirNone {
				a.client.SendInput(dir)
			}
		} else if a.gameOver {
			if ui.IsStartKey(e.Key()) {
				if a.server != nil {
					a.server.ResetForRematch()
					a.server.SetClientRematchReady(a.client.PlayerID)
				} else {
					a.client.SendRematchReady()
				}
			}
		} else if a.inRematch {
			if ui.IsStartKey(e.Key()) {
				if a.server != nil && a.rematchState.AllReady {
					go a.server.StartGameWithCountdown()
				} else if !a.rematchState.IsHost {
					a.client.SendRematchReady()
				}
			}
		}

	case *tcell.EventResize:
		a.screen.Clear()
		a.render()
	}

	return false
}

// render draws the current state
func (a *App) render() {
	if a.inCountdown {
		a.renderer.RenderCountdown(a.countdown)
	} else if a.inRematch {
		a.renderer.RenderRematch(a.rematchState)
	} else if a.inLobby {
		a.renderer.RenderLobby(a.lobbyState)
	} else if a.gameOver {
		a.renderer.RenderGameOver(a.overState)
	} else if a.inGame {
		a.renderer.RenderGame(a.gameState)
	}
}

// cleanup shuts down resources
func (a *App) cleanup() {
	signal.Stop(a.sigChan)

	if a.client != nil {
		a.client.Close()
	}

	if a.server != nil {
		a.server.Stop()
	}

	if a.screen != nil {
		a.screen.Fini()
	}
}

// hasPort checks if address contains a port
func hasPort(addr string) bool {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return true
		}
		if addr[i] == ']' {
			return false
		}
	}
	return false
}
```

**Step 2: Commit**

```bash
git add internal/app/
git commit -m "feat: add app controller with main event loop"
```

---

## Task 15: Main Entry Point

**Files:**
- Modify: `cmd/pixpong/main.go`

**Step 1: Write implementation**

```go
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
```

**Step 2: Verify build**

Run: `go build ./cmd/pixpong`
Expected: Builds successfully

**Step 3: Commit**

```bash
git add cmd/pixpong/
git commit -m "feat: add main entry point with CLI handling"
```

---

## Task 16: Integration Testing

**Step 1: Manual test - Server mode**

Run: `./pixpong --server --name TestHost`
Expected: Shows lobby with IP addresses

**Step 2: Manual test - Client mode (in another terminal)**

Run: `./pixpong --join localhost:5555 --name TestClient`
Expected: Connects and appears in lobby

**Step 3: Manual test - Start game**

Press Enter in host terminal
Expected: Game starts, ball bounces, paddles move with W/S

**Step 4: Document any bugs found**

Create issues or fix immediately

**Step 5: Final commit**

```bash
git add -A
git commit -m "feat: complete pixpong multiplayer pong game

- Server/client architecture with TCP/gob networking
- Team-based gameplay with random team assignment
- Ball physics with angle control and speed escalation
- Paddle size scaling based on team size
- Stadium-style scoreboard
- Lobby, countdown, and rematch systems
- Full terminal UI with tcell"
```

---

## Summary

This plan implements pixpong in 16 tasks:

1. **Tasks 1-4**: Project setup, config, protocol types and codec
2. **Tasks 5-7**: Game logic (ball, paddle, state)
3. **Tasks 8-10**: UI (screen, input, renderer)
4. **Tasks 11-13**: Networking (server client, server, client)
5. **Tasks 14-15**: App controller and main entry
6. **Task 16**: Integration testing

Each task follows TDD where applicable, with bite-sized steps and commits.
