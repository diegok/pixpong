package server

import (
	"net"
	"sync"

	"github.com/diegok/pixpong/internal/protocol"
)

const sendBufferSize = 64

// Client represents a connected player on the server
type Client struct {
	ID       int
	Name     string
	Width    int
	Height   int
	PlayerID int
	conn     net.Conn
	Codec    *protocol.Codec
	sendCh   chan *protocol.Message
	done     chan struct{}
	mu       sync.Mutex
}

// NewClient creates a new client with the given connection
func NewClient(id int, conn net.Conn) *Client {
	return &Client{
		ID:       id,
		PlayerID: -1,
		conn:     conn,
		Codec:    protocol.NewCodec(conn),
		sendCh:   make(chan *protocol.Message, sendBufferSize),
		done:     make(chan struct{}),
	}
}

// StartWriter starts the goroutine that writes messages to the connection
func (c *Client) StartWriter() {
	go func() {
		for {
			select {
			case <-c.done:
				return
			case msg := <-c.sendCh:
				if err := c.Codec.Encode(msg); err != nil {
					return
				}
			}
		}
	}()
}

// Send queues a message to be sent to the client (non-blocking)
func (c *Client) Send(msg *protocol.Message) {
	select {
	case c.sendCh <- msg:
	default:
		// Buffer full, drop message
	}
}

// SendDirect sends a message immediately (for handshake)
func (c *Client) SendDirect(msg *protocol.Message) error {
	return c.Codec.Encode(msg)
}

// Close closes the client connection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case <-c.done:
		return
	default:
		close(c.done)
	}

	if c.conn != nil {
		c.conn.Close()
	}
}
