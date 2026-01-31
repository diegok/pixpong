package ui

import "github.com/gdamore/tcell/v2"

// PlayerColors defines colors for up to 8 players
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

type Screen struct {
	screen tcell.Screen
}

func NewScreen(s tcell.Screen) *Screen {
	return &Screen{screen: s}
}

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

func (s *Screen) Size() (int, int) {
	return s.screen.Size()
}

func (s *Screen) Clear() {
	s.screen.Clear()
}

func (s *Screen) Show() {
	s.screen.Show()
}

func (s *Screen) Fini() {
	s.screen.Fini()
}

func (s *Screen) SetCell(x, y int, style tcell.Style, r rune) {
	s.screen.SetContent(x, y, r, nil, style)
}

func (s *Screen) DrawText(x, y int, text string, style tcell.Style) {
	for i, r := range text {
		s.screen.SetContent(x+i, y, r, nil, style)
	}
}

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

func (s *Screen) FillRect(x, y, w, h int, style tcell.Style, r rune) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			s.screen.SetContent(x+dx, y+dy, r, nil, style)
		}
	}
}

func (s *Screen) DrawVerticalLine(x, y1, y2 int, style tcell.Style, r rune) {
	for y := y1; y <= y2; y++ {
		s.screen.SetContent(x, y, r, nil, style)
	}
}

func (s *Screen) PollEvent() tcell.Event {
	return s.screen.PollEvent()
}

func GetPlayerStyle(colorIndex int) tcell.Style {
	if colorIndex < 0 || colorIndex >= len(PlayerColors) {
		return tcell.StyleDefault
	}
	return tcell.StyleDefault.Foreground(PlayerColors[colorIndex])
}

func GetPlayerBgStyle(colorIndex int) tcell.Style {
	if colorIndex < 0 || colorIndex >= len(PlayerColors) {
		return tcell.StyleDefault
	}
	return tcell.StyleDefault.Background(PlayerColors[colorIndex])
}

func GetPlayerColor(colorIndex int) tcell.Color {
	if colorIndex < 0 || colorIndex >= len(PlayerColors) {
		return tcell.ColorWhite
	}
	return PlayerColors[colorIndex]
}
