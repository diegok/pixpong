package game

import "github.com/diegok/pixpong/internal/protocol"

// PaddleMoveAmount is how far the paddle moves per input
const PaddleMoveAmount = 1.5

type Paddle struct {
	ID          int
	Team        protocol.Team
	Column      int // X position (fixed)
	Y           float64
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

// MoveUp moves the paddle up by the standard amount
func (p *Paddle) MoveUp() {
	halfHeight := float64(p.Height) / 2
	p.Y -= PaddleMoveAmount
	if p.Y < halfHeight {
		p.Y = halfHeight
	}
}

// MoveDown moves the paddle down by the standard amount
func (p *Paddle) MoveDown() {
	halfHeight := float64(p.Height) / 2
	maxY := float64(p.CourtHeight) - halfHeight
	p.Y += PaddleMoveAmount
	if p.Y > maxY {
		p.Y = maxY
	}
}

// ProcessInput moves the paddle based on direction input
func (p *Paddle) ProcessInput(dir protocol.Direction) {
	switch dir {
	case protocol.DirUp:
		p.MoveUp()
	case protocol.DirDown:
		p.MoveDown()
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
