package game

import (
	"testing"

	"github.com/diegok/pixpong/internal/protocol"
)

func TestPaddle_MoveUp(t *testing.T) {
	paddle := NewPaddle(1, protocol.TeamLeft, 2, 1)
	paddle.Height = 6
	paddle.CourtHeight = 24
	paddle.Y = 12.0 // Center of court

	paddle.SetDirection(protocol.DirUp)
	initialY := paddle.Y

	paddle.Move()

	if paddle.Y >= initialY {
		t.Errorf("expected Y to decrease when moving up, was %f, now %f", initialY, paddle.Y)
	}
	expectedY := initialY - PaddleSpeed
	if paddle.Y != expectedY {
		t.Errorf("expected Y=%f, got %f", expectedY, paddle.Y)
	}
}

func TestPaddle_MoveDown(t *testing.T) {
	paddle := NewPaddle(1, protocol.TeamLeft, 2, 1)
	paddle.Height = 6
	paddle.CourtHeight = 24
	paddle.Y = 12.0 // Center of court

	paddle.SetDirection(protocol.DirDown)
	initialY := paddle.Y

	paddle.Move()

	if paddle.Y <= initialY {
		t.Errorf("expected Y to increase when moving down, was %f, now %f", initialY, paddle.Y)
	}
	expectedY := initialY + PaddleSpeed
	if paddle.Y != expectedY {
		t.Errorf("expected Y=%f, got %f", expectedY, paddle.Y)
	}
}

func TestPaddle_StaysInBounds_Top(t *testing.T) {
	paddle := NewPaddle(1, protocol.TeamLeft, 2, 1)
	paddle.Height = 6
	paddle.CourtHeight = 24
	paddle.Y = 3.5 // Near top, half height is 3.0

	paddle.SetDirection(protocol.DirUp)

	// Move multiple times to try to go out of bounds
	for i := 0; i < 10; i++ {
		paddle.Move()
	}

	halfHeight := float64(paddle.Height) / 2
	if paddle.Y < halfHeight {
		t.Errorf("paddle went above top boundary: Y=%f, halfHeight=%f", paddle.Y, halfHeight)
	}
	if paddle.TopY() < 0 {
		t.Errorf("paddle top went above 0: TopY=%f", paddle.TopY())
	}
}

func TestPaddle_StaysInBounds_Bottom(t *testing.T) {
	paddle := NewPaddle(1, protocol.TeamLeft, 2, 1)
	paddle.Height = 6
	paddle.CourtHeight = 24
	paddle.Y = 20.5 // Near bottom

	paddle.SetDirection(protocol.DirDown)

	// Move multiple times to try to go out of bounds
	for i := 0; i < 10; i++ {
		paddle.Move()
	}

	halfHeight := float64(paddle.Height) / 2
	maxY := float64(paddle.CourtHeight) - halfHeight
	if paddle.Y > maxY {
		t.Errorf("paddle went below bottom boundary: Y=%f, maxY=%f", paddle.Y, maxY)
	}
	if paddle.BottomY() > float64(paddle.CourtHeight) {
		t.Errorf("paddle bottom went below court: BottomY=%f, CourtHeight=%d", paddle.BottomY(), paddle.CourtHeight)
	}
}

func TestPaddle_ContainsY(t *testing.T) {
	paddle := NewPaddle(1, protocol.TeamLeft, 2, 1)
	paddle.Height = 6
	paddle.Y = 12.0 // Center Y, paddle extends from 9 to 15

	tests := []struct {
		name     string
		y        float64
		expected bool
	}{
		{"center", 12.0, true},
		{"top edge", 9.0, true},
		{"bottom edge", 15.0, true},
		{"inside top", 10.0, true},
		{"inside bottom", 14.0, true},
		{"above paddle", 8.0, false},
		{"below paddle", 16.0, false},
		{"way above", 0.0, false},
		{"way below", 24.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := paddle.ContainsY(tt.y)
			if result != tt.expected {
				t.Errorf("ContainsY(%f) = %v, want %v", tt.y, result, tt.expected)
			}
		})
	}
}

func TestPaddle_TopY(t *testing.T) {
	paddle := NewPaddle(1, protocol.TeamLeft, 2, 1)
	paddle.Height = 6
	paddle.Y = 12.0

	topY := paddle.TopY()
	expectedTop := 9.0 // 12 - 6/2

	if topY != expectedTop {
		t.Errorf("expected TopY=%f, got %f", expectedTop, topY)
	}
}

func TestPaddle_BottomY(t *testing.T) {
	paddle := NewPaddle(1, protocol.TeamLeft, 2, 1)
	paddle.Height = 6
	paddle.Y = 12.0

	bottomY := paddle.BottomY()
	expectedBottom := 15.0 // 12 + 6/2

	if bottomY != expectedBottom {
		t.Errorf("expected BottomY=%f, got %f", expectedBottom, bottomY)
	}
}

func TestNewPaddle(t *testing.T) {
	paddle := NewPaddle(1, protocol.TeamRight, 78, 5)

	if paddle.ID != 1 {
		t.Errorf("expected ID=1, got %d", paddle.ID)
	}
	if paddle.Team != protocol.TeamRight {
		t.Errorf("expected Team=TeamRight, got %v", paddle.Team)
	}
	if paddle.Column != 78 {
		t.Errorf("expected Column=78, got %d", paddle.Column)
	}
	if paddle.Color != 5 {
		t.Errorf("expected Color=5, got %d", paddle.Color)
	}
	if paddle.Direction != protocol.DirNone {
		t.Errorf("expected Direction=DirNone, got %v", paddle.Direction)
	}
}

func TestPaddle_MoveNone(t *testing.T) {
	paddle := NewPaddle(1, protocol.TeamLeft, 2, 1)
	paddle.Height = 6
	paddle.CourtHeight = 24
	paddle.Y = 12.0

	paddle.SetDirection(protocol.DirNone)
	initialY := paddle.Y

	paddle.Move()

	if paddle.Y != initialY {
		t.Errorf("expected Y to remain unchanged with DirNone, was %f, now %f", initialY, paddle.Y)
	}
}
