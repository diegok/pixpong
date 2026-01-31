package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/diegok/pixpong/internal/protocol"
)

func TestKeyToDirection(t *testing.T) {
	tests := []struct {
		key  tcell.Key
		rune rune
		want protocol.Direction
	}{
		{tcell.KeyUp, 0, protocol.DirUp},
		{tcell.KeyDown, 0, protocol.DirDown},
		{tcell.KeyRune, 'w', protocol.DirUp},
		{tcell.KeyRune, 'W', protocol.DirUp},
		{tcell.KeyRune, 's', protocol.DirDown},
		{tcell.KeyRune, 'S', protocol.DirDown},
		{tcell.KeyRune, 'x', protocol.DirNone},
	}

	for _, tt := range tests {
		got := KeyToDirection(tt.key, tt.rune)
		if got != tt.want {
			t.Errorf("KeyToDirection(%v, %c) = %v, want %v", tt.key, tt.rune, got, tt.want)
		}
	}
}

func TestIsQuitKey(t *testing.T) {
	if !IsQuitKey(tcell.KeyRune, 'q') {
		t.Error("'q' should be quit key")
	}
	if !IsQuitKey(tcell.KeyRune, 'Q') {
		t.Error("'Q' should be quit key")
	}
	if !IsQuitKey(tcell.KeyEscape, 0) {
		t.Error("Escape should be quit key")
	}
	if !IsQuitKey(tcell.KeyCtrlC, 0) {
		t.Error("Ctrl+C should be quit key")
	}
	if IsQuitKey(tcell.KeyRune, 'x') {
		t.Error("'x' should not be quit key")
	}
}

func TestIsStartKey(t *testing.T) {
	if !IsStartKey(tcell.KeyEnter) {
		t.Error("Enter should be start key")
	}
	if IsStartKey(tcell.KeyRune) {
		t.Error("other keys should not be start key")
	}
}
