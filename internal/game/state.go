package game

import (
	"fmt"
	"math/rand"

	"github.com/diegok/pixpong/internal/protocol"
)

// Constants for game state management
const (
	TickRate              = 60   // Ticks per second
	BaseSpeedCap          = 1.5  // Base maximum ball speed
	SpeedCapPerPlayer     = 0.3  // Additional speed cap per player
	SpeedIncrement        = 1.05 // 5% speed increase per paddle hit
	BasePaddleHeight      = 5    // Default paddle height
	MinPaddleHeight       = 3    // Minimum paddle height
	PaddleHeightPerPlayer = 1    // Height reduction per additional player
)

// PlayerInfo stores player metadata
type PlayerInfo struct {
	ID   int
	Name string
}

// GameState manages the complete game state
type GameState struct {
	Width          int
	Height         int
	Ball           *Ball
	Paddles        []*Paddle
	Players        []PlayerInfo
	LeftScore      int
	RightScore     int
	PointsToWin    int
	Tick           int
	Paused         bool
	PauseTicksLeft int
	LastScorer     protocol.Team
	WaitingForServe bool
	ServingTeam    protocol.Team
}

// NewGameState creates a new game state with the given dimensions
func NewGameState(width, height, pointsToWin int) *GameState {
	return &GameState{
		Width:       width,
		Height:      height,
		PointsToWin: pointsToWin,
		Ball:        NewBall(float64(width)/2, float64(height)/2),
		Paddles:     make([]*Paddle, 0),
		Players:     make([]PlayerInfo, 0),
	}
}

// AddPlayer adds a new player and creates their paddle
func (gs *GameState) AddPlayer(id int, name string) *Paddle {
	color := (id - 1) % 8 // 8 colors available, wrap around

	// Create paddle with temporary team (will be assigned later)
	paddle := NewPaddle(id, protocol.TeamLeft, 0, color)

	gs.Paddles = append(gs.Paddles, paddle)
	gs.Players = append(gs.Players, PlayerInfo{ID: id, Name: name})

	return paddle
}

// GetPaddle returns the paddle for the given player ID
func (gs *GameState) GetPaddle(id int) *Paddle {
	for _, p := range gs.Paddles {
		if p.ID == id {
			return p
		}
	}
	return nil
}

// AssignTeams randomly assigns players to teams and positions paddles
func (gs *GameState) AssignTeams() {
	if len(gs.Paddles) == 0 {
		return
	}

	// Shuffle paddles randomly
	shuffled := make([]*Paddle, len(gs.Paddles))
	copy(shuffled, gs.Paddles)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	// Split into teams
	halfCount := len(shuffled) / 2
	leftTeam := shuffled[:halfCount]
	rightTeam := shuffled[halfCount:]

	// Calculate paddle height per team - fewer players = bigger paddles
	leftPaddleHeight := gs.CalculatePaddleHeight(len(leftTeam))
	rightPaddleHeight := gs.CalculatePaddleHeight(len(rightTeam))

	// Assign left team
	gs.assignTeamPaddles(leftTeam, protocol.TeamLeft, leftPaddleHeight)

	// Assign right team
	gs.assignTeamPaddles(rightTeam, protocol.TeamRight, rightPaddleHeight)

	// Initialize ball with velocity (launch toward random team)
	launchRight := rand.Intn(2) == 0
	gs.Ball.Reset(float64(gs.Width)/2, float64(gs.Height)/2, launchRight)
}

// assignTeamPaddles sets up paddles for a team
func (gs *GameState) assignTeamPaddles(paddles []*Paddle, team protocol.Team, height int) {
	count := len(paddles)
	if count == 0 {
		return
	}

	// Determine column range for team
	var columnStart, columnEnd int
	if team == protocol.TeamLeft {
		columnStart = 1
		columnEnd = gs.Width / 4
	} else {
		columnStart = gs.Width * 3 / 4
		columnEnd = gs.Width - 1
	}

	// Distribute columns evenly
	columnRange := columnEnd - columnStart
	columnSpacing := columnRange / (count + 1)
	if columnSpacing < 1 {
		columnSpacing = 1
	}

	for i, p := range paddles {
		p.Team = team
		p.Column = columnStart + (i+1)*columnSpacing
		p.Height = height
		p.CourtHeight = gs.Height
		centerY := float64(gs.Height) / 2
		p.Y = centerY
		p.TargetY = centerY // Initialize target to current position
	}
}

// CalculatePaddleHeight computes paddle height based on players per side
func (gs *GameState) CalculatePaddleHeight(playersPerSide int) int {
	height := BasePaddleHeight - (playersPerSide-1)*PaddleHeightPerPlayer
	if height < MinPaddleHeight {
		height = MinPaddleHeight
	}
	return height
}

// GetSpeedCap returns the maximum ball speed based on players per side
func (gs *GameState) GetSpeedCap(playersPerSide int) float64 {
	return BaseSpeedCap + float64(playersPerSide-1)*SpeedCapPerPlayer
}

// ProcessInput handles player input - moves paddle immediately
func (gs *GameState) ProcessInput(playerID int, dir protocol.Direction) {
	paddle := gs.GetPaddle(playerID)
	if paddle != nil {
		paddle.ProcessInput(dir)
	}
}

// Update runs one game tick
func (gs *GameState) Update() {
	gs.Tick++

	// Handle pause state (brief pause after score)
	if gs.Paused {
		gs.PauseTicksLeft--
		if gs.PauseTicksLeft <= 0 {
			gs.Paused = false
			gs.PauseTicksLeft = 0
			// Enter waiting for serve state
			gs.WaitingForServe = true
			gs.ServingTeam = gs.LastScorer
			// Position ball at center
			gs.Ball.X = float64(gs.Width) / 2
			gs.Ball.Y = float64(gs.Height) / 2
			gs.Ball.VX = 0
			gs.Ball.VY = 0
		}
		return
	}

	// Update paddle positions (smooth movement toward targets)
	for _, p := range gs.Paddles {
		p.Update()
	}

	// Handle waiting for serve - ball doesn't move, but paddles can
	if gs.WaitingForServe {
		return
	}

	// Move ball
	gs.Ball.Move()

	// Check wall bounces (top/bottom)
	if gs.Ball.Y <= 0 || gs.Ball.Y >= float64(gs.Height) {
		gs.Ball.BounceVertical()
		// Keep ball in bounds
		if gs.Ball.Y < 0 {
			gs.Ball.Y = 0
		}
		if gs.Ball.Y > float64(gs.Height) {
			gs.Ball.Y = float64(gs.Height)
		}
	}

	// Check paddle collisions
	gs.checkPaddleCollisions()

	// Check scoring
	gs.CheckScore()
}

// checkPaddleCollisions handles ball-paddle collisions
func (gs *GameState) checkPaddleCollisions() {
	for _, p := range gs.Paddles {
		// Check if ball is at paddle column
		ballCol := int(gs.Ball.X)
		if ballCol != p.Column {
			continue
		}

		// Check if ball Y is within paddle
		if !p.ContainsY(gs.Ball.Y) {
			continue
		}

		// Check direction - only collide if ball is moving toward paddle
		if p.Team == protocol.TeamLeft && gs.Ball.VX > 0 {
			continue // Ball moving away from left paddle
		}
		if p.Team == protocol.TeamRight && gs.Ball.VX < 0 {
			continue // Ball moving away from right paddle
		}

		// Bounce off paddle
		gs.Ball.BounceOffPaddle(p.Y, p.Height)

		// Speed up ball
		gs.Ball.SpeedUp(SpeedIncrement)

		// Cap speed based on player count
		playersPerSide := gs.countPlayersOnSide(p.Team)
		speedCap := gs.GetSpeedCap(playersPerSide)
		if gs.Ball.Speed() > speedCap {
			// Scale down to cap
			scale := speedCap / gs.Ball.Speed()
			gs.Ball.VX *= scale
			gs.Ball.VY *= scale
		}

		break // Only one collision per tick
	}
}

// countPlayersOnSide counts players on a team
func (gs *GameState) countPlayersOnSide(team protocol.Team) int {
	count := 0
	for _, p := range gs.Paddles {
		if p.Team == team {
			count++
		}
	}
	return count
}

// CheckScore checks if ball has scored and updates state
func (gs *GameState) CheckScore() {
	// Ball past left edge - right team scores
	if gs.Ball.X < 0 {
		gs.RightScore++
		gs.LastScorer = protocol.TeamRight
		gs.startPause()
	}

	// Ball past right edge - left team scores
	if gs.Ball.X > float64(gs.Width) {
		gs.LeftScore++
		gs.LastScorer = protocol.TeamLeft
		gs.startPause()
	}
}

// startPause begins the post-score pause
func (gs *GameState) startPause() {
	gs.Paused = true
	gs.PauseTicksLeft = TickRate // 1 second pause before serve screen
}

// Serve launches the ball - called when serving team presses Enter
func (gs *GameState) Serve(playerID int) bool {
	if !gs.WaitingForServe {
		return false
	}

	// Check if player is on serving team
	paddle := gs.GetPaddle(playerID)
	if paddle == nil || paddle.Team != gs.ServingTeam {
		return false
	}

	// Launch the ball toward the other team
	launchRight := gs.ServingTeam == protocol.TeamLeft
	gs.Ball.Reset(float64(gs.Width)/2, float64(gs.Height)/2, launchRight)
	gs.WaitingForServe = false
	return true
}

// IsGameOver returns true if either team has won
func (gs *GameState) IsGameOver() bool {
	return gs.LeftScore >= gs.PointsToWin || gs.RightScore >= gs.PointsToWin
}

// GetWinner returns the winning team
func (gs *GameState) GetWinner() protocol.Team {
	if gs.LeftScore >= gs.PointsToWin {
		return protocol.TeamLeft
	}
	return protocol.TeamRight
}

// ToProtocolState converts to network-serializable state
func (gs *GameState) ToProtocolState() protocol.GameState {
	paddles := make([]protocol.PaddleState, len(gs.Paddles))
	for i, p := range gs.Paddles {
		paddles[i] = protocol.PaddleState{
			ID:     fmt.Sprintf("%d", p.ID),
			Team:   p.Team,
			Column: p.Column,
			Y:      p.Y,
			Height: p.Height,
			Color:  p.Color,
		}
	}

	return protocol.GameState{
		Tick:        gs.Tick,
		Ball:        protocol.BallState{X: gs.Ball.X, Y: gs.Ball.Y, VX: gs.Ball.VX, VY: gs.Ball.VY},
		Paddles:     paddles,
		LeftScore:   gs.LeftScore,
		RightScore:  gs.RightScore,
		CourtWidth:  gs.Width,
		CourtHeight: gs.Height,
		PointsToWin: gs.PointsToWin,
	}
}
