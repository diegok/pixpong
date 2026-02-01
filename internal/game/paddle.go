package game

import "github.com/diegok/pixpong/internal/protocol"

const (
	PaddleTargetStep = 2.5  // How far target moves per input
	PaddleSmoothSpeed = 0.4 // How fast paddle moves toward target (0-1, higher = faster)
)

type Paddle struct {
	ID          int
	Team        protocol.Team
	Column      int // X position (fixed)
	Y           float64
	TargetY     float64 // Target position for smooth movement
	Height      int
	Color       int
	CourtHeight int
}

func NewPaddle(id int, team protocol.Team, column int, color int) *Paddle {
	return &Paddle{
		ID:     id,
		Team:   team,
		Column: column,
		Color:  color,
	}
}

// ProcessInput adjusts target position based on direction input
func (p *Paddle) ProcessInput(dir protocol.Direction) {
	halfHeight := float64(p.Height) / 2
	minY := halfHeight
	maxY := float64(p.CourtHeight) - halfHeight

	switch dir {
	case protocol.DirUp:
		p.TargetY -= PaddleTargetStep
		if p.TargetY < minY {
			p.TargetY = minY
		}
	case protocol.DirDown:
		p.TargetY += PaddleTargetStep
		if p.TargetY > maxY {
			p.TargetY = maxY
		}
	}
}

// Update smoothly moves paddle toward target position
func (p *Paddle) Update() {
	// Smooth interpolation toward target
	diff := p.TargetY - p.Y
	p.Y += diff * PaddleSmoothSpeed

	// Clamp to bounds
	halfHeight := float64(p.Height) / 2
	if p.Y < halfHeight {
		p.Y = halfHeight
	}
	maxY := float64(p.CourtHeight) - halfHeight
	if p.Y > maxY {
		p.Y = maxY
	}
}

func (p *Paddle) ContainsY(y float64) bool {
	halfHeight := float64(p.Height) / 2
	return y >= p.Y-halfHeight && y <= p.Y+halfHeight
}

func (p *Paddle) TopY() float64 {
	return p.Y - float64(p.Height)/2
}

func (p *Paddle) BottomY() float64 {
	return p.Y + float64(p.Height)/2
}
