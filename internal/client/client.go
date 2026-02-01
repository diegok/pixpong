package client

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/diegok/pixpong/internal/protocol"
)

const (
	channelBufferSize = 16
	connectTimeout    = 5 * time.Second
)

// Client represents a network client that connects to a pixpong server.
type Client struct {
	Name         string
	Width        int
	Height       int
	PlayerID     string
	conn         net.Conn
	codec        *protocol.Codec
	mu           sync.Mutex
	connected    bool
	GameState    chan protocol.GameState
	LobbyState   chan protocol.LobbyState
	GameOver     chan protocol.GameOverState
	RematchState chan protocol.RematchState
	Countdown    chan protocol.Countdown
	PauseState   chan protocol.PauseState
	GameStart    chan struct{}
	Error        chan error
	done         chan struct{}
}

// NewClient creates a new client with the given name and terminal dimensions.
func NewClient(name string, width, height int) *Client {
	return &Client{
		Name:         name,
		Width:        width,
		Height:       height,
		PlayerID:     "",
		GameState:    make(chan protocol.GameState, channelBufferSize),
		LobbyState:   make(chan protocol.LobbyState, channelBufferSize),
		GameOver:     make(chan protocol.GameOverState, channelBufferSize),
		RematchState: make(chan protocol.RematchState, channelBufferSize),
		Countdown:    make(chan protocol.Countdown, channelBufferSize),
		PauseState:   make(chan protocol.PauseState, channelBufferSize),
		GameStart:    make(chan struct{}, 1),
		Error:        make(chan error, channelBufferSize),
		done:         make(chan struct{}),
	}
}

// Connect establishes a connection to the server at the given address.
// It sends a JoinRequest and waits for a JoinResponse before returning.
func (c *Client) Connect(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, connectTimeout)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	c.conn = conn
	c.codec = protocol.NewCodec(conn)

	// Send join request
	joinReq := protocol.Message{
		Type: protocol.MsgJoinRequest,
		Payload: protocol.JoinRequest{
			PlayerName:     c.Name,
			TerminalWidth:  c.Width,
			TerminalHeight: c.Height,
		},
	}
	if err := c.codec.Encode(&joinReq); err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to send join request: %w", err)
	}

	// Set read deadline for join response
	c.conn.SetReadDeadline(time.Now().Add(connectTimeout))

	// Wait for join response
	msg, err := c.codec.Decode()
	if err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to receive join response: %w", err)
	}

	// Clear read deadline
	c.conn.SetReadDeadline(time.Time{})

	if msg.Type != protocol.MsgJoinResponse {
		c.conn.Close()
		return fmt.Errorf("expected join response, got message type %d", msg.Type)
	}

	resp, ok := msg.Payload.(protocol.JoinResponse)
	if !ok {
		c.conn.Close()
		return fmt.Errorf("invalid join response payload")
	}

	if !resp.Accepted {
		c.conn.Close()
		return fmt.Errorf("join request rejected: %s", resp.Reason)
	}

	c.PlayerID = resp.PlayerID
	c.mu.Lock()
	c.connected = true
	c.mu.Unlock()

	// Start receive loop
	go c.receiveLoop()

	return nil
}

// SendInput sends a player input message to the server.
func (c *Client) SendInput(dir protocol.Direction) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return fmt.Errorf("not connected to server")
	}

	msg := protocol.Message{
		Type: protocol.MsgPlayerInput,
		Payload: protocol.PlayerInput{
			Direction: dir,
		},
	}
	return c.codec.Encode(&msg)
}

// SendRematchReady sends a rematch ready message to the server.
func (c *Client) SendRematchReady() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return fmt.Errorf("not connected to server")
	}

	msg := protocol.Message{
		Type:    protocol.MsgRematchReady,
		Payload: nil,
	}
	return c.codec.Encode(&msg)
}

// SendServe sends a serve request to the server.
func (c *Client) SendServe() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return fmt.Errorf("not connected to server")
	}

	msg := protocol.Message{
		Type:    protocol.MsgServe,
		Payload: nil,
	}
	return c.codec.Encode(&msg)
}

// Close closes the connection to the server.
func (c *Client) Close() {
	c.mu.Lock()
	wasConnected := c.connected
	c.connected = false
	c.mu.Unlock()

	if wasConnected {
		close(c.done)
		if c.conn != nil {
			c.conn.Close()
		}
	}
}

// IsConnected returns true if the client is connected to the server.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// receiveLoop continuously reads messages from the server and dispatches them.
func (c *Client) receiveLoop() {
	defer func() {
		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()
	}()

	for {
		select {
		case <-c.done:
			return
		default:
		}

		msg, err := c.codec.Decode()
		if err != nil {
			select {
			case <-c.done:
				return
			default:
				select {
				case c.Error <- fmt.Errorf("receive error: %w", err):
				default:
					// Drop error if channel is full
				}
				return
			}
		}

		c.dispatchMessage(msg)
	}
}

// dispatchMessage routes a message to the appropriate channel.
func (c *Client) dispatchMessage(msg *protocol.Message) {
	switch msg.Type {
	case protocol.MsgGameState:
		if state, ok := msg.Payload.(protocol.GameState); ok {
			select {
			case c.GameState <- state:
			default:
				// Drop old message if channel is full
				select {
				case <-c.GameState:
				default:
				}
				c.GameState <- state
			}
		}

	case protocol.MsgLobbyState:
		if state, ok := msg.Payload.(protocol.LobbyState); ok {
			select {
			case c.LobbyState <- state:
			default:
				// Drop old message if channel is full
				select {
				case <-c.LobbyState:
				default:
				}
				c.LobbyState <- state
			}
		}

	case protocol.MsgGameOver:
		if state, ok := msg.Payload.(protocol.GameOverState); ok {
			select {
			case c.GameOver <- state:
			default:
				// Drop old message if channel is full
				select {
				case <-c.GameOver:
				default:
				}
				c.GameOver <- state
			}
		}

	case protocol.MsgRematchState:
		if state, ok := msg.Payload.(protocol.RematchState); ok {
			select {
			case c.RematchState <- state:
			default:
				// Drop old message if channel is full
				select {
				case <-c.RematchState:
				default:
				}
				c.RematchState <- state
			}
		}

	case protocol.MsgCountdown:
		if state, ok := msg.Payload.(protocol.Countdown); ok {
			select {
			case c.Countdown <- state:
			default:
				// Drop old message if channel is full
				select {
				case <-c.Countdown:
				default:
				}
				c.Countdown <- state
			}
		}

	case protocol.MsgPauseState:
		if state, ok := msg.Payload.(protocol.PauseState); ok {
			select {
			case c.PauseState <- state:
			default:
				// Drop old message if channel is full
				select {
				case <-c.PauseState:
				default:
				}
				c.PauseState <- state
			}
		}

	case protocol.MsgStartGame:
		select {
		case c.GameStart <- struct{}{}:
		default:
			// Only keep one game start signal
		}
	}
}
