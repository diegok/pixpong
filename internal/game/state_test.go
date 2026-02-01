package game

import (
	"testing"

	"github.com/diegok/pixpong/internal/protocol"
)

func TestGameState_AddPlayer(t *testing.T) {
	gs := NewGameState(80, 24, 10)

	// Add first player
	p1 := gs.AddPlayer(1, "Player1")
	if p1 == nil {
		t.Fatal("expected paddle to be created")
	}
	if p1.ID != 1 {
		t.Errorf("expected ID=1, got %d", p1.ID)
	}
	if p1.Color != 0 {
		t.Errorf("expected Color=0 for player 1, got %d", p1.Color)
	}

	// Add second player
	p2 := gs.AddPlayer(2, "Player2")
	if p2 == nil {
		t.Fatal("expected paddle to be created")
	}
	if p2.ID != 2 {
		t.Errorf("expected ID=2, got %d", p2.ID)
	}
	if p2.Color != 1 {
		t.Errorf("expected Color=1 for player 2, got %d", p2.Color)
	}

	// Verify paddles are stored
	if len(gs.Paddles) != 2 {
		t.Errorf("expected 2 paddles, got %d", len(gs.Paddles))
	}

	// Verify GetPaddle works
	got := gs.GetPaddle(1)
	if got != p1 {
		t.Errorf("GetPaddle(1) returned wrong paddle")
	}
	got = gs.GetPaddle(2)
	if got != p2 {
		t.Errorf("GetPaddle(2) returned wrong paddle")
	}
	got = gs.GetPaddle(999)
	if got != nil {
		t.Errorf("GetPaddle(999) should return nil")
	}

	// Test color wrapping (8 colors)
	gs2 := NewGameState(80, 24, 10)
	for i := 1; i <= 10; i++ {
		gs2.AddPlayer(i, "Player")
	}
	p9 := gs2.GetPaddle(9)
	if p9.Color != 0 {
		t.Errorf("expected Color=0 for player 9 (wrap around), got %d", p9.Color)
	}
}

func TestGameState_AssignTeams(t *testing.T) {
	gs := NewGameState(80, 24, 10)

	// Add 4 players
	gs.AddPlayer(1, "P1")
	gs.AddPlayer(2, "P2")
	gs.AddPlayer(3, "P3")
	gs.AddPlayer(4, "P4")

	gs.AssignTeams()

	// Count players per team
	leftCount := 0
	rightCount := 0
	for _, p := range gs.Paddles {
		if p.Team == protocol.TeamLeft {
			leftCount++
		} else {
			rightCount++
		}
	}

	// Should be balanced
	if leftCount != 2 {
		t.Errorf("expected 2 left team players, got %d", leftCount)
	}
	if rightCount != 2 {
		t.Errorf("expected 2 right team players, got %d", rightCount)
	}

	// Verify paddles have valid columns
	for _, p := range gs.Paddles {
		if p.Team == protocol.TeamLeft {
			if p.Column < 1 || p.Column > gs.Width/4 {
				t.Errorf("left paddle column %d out of expected range", p.Column)
			}
		} else {
			if p.Column < gs.Width*3/4 || p.Column >= gs.Width {
				t.Errorf("right paddle column %d out of expected range", p.Column)
			}
		}
	}

	// Verify paddle height is set
	for _, p := range gs.Paddles {
		if p.Height <= 0 {
			t.Errorf("paddle height should be > 0, got %d", p.Height)
		}
	}

	// Verify court height is set
	for _, p := range gs.Paddles {
		if p.CourtHeight != gs.Height {
			t.Errorf("expected CourtHeight=%d, got %d", gs.Height, p.CourtHeight)
		}
	}

	// Verify Y positions are set (centered)
	for _, p := range gs.Paddles {
		if p.Y == 0 {
			t.Errorf("paddle Y should be initialized, got 0")
		}
	}
}

func TestGameState_AssignTeams_OddPlayers(t *testing.T) {
	gs := NewGameState(80, 24, 10)

	// Add 3 players
	gs.AddPlayer(1, "P1")
	gs.AddPlayer(2, "P2")
	gs.AddPlayer(3, "P3")

	gs.AssignTeams()

	// Count players per team
	leftCount := 0
	rightCount := 0
	for _, p := range gs.Paddles {
		if p.Team == protocol.TeamLeft {
			leftCount++
		} else {
			rightCount++
		}
	}

	// Should be as balanced as possible (1 and 2, or 2 and 1)
	if leftCount+rightCount != 3 {
		t.Errorf("expected 3 total players, got %d", leftCount+rightCount)
	}
	diff := leftCount - rightCount
	if diff < -1 || diff > 1 {
		t.Errorf("teams should differ by at most 1: left=%d, right=%d", leftCount, rightCount)
	}
}

func TestGameState_CalculatePaddleHeight(t *testing.T) {
	gs := NewGameState(80, 24, 10)

	tests := []struct {
		playersPerSide int
		expected       int
	}{
		{1, BasePaddleHeight},                       // 8 - 0 = 8
		{2, BasePaddleHeight - PaddleHeightPerPlayer}, // 8 - 1 = 7
		{3, BasePaddleHeight - 2*PaddleHeightPerPlayer}, // 8 - 2 = 6
		{6, MinPaddleHeight},                        // Would be 8-5=3, min is 3
		{10, MinPaddleHeight},                       // Would be negative, clamped to 3
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			height := gs.CalculatePaddleHeight(tt.playersPerSide)
			if height != tt.expected {
				t.Errorf("CalculatePaddleHeight(%d) = %d, want %d",
					tt.playersPerSide, height, tt.expected)
			}
		})
	}
}

func TestGameState_BallScoring(t *testing.T) {
	// Test right team scores (ball goes past left edge)
	t.Run("right team scores", func(t *testing.T) {
		gs := NewGameState(80, 24, 10)
		gs.AddPlayer(1, "P1")
		gs.AddPlayer(2, "P2")
		gs.AssignTeams()
		gs.Ball = NewBall(float64(gs.Width)/2, float64(gs.Height)/2)
		gs.Ball.VX = -0.5
		gs.Ball.VY = 0

		// Move ball past left edge
		gs.Ball.X = -1

		gs.CheckScore()

		if gs.RightScore != 1 {
			t.Errorf("expected RightScore=1, got %d", gs.RightScore)
		}
		if gs.LeftScore != 0 {
			t.Errorf("expected LeftScore=0, got %d", gs.LeftScore)
		}
		if gs.LastScorer != protocol.TeamRight {
			t.Errorf("expected LastScorer=TeamRight")
		}
		if !gs.Paused {
			t.Errorf("expected game to be paused after score")
		}
	})

	// Test left team scores (ball goes past right edge)
	t.Run("left team scores", func(t *testing.T) {
		gs := NewGameState(80, 24, 10)
		gs.AddPlayer(1, "P1")
		gs.AddPlayer(2, "P2")
		gs.AssignTeams()
		gs.Ball = NewBall(float64(gs.Width)/2, float64(gs.Height)/2)
		gs.Ball.VX = 0.5
		gs.Ball.VY = 0

		// Move ball past right edge
		gs.Ball.X = float64(gs.Width) + 1

		gs.CheckScore()

		if gs.LeftScore != 1 {
			t.Errorf("expected LeftScore=1, got %d", gs.LeftScore)
		}
		if gs.RightScore != 0 {
			t.Errorf("expected RightScore=0, got %d", gs.RightScore)
		}
		if gs.LastScorer != protocol.TeamLeft {
			t.Errorf("expected LastScorer=TeamLeft")
		}
	})
}

func TestGameState_WinCondition(t *testing.T) {
	gs := NewGameState(80, 24, 5)

	// No winner initially
	if gs.IsGameOver() {
		t.Errorf("game should not be over initially")
	}
	if gs.GetWinner() != protocol.TeamLeft { // Default, doesn't matter
		// Just checking it doesn't crash
	}

	// Left team wins
	gs.LeftScore = 5
	if !gs.IsGameOver() {
		t.Errorf("game should be over when left score reaches points to win")
	}
	if gs.GetWinner() != protocol.TeamLeft {
		t.Errorf("expected winner to be TeamLeft")
	}

	// Reset and test right team wins
	gs.LeftScore = 0
	gs.RightScore = 5
	if !gs.IsGameOver() {
		t.Errorf("game should be over when right score reaches points to win")
	}
	if gs.GetWinner() != protocol.TeamRight {
		t.Errorf("expected winner to be TeamRight")
	}

	// Not over if below points to win
	gs.RightScore = 4
	if gs.IsGameOver() {
		t.Errorf("game should not be over with score below points to win")
	}
}

func TestGameState_SpeedCap(t *testing.T) {
	gs := NewGameState(80, 24, 10)

	tests := []struct {
		playersPerSide int
		expected       float64
	}{
		{1, BaseSpeedCap},                           // 1.5 + 0 = 1.5
		{2, BaseSpeedCap + SpeedCapPerPlayer},       // 1.5 + 0.3 = 1.8
		{3, BaseSpeedCap + 2*SpeedCapPerPlayer},     // 1.5 + 0.6 = 2.1
		{5, BaseSpeedCap + 4*SpeedCapPerPlayer},     // 1.5 + 1.2 = 2.7
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			cap := gs.GetSpeedCap(tt.playersPerSide)
			if cap != tt.expected {
				t.Errorf("GetSpeedCap(%d) = %f, want %f",
					tt.playersPerSide, cap, tt.expected)
			}
		})
	}
}

func TestGameState_ProcessInput(t *testing.T) {
	gs := NewGameState(80, 24, 10)
	gs.AddPlayer(1, "P1")
	gs.AddPlayer(2, "P2")
	gs.AssignTeams()

	p1 := gs.GetPaddle(1)
	p1InitialY := p1.Y

	// Input for player 1 - should move paddle up immediately
	gs.ProcessInput(1, protocol.DirUp)

	if p1.Y >= p1InitialY {
		t.Errorf("expected paddle Y to decrease after DirUp, was %f, now %f", p1InitialY, p1.Y)
	}

	p2 := gs.GetPaddle(2)
	p2InitialY := p2.Y

	// Input for player 2 - should move paddle down immediately
	gs.ProcessInput(2, protocol.DirDown)

	if p2.Y <= p2InitialY {
		t.Errorf("expected paddle Y to increase after DirDown, was %f, now %f", p2InitialY, p2.Y)
	}

	// Invalid player should not crash
	gs.ProcessInput(999, protocol.DirUp)
}

func TestGameState_Tick(t *testing.T) {
	gs := NewGameState(80, 24, 10)
	gs.AddPlayer(1, "P1")
	gs.AddPlayer(2, "P2")
	gs.AssignTeams()

	// Initialize ball
	gs.Ball = NewBall(float64(gs.Width)/2, float64(gs.Height)/2)
	gs.Ball.VX = 0.5
	gs.Ball.VY = 0.3

	initialTick := gs.Tick
	initialBallX := gs.Ball.X

	// Run a tick
	gs.Update()

	if gs.Tick != initialTick+1 {
		t.Errorf("expected Tick to increment, was %d, now %d", initialTick, gs.Tick)
	}

	if gs.Ball.X == initialBallX {
		t.Errorf("expected ball to move")
	}
}

func TestGameState_WallBounce(t *testing.T) {
	gs := NewGameState(80, 24, 10)
	gs.AddPlayer(1, "P1")
	gs.AddPlayer(2, "P2")
	gs.AssignTeams()

	// Ball moving toward top wall - start at 0.3 so after move (-0.5) it goes to -0.2
	gs.Ball = NewBall(40, 0.3)
	gs.Ball.VX = 0.5
	gs.Ball.VY = -0.5 // Moving up

	// Run tick - should bounce
	gs.Update()

	// VY should be positive (bounced)
	if gs.Ball.VY <= 0 {
		t.Errorf("expected VY > 0 after top wall bounce, got %f", gs.Ball.VY)
	}

	// Ball moving toward bottom wall - start near bottom so after move it exceeds
	gs.Ball = NewBall(40, float64(gs.Height)-0.3)
	gs.Ball.VX = 0.5
	gs.Ball.VY = 0.5 // Moving down

	gs.Update()

	// VY should be negative (bounced)
	if gs.Ball.VY >= 0 {
		t.Errorf("expected VY < 0 after bottom wall bounce, got %f", gs.Ball.VY)
	}
}

func TestGameState_ToProtocolState(t *testing.T) {
	gs := NewGameState(80, 24, 10)
	gs.AddPlayer(1, "P1")
	gs.AddPlayer(2, "P2")
	gs.AssignTeams()
	gs.Ball = NewBall(40, 12)
	gs.Ball.VX = 0.5
	gs.Ball.VY = 0.3
	gs.LeftScore = 3
	gs.RightScore = 2
	gs.Tick = 100

	state := gs.ToProtocolState()

	if state.Tick != 100 {
		t.Errorf("expected Tick=100, got %d", state.Tick)
	}
	if state.LeftScore != 3 {
		t.Errorf("expected LeftScore=3, got %d", state.LeftScore)
	}
	if state.RightScore != 2 {
		t.Errorf("expected RightScore=2, got %d", state.RightScore)
	}
	if state.CourtWidth != 80 {
		t.Errorf("expected CourtWidth=80, got %d", state.CourtWidth)
	}
	if state.CourtHeight != 24 {
		t.Errorf("expected CourtHeight=24, got %d", state.CourtHeight)
	}
	if state.Ball.X != 40 {
		t.Errorf("expected Ball.X=40, got %f", state.Ball.X)
	}
	if len(state.Paddles) != 2 {
		t.Errorf("expected 2 paddles, got %d", len(state.Paddles))
	}
}

func TestNewGameState(t *testing.T) {
	gs := NewGameState(80, 24, 10)

	if gs.Width != 80 {
		t.Errorf("expected Width=80, got %d", gs.Width)
	}
	if gs.Height != 24 {
		t.Errorf("expected Height=24, got %d", gs.Height)
	}
	if gs.PointsToWin != 10 {
		t.Errorf("expected PointsToWin=10, got %d", gs.PointsToWin)
	}
	if gs.Ball == nil {
		t.Errorf("expected Ball to be initialized")
	}
	if gs.LeftScore != 0 {
		t.Errorf("expected LeftScore=0, got %d", gs.LeftScore)
	}
	if gs.RightScore != 0 {
		t.Errorf("expected RightScore=0, got %d", gs.RightScore)
	}
	if gs.Paused {
		t.Errorf("expected game not to be paused initially")
	}
}

func TestGameState_PauseAfterScore(t *testing.T) {
	gs := NewGameState(80, 24, 10)
	gs.AddPlayer(1, "P1")
	gs.AddPlayer(2, "P2")
	gs.AssignTeams()
	gs.Ball = NewBall(float64(gs.Width)/2, float64(gs.Height)/2)

	// Score a point
	gs.Ball.X = -1
	gs.CheckScore()

	if !gs.Paused {
		t.Errorf("expected game to be paused after score")
	}

	expectedPauseTicks := TickRate // 1 second
	if gs.PauseTicksLeft != expectedPauseTicks {
		t.Errorf("expected PauseTicksLeft=%d, got %d", expectedPauseTicks, gs.PauseTicksLeft)
	}

	// Tick should decrement pause ticks
	gs.Update()
	if gs.PauseTicksLeft != expectedPauseTicks-1 {
		t.Errorf("expected PauseTicksLeft to decrement")
	}

	// Fast forward to unpause
	gs.PauseTicksLeft = 1
	gs.Update()

	if gs.Paused {
		t.Errorf("expected game to unpause when PauseTicksLeft reaches 0")
	}
}
