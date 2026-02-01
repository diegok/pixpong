package app

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/diegok/pixpong/internal/audio"
	"github.com/diegok/pixpong/internal/client"
	"github.com/diegok/pixpong/internal/config"
	"github.com/diegok/pixpong/internal/protocol"
	"github.com/diegok/pixpong/internal/server"
	"github.com/diegok/pixpong/internal/ui"
)

// App is the main application controller that manages the game lifecycle.
type App struct {
	cfg      *config.Config
	screen   *ui.Screen
	renderer *ui.Renderer
	client   *client.Client
	server   *server.Server

	// State
	inLobby         bool
	inGame          bool
	gameOver        bool
	inRematch       bool
	inCountdown     bool
	inPause         bool
	waitingForServe bool
	servingTeam     protocol.Team
	lobbyState      protocol.LobbyState
	gameState       protocol.GameState
	prevGameState   protocol.GameState // For sound detection
	pauseState      protocol.PauseState
	overState       protocol.GameOverState
	rematchState    protocol.RematchState
	countdown       int

	quit    chan struct{}
	sigChan chan os.Signal
}

// NewApp creates a new App instance with the given configuration.
func NewApp(cfg *config.Config) *App {
	return &App{
		cfg:  cfg,
		quit: make(chan struct{}),
	}
}

// Run is the main entry point for the application.
// It initializes the screen, sets up signal handling, and starts the game.
func (a *App) Run() error {
	// Initialize audio (ignore errors - game works without sound)
	_ = audio.Init()

	// Initialize screen
	screen, err := ui.InitScreen()
	if err != nil {
		return fmt.Errorf("failed to initialize screen: %w", err)
	}
	a.screen = screen
	a.renderer = ui.NewRenderer(screen)

	// Setup signal handling
	a.sigChan = make(chan os.Signal, 1)
	signal.Notify(a.sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-a.sigChan
		close(a.quit)
	}()

	// Get terminal size
	w, h := a.screen.Size()

	// Run as server or client
	var runErr error
	if a.cfg.IsServer {
		runErr = a.runServer(w, h)
	} else {
		runErr = a.runClient(w, h)
	}

	// Cleanup
	a.cleanup()

	return runErr
}

// runServer creates and starts a server, then connects to it as a client.
func (a *App) runServer(w, h int) error {
	// Create and start server
	a.server = server.NewServer(a.cfg)
	if err := a.server.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Connect to own server
	addr := fmt.Sprintf("localhost:%d", a.cfg.Port)
	return a.connectAndRun(addr, w, h)
}

// runClient connects to a remote server.
func (a *App) runClient(w, h int) error {
	addr := a.cfg.ServerAddr
	// Add default port if not specified
	if !a.hasPort(addr) {
		addr = fmt.Sprintf("%s:%d", addr, config.DefaultPort)
	}
	return a.connectAndRun(addr, w, h)
}

// connectAndRun establishes a connection to the server and runs the main loop.
func (a *App) connectAndRun(addr string, w, h int) error {
	// Show connecting screen
	a.renderer.RenderConnecting(addr)

	// Generate random name if not provided
	name := a.cfg.PlayerName
	if name == "" {
		name = a.generateRandomName()
	}

	// Create and connect client
	a.client = client.NewClient(name, w, h)
	if err := a.client.Connect(addr); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Enter lobby state
	a.inLobby = true

	// Run main loop
	return a.mainLoop()
}

// mainLoop is the main event loop that handles all input and state updates.
func (a *App) mainLoop() error {
	// Create event channel for screen events
	events := make(chan tcell.Event)
	go func() {
		for {
			ev := a.screen.PollEvent()
			if ev == nil {
				return
			}
			select {
			case events <- ev:
			case <-a.quit:
				return
			}
		}
	}()

	// Ticker for rendering at ~60fps
	ticker := time.NewTicker(16 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-a.quit:
			return nil

		case ev := <-events:
			if a.handleEvent(ev) {
				return nil
			}

		case state := <-a.client.LobbyState:
			a.lobbyState = state
			a.inLobby = true
			a.inGame = false
			a.gameOver = false
			a.inRematch = false
			a.inCountdown = false

		case <-a.client.GameStart:
			a.inLobby = false
			a.inGame = true
			a.gameOver = false
			a.inRematch = false
			a.inCountdown = false

		case state := <-a.client.GameState:
			// Detect sound events by comparing with previous state
			a.detectSoundEvents(state)
			a.prevGameState = state
			a.gameState = state
			a.inGame = true
			a.inPause = false
			a.waitingForServe = false
			a.inLobby = false
			a.inCountdown = false

		case state := <-a.client.GameOver:
			a.overState = state
			a.gameOver = true
			a.inGame = false

		case state := <-a.client.RematchState:
			a.rematchState = state
			a.inRematch = true
			a.gameOver = false
			a.inGame = false
			a.inLobby = false

		case countdown := <-a.client.Countdown:
			a.countdown = countdown.Seconds
			a.inCountdown = true
			a.inLobby = false
			a.inGame = false

		case state := <-a.client.PauseState:
			a.pauseState = state
			a.inPause = true
			a.waitingForServe = state.WaitingForServe
			a.servingTeam = state.ServingTeam
			a.inGame = true
			a.inLobby = false
			a.inCountdown = false

		case err := <-a.client.Error:
			a.renderer.RenderError(err.Error())
			// Wait for a key press
			a.screen.PollEvent()
			return err

		case <-ticker.C:
			a.render()
		}
	}
}

// handleEvent processes keyboard and other events.
// Returns true if the application should quit.
func (a *App) handleEvent(ev tcell.Event) bool {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		// Quit keys always work
		if ev.Key() == tcell.KeyEscape || ev.Rune() == 'q' || ev.Rune() == 'Q' {
			return true
		}

		// Handle context-specific keys
		if a.inLobby {
			return a.handleLobbyEvent(ev)
		} else if a.inGame && !a.inCountdown {
			return a.handleGameEvent(ev)
		} else if a.gameOver {
			return a.handleGameOverEvent(ev)
		} else if a.inRematch {
			return a.handleRematchEvent(ev)
		}

	case *tcell.EventResize:
		// Handle resize by updating screen
		a.screen.Clear()
		a.render()
	}

	return false
}

// handleLobbyEvent handles events while in the lobby.
func (a *App) handleLobbyEvent(ev *tcell.EventKey) bool {
	if ev.Key() == tcell.KeyEnter {
		// Host can start the game if enough players
		if a.lobbyState.IsHost && a.lobbyState.CanStart {
			if a.server != nil {
				go a.server.StartGameWithCountdown()
			}
		}
	}
	return false
}

// handleGameEvent handles events during gameplay.
func (a *App) handleGameEvent(ev *tcell.EventKey) bool {
	// Handle serve with Enter when waiting
	if ev.Key() == tcell.KeyEnter && a.waitingForServe {
		a.client.SendServe()
		return false
	}

	switch ev.Key() {
	case tcell.KeyUp:
		a.client.SendInput(protocol.DirUp)
	case tcell.KeyDown:
		a.client.SendInput(protocol.DirDown)
	default:
		switch ev.Rune() {
		case 'w', 'W':
			a.client.SendInput(protocol.DirUp)
		case 's', 'S':
			a.client.SendInput(protocol.DirDown)
		}
	}
	return false
}

// handleGameOverEvent handles events on the game over screen.
func (a *App) handleGameOverEvent(ev *tcell.EventKey) bool {
	if ev.Key() == tcell.KeyEnter {
		// Signal ready for rematch
		a.client.SendRematchReady()
	}
	return false
}

// handleRematchEvent handles events on the rematch screen.
func (a *App) handleRematchEvent(ev *tcell.EventKey) bool {
	if ev.Key() == tcell.KeyEnter {
		if a.rematchState.IsHost && a.rematchState.AllReady {
			// Host can start when all ready
			if a.server != nil {
				go a.server.StartGameWithCountdown()
			}
		} else {
			// Mark ourselves ready (both host and non-host)
			a.client.SendRematchReady()
		}
	}
	return false
}

// render calls the appropriate renderer method based on the current state.
func (a *App) render() {
	if a.inCountdown {
		a.renderer.RenderCountdown(a.countdown)
	} else if a.inLobby {
		a.renderer.RenderLobby(a.lobbyState)
	} else if a.inPause {
		a.renderer.RenderPause(a.pauseState)
	} else if a.inGame {
		a.renderer.RenderGame(a.gameState)
	} else if a.gameOver {
		a.renderer.RenderGameOver(a.overState)
	} else if a.inRematch {
		a.renderer.RenderRematch(a.rematchState)
	}
}

// detectSoundEvents compares current and previous game state to trigger sounds
func (a *App) detectSoundEvents(state protocol.GameState) {
	prev := a.prevGameState

	// Skip if no previous state (first frame)
	if prev.Tick == 0 {
		return
	}

	// Detect paddle hit: ball horizontal velocity reversed (and ball is in play area)
	if state.Ball.X > 0 && state.Ball.X < float64(state.CourtWidth) {
		// VX sign changed = hit something
		if (prev.Ball.VX > 0 && state.Ball.VX < 0) || (prev.Ball.VX < 0 && state.Ball.VX > 0) {
			audio.PlayPaddleHit()
		}
	}

	// Detect wall bounce: ball vertical velocity reversed
	if (prev.Ball.VY > 0 && state.Ball.VY < 0) || (prev.Ball.VY < 0 && state.Ball.VY > 0) {
		audio.PlayWallBounce()
	}

	// Detect score: score changed
	if state.LeftScore > prev.LeftScore || state.RightScore > prev.RightScore {
		audio.PlayScore()
	}
}

// cleanup shuts down all resources.
func (a *App) cleanup() {
	// Close audio
	audio.Close()

	// Close client connection
	if a.client != nil {
		a.client.Close()
	}

	// Stop server
	if a.server != nil {
		a.server.Stop()
	}

	// Finalize screen
	if a.screen != nil {
		a.screen.Fini()
	}

	// Stop signal handling
	signal.Stop(a.sigChan)
}

// hasPort checks if the address string contains a port number.
func (a *App) hasPort(addr string) bool {
	return strings.Contains(addr, ":")
}

// generateRandomName creates a random player name.
func (a *App) generateRandomName() string {
	adjectives := []string{"Swift", "Brave", "Quick", "Sharp", "Bold", "Cool", "Fast", "Keen"}
	nouns := []string{"Player", "Paddle", "Pong", "Champ", "Star", "Pro", "Ace", "Hero"}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	adj := adjectives[r.Intn(len(adjectives))]
	noun := nouns[r.Intn(len(nouns))]
	num := r.Intn(100)

	return fmt.Sprintf("%s%s%d", adj, noun, num)
}
