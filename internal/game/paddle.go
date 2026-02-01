package game

import "github.com/diegok/pixpong/internal/protocol"

const (
	PaddleSpeed       = 1.2 // Increased for more responsive feel
	MovementTimeout   = 8   // Ticks to keep moving after last input (~133ms at 60Hz)
)

type Paddle struct {
	ID            int
	Team          protocol.Team
	Column        int // X position (fixed)
	Y             float64
	Height        int
	Color         int
	Direction     protocol.Direction
	CourtHeight   int
	MovementTicks int // Countdown for movement timeout
}

func NewPaddle(id int, team protocol.Team, column int, color int) *Paddle {
	return &Paddle{
		ID:        id,
		Team:      team,
		Column:    column,
		Color:     color,
		Direction: protocol.DirNone,
	}
}

func (p *Paddle) SetDirection(dir protocol.Direction) {
	p.Direction = dir
	if dir != protocol.DirNone {
		p.MovementTicks = MovementTimeout // Reset timeout on new input
	}
}

func (p *Paddle) Move() {
	halfHeight := float64(p.Height) / 2

	switch p.Direction {
	case protocol.DirUp:
		p.Y -= PaddleSpeed
		if p.Y < halfHeight {
			p.Y = halfHeight
		}
	case protocol.DirDown:
		p.Y += PaddleSpeed
		maxY := float64(p.CourtHeight) - halfHeight
		if p.Y > maxY {
			p.Y = maxY
		}
	}

	// Decrement movement timeout and stop when it expires
	if p.MovementTicks > 0 {
		p.MovementTicks--
		if p.MovementTicks == 0 {
			p.Direction = protocol.DirNone
		}
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
