package protocol

import (
	"encoding/gob"
)

// Direction represents paddle movement direction
type Direction int

const (
	DirNone Direction = 0
	DirUp   Direction = 1
	DirDown Direction = 2
)

// Team represents which side a player is on
type Team int

const (
	TeamLeft  Team = 0
	TeamRight Team = 1
)

// MessageType identifies the type of network message
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
	MsgPauseState
)

// Message is the wrapper for all network messages
type Message struct {
	Type    MessageType
	Payload interface{}
}

// PlayerInput represents a player's input direction
type PlayerInput struct {
	Direction Direction
}

// JoinRequest is sent by a client wanting to join the game
type JoinRequest struct {
	PlayerName     string
	TerminalWidth  int
	TerminalHeight int
}

// JoinResponse is sent by the server in response to a join request
type JoinResponse struct {
	PlayerID string
	Accepted bool
	Reason   string
}

// BallState represents the ball's position and velocity
type BallState struct {
	X  float64
	Y  float64
	VX float64
	VY float64
}

// PaddleState represents a paddle's state
type PaddleState struct {
	ID     string
	Team   Team
	Column int
	Y      float64
	Height int
	Color  int
}

// GameState represents the complete game state
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

// LobbyPlayer represents a player in the lobby
type LobbyPlayer struct {
	ID    string
	Name  string
	Color int
}

// LobbyState represents the lobby state
type LobbyState struct {
	Players     []LobbyPlayer
	IsHost      bool
	CanStart    bool
	ServerAddrs []string
	PointsToWin int
}

// GameOverState represents the end of game state
type GameOverState struct {
	WinningTeam Team
	LeftScore   int
	RightScore  int
}

// RematchPlayer represents a player in the rematch screen
type RematchPlayer struct {
	ID    string
	Name  string
	Color int
	Ready bool
}

// RematchState represents the rematch screen state
type RematchState struct {
	Players  []RematchPlayer
	IsHost   bool
	AllReady bool
}

// Countdown represents the countdown before game starts
type Countdown struct {
	Seconds int
}

// PauseState represents the pause state after a point is scored
type PauseState struct {
	SecondsLeft int
	LeftScore   int
	RightScore  int
	LastScorer  Team
}

func init() {
	// Register all payload types with gob for network serialization
	gob.Register(PlayerInput{})
	gob.Register(JoinRequest{})
	gob.Register(JoinResponse{})
	gob.Register(BallState{})
	gob.Register(PaddleState{})
	gob.Register(GameState{})
	gob.Register(LobbyPlayer{})
	gob.Register(LobbyState{})
	gob.Register(GameOverState{})
	gob.Register(RematchPlayer{})
	gob.Register(RematchState{})
	gob.Register(Countdown{})
	gob.Register(PauseState{})
}
