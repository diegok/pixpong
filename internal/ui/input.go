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
