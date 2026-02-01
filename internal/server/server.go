package server

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/diegok/pixpong/internal/config"
	"github.com/diegok/pixpong/internal/game"
	"github.com/diegok/pixpong/internal/protocol"
)

// Server constants
const (
	TickRate      = 60
	MinTermWidth  = 40
	MinTermHeight = 20
)

// Server manages the game server
type Server struct {
	cfg          *config.Config
	listener     net.Listener
	mu           sync.RWMutex
	clients      map[int]*Client
	nextID       int
	gameState    *game.GameState
	inLobby      bool
	inRematch    bool
	rematchReady map[int]bool
	minWidth     int
	minHeight    int
	done         chan struct{}
}

// NewServer creates a new server with the given configuration
func NewServer(cfg *config.Config) *Server {
	return &Server{
		cfg:          cfg,
		clients:      make(map[int]*Client),
		nextID:       1,
		inLobby:      true,
		rematchReady: make(map[int]bool),
		minWidth:     MinTermWidth,
		minHeight:    MinTermHeight,
		done:         make(chan struct{}),
	}
}

// Start begins listening for connections
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	s.listener = listener

	go s.acceptLoop()

	return nil
}

// Stop gracefully shuts down the server
func (s *Server) Stop() {
	s.mu.Lock()
	select {
	case <-s.done:
		s.mu.Unlock()
		return
	default:
		close(s.done)
	}
	s.mu.Unlock()

	if s.listener != nil {
		s.listener.Close()
	}

	s.mu.Lock()
	for _, client := range s.clients {
		client.Close()
	}
	s.mu.Unlock()
}

// GetServerAddresses returns all local IP addresses
func (s *Server) GetServerAddresses() []string {
	var addresses []string

	interfaces, err := net.Interfaces()
	if err != nil {
		return addresses
	}

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Only include IPv4 addresses
			if ip != nil && ip.To4() != nil {
				addresses = append(addresses, fmt.Sprintf("%s:%d", ip.String(), s.cfg.Port))
			}
		}
	}

	return addresses
}

// acceptLoop accepts incoming connections
func (s *Server) acceptLoop() {
	for {
		select {
		case <-s.done:
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				continue
			}
		}

		go s.handleConnection(conn)
	}
}

// handleConnection processes a new client connection
func (s *Server) handleConnection(conn net.Conn) {
	s.mu.Lock()

	// Reject if game is in progress
	if !s.inLobby && !s.inRematch {
		s.mu.Unlock()
		codec := protocol.NewCodec(conn)
		codec.Encode(&protocol.Message{
			Type: protocol.MsgJoinResponse,
			Payload: protocol.JoinResponse{
				Accepted: false,
				Reason:   "Game in progress",
			},
		})
		conn.Close()
		return
	}

	clientID := s.nextID
	s.nextID++
	client := NewClient(clientID, conn)
	s.mu.Unlock()

	// Wait for join request
	msg, err := client.Codec.Decode()
	if err != nil {
		conn.Close()
		return
	}

	if msg.Type != protocol.MsgJoinRequest {
		conn.Close()
		return
	}

	joinReq, ok := msg.Payload.(protocol.JoinRequest)
	if !ok {
		conn.Close()
		return
	}

	// Validate terminal size
	if joinReq.TerminalWidth < MinTermWidth || joinReq.TerminalHeight < MinTermHeight {
		client.SendDirect(&protocol.Message{
			Type: protocol.MsgJoinResponse,
			Payload: protocol.JoinResponse{
				Accepted: false,
				Reason:   fmt.Sprintf("Terminal too small. Minimum: %dx%d", MinTermWidth, MinTermHeight),
			},
		})
		conn.Close()
		return
	}

	// Set client info
	client.Name = joinReq.PlayerName
	if client.Name == "" {
		client.Name = fmt.Sprintf("Player%d", clientID)
	}
	client.Width = joinReq.TerminalWidth
	client.Height = joinReq.TerminalHeight
	client.PlayerID = clientID

	// Send accept response
	err = client.SendDirect(&protocol.Message{
		Type: protocol.MsgJoinResponse,
		Payload: protocol.JoinResponse{
			PlayerID: fmt.Sprintf("%d", clientID),
			Accepted: true,
		},
	})
	if err != nil {
		conn.Close()
		return
	}

	// Add to clients map
	s.mu.Lock()
	s.clients[clientID] = client

	// Update min terminal size
	if client.Width < s.minWidth {
		s.minWidth = client.Width
	}
	if client.Height < s.minHeight {
		s.minHeight = client.Height
	}
	s.mu.Unlock()

	// Start client writer
	client.StartWriter()

	// Broadcast updated lobby state
	s.BroadcastLobbyState()

	// Read messages from client
	for {
		select {
		case <-s.done:
			return
		case <-client.done:
			s.handleDisconnect(clientID)
			return
		default:
		}

		msg, err := client.Codec.Decode()
		if err != nil {
			s.handleDisconnect(clientID)
			return
		}

		s.handleMessage(client, msg)
	}
}

// handleDisconnect handles a client disconnecting
func (s *Server) handleDisconnect(clientID int) {
	s.mu.Lock()
	client, exists := s.clients[clientID]
	if !exists {
		s.mu.Unlock()
		return
	}

	client.Close()
	delete(s.clients, clientID)
	delete(s.rematchReady, clientID)

	// If game is in progress, end it
	wasInGame := s.gameState != nil && !s.inLobby && !s.inRematch
	s.mu.Unlock()

	if wasInGame {
		// End the game due to disconnect
		s.mu.Lock()
		s.inLobby = true
		s.gameState = nil
		s.mu.Unlock()
		s.BroadcastLobbyState()
	} else if s.inRematch {
		s.BroadcastRematchState()
	} else {
		s.BroadcastLobbyState()
	}
}

// handleMessage processes incoming messages from clients
func (s *Server) handleMessage(client *Client, msg *protocol.Message) {
	switch msg.Type {
	case protocol.MsgPlayerInput:
		input, ok := msg.Payload.(protocol.PlayerInput)
		if !ok {
			return
		}

		s.mu.Lock()
		if s.gameState != nil {
			s.gameState.ProcessInput(client.PlayerID, input.Direction)
		}
		s.mu.Unlock()

	case protocol.MsgServe:
		s.mu.Lock()
		if s.gameState != nil {
			s.gameState.Serve(client.PlayerID)
		}
		s.mu.Unlock()

	case protocol.MsgRematchReady:
		s.SetClientRematchReady(client.ID)
	}
}

// StartGame initializes and starts a new game
func (s *Server) StartGame() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.clients) < 2 {
		return
	}

	// Recalculate min terminal size from all clients
	s.minWidth = MinTermWidth
	s.minHeight = MinTermHeight
	for _, client := range s.clients {
		if client.Width < s.minWidth {
			s.minWidth = client.Width
		}
		if client.Height < s.minHeight {
			s.minHeight = client.Height
		}
	}

	// Create game state with minimum terminal size
	s.gameState = game.NewGameState(s.minWidth, s.minHeight, s.cfg.PointsToWin)

	// Add all players to the game
	for _, client := range s.clients {
		s.gameState.AddPlayer(client.ID, client.Name)
	}

	// Assign teams randomly
	s.gameState.AssignTeams()

	s.inLobby = false
	s.inRematch = false
}

// StartGameWithCountdown starts the game with a 3,2,1 countdown
func (s *Server) StartGameWithCountdown() {
	// Send countdown 3, 2, 1
	for i := 3; i > 0; i-- {
		s.broadcast(&protocol.Message{
			Type: protocol.MsgCountdown,
			Payload: protocol.Countdown{
				Seconds: i,
			},
		})
		time.Sleep(time.Second)
	}

	s.StartGame()

	// Start the game loop
	go s.gameLoop()
}

// gameLoop runs the game at 60Hz
func (s *Server) gameLoop() {
	ticker := time.NewTicker(time.Second / TickRate)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			if s.gameState == nil {
				s.mu.Unlock()
				return
			}

			// Update game state
			s.gameState.Update()

			// Check if game is over
			gameOver := s.gameState.IsGameOver()

			// Prepare state to broadcast
			var msg *protocol.Message
			if s.gameState.Paused || s.gameState.WaitingForServe {
				msg = &protocol.Message{
					Type: protocol.MsgPauseState,
					Payload: protocol.PauseState{
						SecondsLeft:     s.gameState.PauseTicksLeft / TickRate,
						LeftScore:       s.gameState.LeftScore,
						RightScore:      s.gameState.RightScore,
						LastScorer:      s.gameState.LastScorer,
						WaitingForServe: s.gameState.WaitingForServe,
						ServingTeam:     s.gameState.ServingTeam,
					},
				}
			} else {
				msg = &protocol.Message{
					Type:    protocol.MsgGameState,
					Payload: s.gameState.ToProtocolState(),
				}
			}
			s.mu.Unlock()

			// Broadcast game state
			s.broadcast(msg)

			if gameOver {
				s.broadcastGameOver()
				return
			}
		}
	}
}

// broadcast sends a message to all connected clients
func (s *Server) broadcast(msg *protocol.Message) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, client := range s.clients {
		client.Send(msg)
	}
}

// BroadcastLobbyState sends the current lobby state to all clients
func (s *Server) BroadcastLobbyState() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build player list
	players := make([]protocol.LobbyPlayer, 0, len(s.clients))
	for _, client := range s.clients {
		players = append(players, protocol.LobbyPlayer{
			ID:    fmt.Sprintf("%d", client.ID),
			Name:  client.Name,
			Color: (client.ID - 1) % 8,
		})
	}

	// Get server addresses
	addresses := s.GetServerAddresses()

	canStart := len(s.clients) >= 2

	// Send to each client
	for _, client := range s.clients {
		isHost := client.ID == 1 // First client is host

		msg := &protocol.Message{
			Type: protocol.MsgLobbyState,
			Payload: protocol.LobbyState{
				Players:     players,
				IsHost:      isHost,
				CanStart:    canStart,
				ServerAddrs: nil, // Only host sees addresses
				PointsToWin: s.cfg.PointsToWin,
			},
		}

		// Only send server addresses to host
		if isHost {
			lobbyState := msg.Payload.(protocol.LobbyState)
			lobbyState.ServerAddrs = addresses
			msg.Payload = lobbyState
		}

		client.Send(msg)
	}
}

// BroadcastRematchState sends the current rematch state to all clients
func (s *Server) BroadcastRematchState() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Build player list with ready status
	players := make([]protocol.RematchPlayer, 0, len(s.clients))
	allReady := true
	for _, client := range s.clients {
		ready := s.rematchReady[client.ID]
		if !ready {
			allReady = false
		}
		players = append(players, protocol.RematchPlayer{
			ID:    fmt.Sprintf("%d", client.ID),
			Name:  client.Name,
			Color: (client.ID - 1) % 8,
			Ready: ready,
		})
	}

	// Need at least 2 players and all must be ready
	if len(s.clients) < 2 {
		allReady = false
	}

	// Send to each client
	for _, client := range s.clients {
		isHost := client.ID == 1

		msg := &protocol.Message{
			Type: protocol.MsgRematchState,
			Payload: protocol.RematchState{
				Players:  players,
				IsHost:   isHost,
				AllReady: allReady,
			},
		}

		client.Send(msg)
	}
}

// ResetForRematch enters rematch mode
func (s *Server) ResetForRematch() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.inRematch = true
	s.inLobby = false
	s.gameState = nil
	s.rematchReady = make(map[int]bool)
}

// SetClientRematchReady marks a client as ready for rematch
func (s *Server) SetClientRematchReady(clientID int) {
	s.mu.Lock()
	s.rematchReady[clientID] = true
	s.mu.Unlock()

	s.BroadcastRematchState()
}

// broadcastGameOver sends the game over state to all clients
func (s *Server) broadcastGameOver() {
	s.mu.RLock()
	if s.gameState == nil {
		s.mu.RUnlock()
		return
	}

	msg := &protocol.Message{
		Type: protocol.MsgGameOver,
		Payload: protocol.GameOverState{
			WinningTeam: s.gameState.GetWinner(),
			LeftScore:   s.gameState.LeftScore,
			RightScore:  s.gameState.RightScore,
		},
	}
	s.mu.RUnlock()

	s.broadcast(msg)

	// Enter rematch mode
	s.ResetForRematch()
	s.BroadcastRematchState()
}

// removeClient removes a client from the server
func (s *Server) removeClient(clientID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	client, exists := s.clients[clientID]
	if !exists {
		return
	}

	client.Close()
	delete(s.clients, clientID)
	delete(s.rematchReady, clientID)
}
