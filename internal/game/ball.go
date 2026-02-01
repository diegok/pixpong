package game

import (
	"math"
	"math/rand"
)

const (
	InitialBallSpeed = 0.25 // Slower start for better gameplay
	MaxBounceAngle   = math.Pi / 3 // 60 degrees max
)

type Ball struct {
	X, Y   float64
	VX, VY float64
}

func NewBall(x, y float64) *Ball {
	return &Ball{X: x, Y: y}
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

// BounceOffPaddle bounces ball off paddle, angle based on hit position
// paddleY is center Y, paddleHeight is total height
func (b *Ball) BounceOffPaddle(paddleY float64, paddleHeight int) {
	// Calculate where on paddle ball hit (-1 to 1, 0 = center)
	relativeHit := (b.Y - paddleY) / (float64(paddleHeight) / 2)
	if relativeHit < -1 {
		relativeHit = -1
	}
	if relativeHit > 1 {
		relativeHit = 1
	}

	// Bounce angle based on hit position
	bounceAngle := relativeHit * MaxBounceAngle
	speed := math.Sqrt(b.VX*b.VX + b.VY*b.VY)

	// Reverse horizontal direction
	if b.VX > 0 {
		b.VX = -speed * math.Cos(bounceAngle)
	} else {
		b.VX = speed * math.Cos(bounceAngle)
	}
	b.VY = speed * math.Sin(bounceAngle)
}

// SpeedUp multiplies ball speed by factor
func (b *Ball) SpeedUp(factor float64) {
	b.VX *= factor
	b.VY *= factor
}

// Speed returns current speed
func (b *Ball) Speed() float64 {
	return math.Sqrt(b.VX*b.VX + b.VY*b.VY)
}

// Reset places ball at center and launches in specified direction
func (b *Ball) Reset(centerX, centerY float64, launchRight bool) {
	b.X = centerX
	b.Y = centerY

	angle := (rand.Float64() - 0.5) * math.Pi / 3
	speed := InitialBallSpeed
	if launchRight {
		b.VX = speed * math.Cos(angle)
	} else {
		b.VX = -speed * math.Cos(angle)
	}
	b.VY = speed * math.Sin(angle)
}
