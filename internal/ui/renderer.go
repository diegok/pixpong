package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/diegok/pixpong/internal/protocol"
)

const (
	BallChar   = '\u2B24' // ⬤
	PaddleChar = '\u2588' // █
)

// Renderer handles rendering all game screens
type Renderer struct {
	screen *Screen
}

// NewRenderer creates a new renderer with the given screen
func NewRenderer(screen *Screen) *Renderer {
	return &Renderer{screen: screen}
}

// RenderLobby displays the lobby screen
func (r *Renderer) RenderLobby(state protocol.LobbyState) {
	r.screen.Clear()
	screenW, screenH := r.screen.Size()

	// Title
	title := "=== PIXPONG LOBBY ==="
	titleX := (screenW - len(title)) / 2
	titleStyle := tcell.StyleDefault.Bold(true).Foreground(tcell.ColorWhite)
	r.screen.DrawText(titleX, 2, title, titleStyle)

	// Player list
	playerListY := 5
	playersLabel := "Players:"
	r.screen.DrawText(4, playerListY, playersLabel, tcell.StyleDefault.Foreground(tcell.ColorGray))

	for i, player := range state.Players {
		playerStyle := GetPlayerStyle(player.Color)
		playerText := fmt.Sprintf("  %s", player.Name)
		r.screen.DrawText(4, playerListY+1+i, playerText, playerStyle)
	}

	// Server addresses (for host only)
	if state.IsHost && len(state.ServerAddrs) > 0 {
		addrY := playerListY + len(state.Players) + 3
		addrLabel := "Server addresses:"
		r.screen.DrawText(4, addrY, addrLabel, tcell.StyleDefault.Foreground(tcell.ColorGray))
		for i, addr := range state.ServerAddrs {
			r.screen.DrawText(6, addrY+1+i, addr, tcell.StyleDefault.Foreground(tcell.ColorYellow))
		}
	}

	// Points to win setting
	ptY := screenH - 6
	ptText := fmt.Sprintf("Points to win: %d", state.PointsToWin)
	r.screen.DrawText(4, ptY, ptText, tcell.StyleDefault.Foreground(tcell.ColorTeal))

	// Instructions
	instructY := screenH - 4
	var instructions string
	if state.IsHost {
		if state.CanStart {
			instructions = "Press ENTER to start game"
		} else {
			instructions = "Waiting for more players..."
		}
	} else {
		instructions = "Waiting for host to start..."
	}
	instructStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen)
	r.screen.DrawText(4, instructY, instructions, instructStyle)

	// Quit hint
	quitText := "Press 'q' to quit"
	r.screen.DrawText(4, screenH-2, quitText, tcell.StyleDefault.Foreground(tcell.ColorGray))

	r.screen.Show()
}

// RenderGame displays the game screen
func (r *Renderer) RenderGame(state protocol.GameState) {
	r.screen.Clear()
	screenW, screenH := r.screen.Size()

	// Calculate scale factors to map court coordinates to screen coordinates
	// Court coordinates come from the smallest terminal, screen may be larger
	scaleX := float64(screenW) / float64(state.CourtWidth)
	scaleY := float64(screenH-2) / float64(state.CourtHeight) // -2 for status bars

	// Draw court background (black)
	courtStyle := tcell.StyleDefault.Background(tcell.ColorBlack)
	r.screen.FillRect(0, 1, screenW, screenH-2, courtStyle, ' ')

	// Draw center dashed line
	centerX := screenW / 2
	lineStyle := tcell.StyleDefault.Foreground(tcell.ColorDarkGray)
	for y := 1; y < screenH-1; y += 2 {
		r.screen.SetCell(centerX, y, lineStyle, '|')
	}

	// Draw scoreboard at top center
	r.renderScoreboard(state, screenW)

	// Draw all paddles (scaled to screen size)
	for _, paddle := range state.Paddles {
		paddleStyle := GetPlayerStyle(paddle.Color)
		// Scale paddle position and height
		scaledX := int(float64(paddle.Column) * scaleX)
		scaledY := int(paddle.Y*scaleY) + 1 // +1 for top status bar
		scaledHeight := int(float64(paddle.Height) * scaleY)
		if scaledHeight < 1 {
			scaledHeight = 1
		}

		paddleTop := scaledY - scaledHeight/2
		for dy := 0; dy < scaledHeight; dy++ {
			py := paddleTop + dy
			if py >= 1 && py < screenH-1 {
				r.screen.SetCell(scaledX, py, paddleStyle, PaddleChar)
			}
		}
	}

	// Draw ball (scaled to screen size)
	ballX := int(state.Ball.X*scaleX)
	ballY := int(state.Ball.Y*scaleY) + 1 // +1 for top status bar
	if ballX >= 0 && ballX < screenW && ballY >= 1 && ballY < screenH-1 {
		ballStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
		r.screen.SetCell(ballX, ballY, ballStyle, BallChar)
	}

	// Status bar at bottom
	statusY := screenH - 1
	statusStyle := tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorWhite)
	for x := 0; x < screenW; x++ {
		r.screen.SetCell(x, statusY, statusStyle, ' ')
	}
	statusText := fmt.Sprintf(" Tick: %d | First to %d wins", state.Tick, state.PointsToWin)
	r.screen.DrawText(0, statusY, statusText, statusStyle)

	r.screen.Show()
}

// renderScoreboard draws a stadium-style scoreboard at top center
func (r *Renderer) renderScoreboard(state protocol.GameState, screenW int) {
	// Scoreboard format: [ LEFT  3 - 2  RIGHT ]
	leftScoreStr := fmt.Sprintf("%d", state.LeftScore)
	rightScoreStr := fmt.Sprintf("%d", state.RightScore)
	separator := " - "
	leftLabel := "LEFT"
	rightLabel := "RIGHT"

	scoreboardText := fmt.Sprintf("[ %s %s%s%s %s ]", leftLabel, leftScoreStr, separator, rightScoreStr, rightLabel)
	scoreboardX := (screenW - len(scoreboardText)) / 2

	// Draw scoreboard background
	scoreboardStyle := tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorWhite).Bold(true)

	// Draw opening bracket and left label
	r.screen.DrawText(scoreboardX, 0, "[ ", scoreboardStyle)

	// Left team color
	leftStyle := tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorRed).Bold(true)
	r.screen.DrawText(scoreboardX+2, 0, leftLabel, leftStyle)
	r.screen.DrawText(scoreboardX+2+len(leftLabel), 0, " ", scoreboardStyle)

	// Left score
	leftScoreStyle := tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorWhite).Bold(true)
	r.screen.DrawText(scoreboardX+3+len(leftLabel), 0, leftScoreStr, leftScoreStyle)

	// Separator
	r.screen.DrawText(scoreboardX+3+len(leftLabel)+len(leftScoreStr), 0, separator, scoreboardStyle)

	// Right score
	rightScoreX := scoreboardX + 3 + len(leftLabel) + len(leftScoreStr) + len(separator)
	rightScoreStyle := tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorWhite).Bold(true)
	r.screen.DrawText(rightScoreX, 0, rightScoreStr, rightScoreStyle)
	r.screen.DrawText(rightScoreX+len(rightScoreStr), 0, " ", scoreboardStyle)

	// Right team color
	rightStyle := tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorBlue).Bold(true)
	rightLabelX := rightScoreX + len(rightScoreStr) + 1
	r.screen.DrawText(rightLabelX, 0, rightLabel, rightStyle)

	// Closing bracket
	r.screen.DrawText(rightLabelX+len(rightLabel), 0, " ]", scoreboardStyle)
}

// RenderPause displays the pause/serve screen
func (r *Renderer) RenderPause(state protocol.PauseState) {
	r.screen.Clear()
	screenW, screenH := r.screen.Size()

	// Draw court background (black)
	courtStyle := tcell.StyleDefault.Background(tcell.ColorBlack)
	r.screen.FillRect(0, 1, screenW, screenH-2, courtStyle, ' ')

	// Draw center dashed line
	centerX := screenW / 2
	lineStyle := tcell.StyleDefault.Foreground(tcell.ColorDarkGray)
	for y := 1; y < screenH-1; y += 2 {
		r.screen.SetCell(centerX, y, lineStyle, '|')
	}

	// Draw scoreboard
	scoreText := fmt.Sprintf("[ LEFT %d - %d RIGHT ]", state.LeftScore, state.RightScore)
	scoreX := (screenW - len(scoreText)) / 2
	scoreStyle := tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorWhite).Bold(true)
	r.screen.DrawText(scoreX, 0, scoreText, scoreStyle)

	// Center message box
	boxW := 30
	boxH := 7
	boxX := (screenW - boxW) / 2
	boxY := (screenH - boxH) / 2
	boxStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	r.screen.DrawBox(boxX, boxY, boxW, boxH, boxStyle)

	// Fill box background
	fillStyle := tcell.StyleDefault.Background(tcell.ColorDarkGray)
	for y := boxY + 1; y < boxY+boxH-1; y++ {
		for x := boxX + 1; x < boxX+boxW-1; x++ {
			r.screen.SetCell(x, y, fillStyle, ' ')
		}
	}

	if state.WaitingForServe {
		// Show which team should serve
		var teamName string
		var teamStyle tcell.Style
		if state.ServingTeam == protocol.TeamLeft {
			teamName = "LEFT TEAM"
			teamStyle = tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorRed).Bold(true)
		} else {
			teamName = "RIGHT TEAM"
			teamStyle = tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorBlue).Bold(true)
		}

		serveText := fmt.Sprintf("%s SERVE", teamName)
		serveX := (screenW - len(serveText)) / 2
		r.screen.DrawText(serveX, boxY+2, serveText, teamStyle)

		instructText := "Press ENTER to serve"
		instructX := (screenW - len(instructText)) / 2
		instructStyle := tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorGreen)
		r.screen.DrawText(instructX, boxY+4, instructText, instructStyle)
	} else {
		// Brief pause after score - show who scored
		var scorerName string
		var scorerStyle tcell.Style
		if state.LastScorer == protocol.TeamLeft {
			scorerName = "LEFT TEAM"
			scorerStyle = tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorRed).Bold(true)
		} else {
			scorerName = "RIGHT TEAM"
			scorerStyle = tcell.StyleDefault.Background(tcell.ColorDarkGray).Foreground(tcell.ColorBlue).Bold(true)
		}

		scoreMsg := fmt.Sprintf("%s SCORES!", scorerName)
		scoreMsgX := (screenW - len(scoreMsg)) / 2
		r.screen.DrawText(scoreMsgX, boxY+3, scoreMsg, scorerStyle)
	}

	r.screen.Show()
}

// RenderGameOver displays the game over screen
func (r *Renderer) RenderGameOver(state protocol.GameOverState) {
	r.screen.Clear()
	screenW, screenH := r.screen.Size()

	// Title
	title := "=== GAME OVER ==="
	titleX := (screenW - len(title)) / 2
	titleStyle := tcell.StyleDefault.Bold(true).Foreground(tcell.ColorYellow)
	r.screen.DrawText(titleX, screenH/2-4, title, titleStyle)

	// Final score
	scoreText := fmt.Sprintf("Final Score: %d - %d", state.LeftScore, state.RightScore)
	scoreX := (screenW - len(scoreText)) / 2
	r.screen.DrawText(scoreX, screenH/2-1, scoreText, tcell.StyleDefault.Foreground(tcell.ColorWhite))

	// Winner announcement
	var winner string
	var winnerStyle tcell.Style
	if state.WinningTeam == protocol.TeamLeft {
		winner = "LEFT TEAM WINS!"
		winnerStyle = tcell.StyleDefault.Foreground(tcell.ColorRed).Bold(true)
	} else {
		winner = "RIGHT TEAM WINS!"
		winnerStyle = tcell.StyleDefault.Foreground(tcell.ColorBlue).Bold(true)
	}
	winnerX := (screenW - len(winner)) / 2
	r.screen.DrawText(winnerX, screenH/2+1, winner, winnerStyle)

	// Instructions
	rematchText := "Press ENTER for rematch | Press 'q' to quit"
	rematchX := (screenW - len(rematchText)) / 2
	r.screen.DrawText(rematchX, screenH/2+4, rematchText, tcell.StyleDefault.Foreground(tcell.ColorGreen))

	r.screen.Show()
}

// RenderRematch displays the rematch screen
func (r *Renderer) RenderRematch(state protocol.RematchState) {
	r.screen.Clear()
	screenW, screenH := r.screen.Size()

	// Title
	title := "=== REMATCH ==="
	titleX := (screenW - len(title)) / 2
	titleStyle := tcell.StyleDefault.Bold(true).Foreground(tcell.ColorTeal)
	r.screen.DrawText(titleX, 3, title, titleStyle)

	// Player list with ready status
	listY := 6
	headerText := "Player Status:"
	r.screen.DrawText(4, listY, headerText, tcell.StyleDefault.Foreground(tcell.ColorGray))

	for i, player := range state.Players {
		var statusIcon string
		var statusStyle tcell.Style
		if player.Ready {
			statusIcon = "[READY]"
			statusStyle = tcell.StyleDefault.Foreground(tcell.ColorGreen)
		} else {
			statusIcon = "[...]"
			statusStyle = tcell.StyleDefault.Foreground(tcell.ColorGray)
		}

		playerStyle := GetPlayerStyle(player.Color)
		playerText := fmt.Sprintf("  %s ", player.Name)
		y := listY + 2 + i
		r.screen.DrawText(4, y, playerText, playerStyle)
		r.screen.DrawText(4+len(playerText), y, statusIcon, statusStyle)
	}

	// Instructions
	instructY := screenH - 4
	var instructions string
	if state.IsHost {
		if state.AllReady {
			instructions = "All players ready! Press ENTER to start"
		} else {
			instructions = "Waiting for all players to be ready..."
		}
	} else {
		instructions = "Press ENTER to mark yourself ready"
	}
	instructStyle := tcell.StyleDefault.Foreground(tcell.ColorGreen)
	r.screen.DrawText(4, instructY, instructions, instructStyle)

	// Quit hint
	quitText := "Press 'q' to quit"
	r.screen.DrawText(4, screenH-2, quitText, tcell.StyleDefault.Foreground(tcell.ColorGray))

	r.screen.Show()
}

// RenderCountdown displays a large countdown number
func (r *Renderer) RenderCountdown(seconds int) {
	r.screen.Clear()
	screenW, screenH := r.screen.Size()

	// Large number display using ASCII art style
	numberStr := fmt.Sprintf("%d", seconds)
	numberStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Bold(true)

	// Center the number
	centerX := screenW / 2
	centerY := screenH / 2

	// Draw a box around the countdown
	boxW := 10
	boxH := 5
	boxX := centerX - boxW/2
	boxY := centerY - boxH/2
	boxStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	r.screen.DrawBox(boxX, boxY, boxW, boxH, boxStyle)

	// Draw the countdown number in the center of the box
	numX := centerX - len(numberStr)/2
	r.screen.DrawText(numX, centerY, numberStr, numberStyle)

	// "GET READY!" text
	readyText := "GET READY!"
	readyX := (screenW - len(readyText)) / 2
	r.screen.DrawText(readyX, boxY-2, readyText, tcell.StyleDefault.Foreground(tcell.ColorGreen).Bold(true))

	r.screen.Show()
}

// RenderConnecting displays the connecting screen
func (r *Renderer) RenderConnecting(addr string) {
	r.screen.Clear()
	screenW, screenH := r.screen.Size()

	// Title
	title := "PIXPONG"
	titleX := (screenW - len(title)) / 2
	titleStyle := tcell.StyleDefault.Bold(true).Foreground(tcell.ColorTeal)
	r.screen.DrawText(titleX, screenH/2-3, title, titleStyle)

	// Connecting message
	connectText := fmt.Sprintf("Connecting to %s...", addr)
	connectX := (screenW - len(connectText)) / 2
	r.screen.DrawText(connectX, screenH/2, connectText, tcell.StyleDefault.Foreground(tcell.ColorYellow))

	// Hint
	hintText := "Press 'q' to cancel"
	hintX := (screenW - len(hintText)) / 2
	r.screen.DrawText(hintX, screenH/2+3, hintText, tcell.StyleDefault.Foreground(tcell.ColorGray))

	r.screen.Show()
}

// RenderError displays an error screen
func (r *Renderer) RenderError(err string) {
	r.screen.Clear()
	screenW, screenH := r.screen.Size()

	// Error title
	title := "ERROR"
	titleX := (screenW - len(title)) / 2
	titleStyle := tcell.StyleDefault.Bold(true).Foreground(tcell.ColorRed)
	r.screen.DrawText(titleX, screenH/2-2, title, titleStyle)

	// Error message
	// Truncate if too long
	maxErrLen := screenW - 4
	errMsg := err
	if len(errMsg) > maxErrLen {
		errMsg = errMsg[:maxErrLen-3] + "..."
	}
	errX := (screenW - len(errMsg)) / 2
	r.screen.DrawText(errX, screenH/2, errMsg, tcell.StyleDefault.Foreground(tcell.ColorWhite))

	// Instructions
	hintText := "Press any key to continue"
	hintX := (screenW - len(hintText)) / 2
	r.screen.DrawText(hintX, screenH/2+3, hintText, tcell.StyleDefault.Foreground(tcell.ColorGray))

	r.screen.Show()
}
