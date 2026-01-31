package game

import (
	"math"
	"testing"
)

func TestBall_Move(t *testing.T) {
	ball := NewBall(10.0, 20.0)
	ball.VX = 1.0
	ball.VY = -0.5

	ball.Move()

	if ball.X != 11.0 {
		t.Errorf("expected X=11.0, got %f", ball.X)
	}
	if ball.Y != 19.5 {
		t.Errorf("expected Y=19.5, got %f", ball.Y)
	}
}

func TestBall_BounceVertical(t *testing.T) {
	ball := NewBall(10.0, 20.0)
	ball.VX = 0.5
	ball.VY = 0.3

	ball.BounceVertical()

	if ball.VX != 0.5 {
		t.Errorf("expected VX=0.5 (unchanged), got %f", ball.VX)
	}
	if ball.VY != -0.3 {
		t.Errorf("expected VY=-0.3, got %f", ball.VY)
	}
}

func TestBall_BounceOffPaddle(t *testing.T) {
	// Ball hitting center of paddle should bounce straight back
	ball := NewBall(5.0, 10.0)
	ball.VX = 0.5
	ball.VY = 0.0

	paddleY := 10.0
	paddleHeight := 6

	ball.BounceOffPaddle(paddleY, paddleHeight)

	// Ball was moving right, should now move left
	if ball.VX >= 0 {
		t.Errorf("expected VX < 0 after bouncing off paddle, got %f", ball.VX)
	}

	// Center hit should have minimal vertical velocity
	if math.Abs(ball.VY) > 0.01 {
		t.Errorf("expected VY near 0 for center hit, got %f", ball.VY)
	}
}

func TestBall_BounceOffPaddle_Edge(t *testing.T) {
	// Ball hitting edge of paddle should bounce at sharper angle
	ball := NewBall(5.0, 10.0)
	ball.VX = 0.5
	ball.VY = 0.0
	originalSpeed := ball.Speed()

	paddleY := 7.0    // Paddle center is at 7, so ball at 10 is hitting upper edge
	paddleHeight := 6 // Paddle extends from 4 to 10

	ball.BounceOffPaddle(paddleY, paddleHeight)

	// Ball was moving right, should now move left
	if ball.VX >= 0 {
		t.Errorf("expected VX < 0 after bouncing off paddle, got %f", ball.VX)
	}

	// Edge hit should have significant vertical velocity (upward, since hit upper edge)
	if ball.VY <= 0 {
		t.Errorf("expected VY > 0 for upper edge hit, got %f", ball.VY)
	}

	// Speed should be preserved
	newSpeed := ball.Speed()
	if math.Abs(newSpeed-originalSpeed) > 0.001 {
		t.Errorf("expected speed preserved, was %f now %f", originalSpeed, newSpeed)
	}
}

func TestBall_SpeedUp(t *testing.T) {
	ball := NewBall(10.0, 20.0)
	ball.VX = 0.4
	ball.VY = 0.3

	originalSpeed := ball.Speed()
	factor := 1.5

	ball.SpeedUp(factor)

	newSpeed := ball.Speed()
	expectedSpeed := originalSpeed * factor

	if math.Abs(newSpeed-expectedSpeed) > 0.001 {
		t.Errorf("expected speed %f, got %f", expectedSpeed, newSpeed)
	}
}

func TestBall_Reset(t *testing.T) {
	ball := NewBall(100.0, 100.0)
	ball.VX = 10.0
	ball.VY = 10.0

	centerX := 40.0
	centerY := 12.0

	// Test reset launching right
	ball.Reset(centerX, centerY, true)

	if ball.X != centerX {
		t.Errorf("expected X=%f, got %f", centerX, ball.X)
	}
	if ball.Y != centerY {
		t.Errorf("expected Y=%f, got %f", centerY, ball.Y)
	}
	if ball.VX <= 0 {
		t.Errorf("expected VX > 0 when launching right, got %f", ball.VX)
	}

	speed := ball.Speed()
	if math.Abs(speed-InitialBallSpeed) > 0.001 {
		t.Errorf("expected initial speed %f, got %f", InitialBallSpeed, speed)
	}

	// Test reset launching left
	ball.Reset(centerX, centerY, false)

	if ball.VX >= 0 {
		t.Errorf("expected VX < 0 when launching left, got %f", ball.VX)
	}
}

func TestBall_Speed(t *testing.T) {
	ball := NewBall(0, 0)
	ball.VX = 3.0
	ball.VY = 4.0

	speed := ball.Speed()

	// 3-4-5 triangle
	if speed != 5.0 {
		t.Errorf("expected speed=5.0, got %f", speed)
	}
}
