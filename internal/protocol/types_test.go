package protocol

import (
	"bytes"
	"encoding/gob"
	"testing"
)

func TestDirection(t *testing.T) {
	tests := []struct {
		name  string
		dir   Direction
		value int
	}{
		{"DirNone is 0", DirNone, 0},
		{"DirUp is 1", DirUp, 1},
		{"DirDown is 2", DirDown, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.dir) != tt.value {
				t.Errorf("expected %s to be %d, got %d", tt.name, tt.value, int(tt.dir))
			}
		})
	}
}

func TestTeam(t *testing.T) {
	tests := []struct {
		name  string
		team  Team
		value int
	}{
		{"TeamLeft is 0", TeamLeft, 0},
		{"TeamRight is 1", TeamRight, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.team) != tt.value {
				t.Errorf("expected %s to be %d, got %d", tt.name, tt.value, int(tt.team))
			}
		})
	}
}

func TestGobRegistration(t *testing.T) {
	// Test that all message types can be encoded and decoded via gob
	testCases := []struct {
		name    string
		message Message
	}{
		{
			name: "PlayerInput",
			message: Message{
				Type:    MsgPlayerInput,
				Payload: PlayerInput{Direction: DirUp},
			},
		},
		{
			name: "JoinRequest",
			message: Message{
				Type: MsgJoinRequest,
				Payload: JoinRequest{
					PlayerName:     "TestPlayer",
					TerminalWidth:  80,
					TerminalHeight: 24,
				},
			},
		},
		{
			name: "JoinResponse",
			message: Message{
				Type: MsgJoinResponse,
				Payload: JoinResponse{
					PlayerID: "player-123",
					Accepted: true,
					Reason:   "",
				},
			},
		},
		{
			name: "BallState",
			message: Message{
				Type: MsgGameState,
				Payload: BallState{
					X:  10.5,
					Y:  20.5,
					VX: 1.0,
					VY: -0.5,
				},
			},
		},
		{
			name: "PaddleState",
			message: Message{
				Type: MsgGameState,
				Payload: PaddleState{
					ID:     "paddle-1",
					Team:   TeamLeft,
					Column: 2,
					Y:      10.0,
					Height: 5,
					Color:  1,
				},
			},
		},
		{
			name: "GameState",
			message: Message{
				Type: MsgGameState,
				Payload: GameState{
					Tick: 100,
					Ball: BallState{X: 40.0, Y: 12.0, VX: 1.0, VY: 0.5},
					Paddles: []PaddleState{
						{ID: "p1", Team: TeamLeft, Column: 2, Y: 10.0, Height: 5, Color: 1},
						{ID: "p2", Team: TeamRight, Column: 77, Y: 10.0, Height: 5, Color: 2},
					},
					LeftScore:   3,
					RightScore:  5,
					CourtWidth:  80,
					CourtHeight: 24,
					PointsToWin: 10,
				},
			},
		},
		{
			name: "LobbyState",
			message: Message{
				Type: MsgLobbyState,
				Payload: LobbyState{
					Players: []LobbyPlayer{
						{ID: "p1", Name: "Alice", Color: 1},
						{ID: "p2", Name: "Bob", Color: 2},
					},
					IsHost:      true,
					CanStart:    true,
					ServerAddrs: []string{"192.168.1.100:5555", "10.0.0.1:5555"},
					PointsToWin: 10,
				},
			},
		},
		{
			name: "GameOverState",
			message: Message{
				Type: MsgGameOver,
				Payload: GameOverState{
					WinningTeam: TeamRight,
					LeftScore:   8,
					RightScore:  10,
				},
			},
		},
		{
			name: "RematchState",
			message: Message{
				Type: MsgRematchState,
				Payload: RematchState{
					Players: []RematchPlayer{
						{ID: "p1", Name: "Alice", Color: 1, Ready: true},
						{ID: "p2", Name: "Bob", Color: 2, Ready: false},
					},
					IsHost:   true,
					AllReady: false,
				},
			},
		},
		{
			name: "Countdown",
			message: Message{
				Type:    MsgCountdown,
				Payload: Countdown{Seconds: 3},
			},
		},
		{
			name: "PauseState",
			message: Message{
				Type: MsgPauseState,
				Payload: PauseState{
					SecondsLeft: 5,
					LeftScore:   2,
					RightScore:  3,
					LastScorer:  TeamRight,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encode
			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)
			if err := enc.Encode(&tc.message); err != nil {
				t.Fatalf("failed to encode %s: %v", tc.name, err)
			}

			// Decode
			var decoded Message
			dec := gob.NewDecoder(&buf)
			if err := dec.Decode(&decoded); err != nil {
				t.Fatalf("failed to decode %s: %v", tc.name, err)
			}

			// Verify type matches
			if decoded.Type != tc.message.Type {
				t.Errorf("expected type %d, got %d", tc.message.Type, decoded.Type)
			}

			// Verify payload is not nil
			if decoded.Payload == nil {
				t.Error("decoded payload is nil")
			}
		})
	}
}

func TestMessageTypes(t *testing.T) {
	// Verify message types are distinct
	types := []MessageType{
		MsgPlayerInput,
		MsgGameState,
		MsgLobbyState,
		MsgJoinRequest,
		MsgJoinResponse,
		MsgStartGame,
		MsgGameOver,
		MsgRematchReady,
		MsgRematchState,
		MsgCountdown,
		MsgPauseState,
	}

	seen := make(map[MessageType]bool)
	for _, mt := range types {
		if seen[mt] {
			t.Errorf("duplicate message type value: %d", mt)
		}
		seen[mt] = true
	}
}
