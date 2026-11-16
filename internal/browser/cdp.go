package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// CDPRequest represents a Chrome DevTools Protocol command request.
type CDPRequest struct {
	ID        int64       `json:"id"`
	SessionID string      `json:"sessionId,omitempty"`
	Method    string      `json:"method"`
	Params    interface{} `json:"params,omitempty"`
}

// CDPResponse represents a response from the browser DevTools backend.
type CDPResponse struct {
	ID        int64           `json:"id,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
	Method    string          `json:"method,omitempty"`
	Params    json.RawMessage `json:"params,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *CDPError       `json:"error,omitempty"`
}

// CDPError captures structured errors returned by CDP.
type CDPError struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

func (e *CDPError) Error() string {
	return fmt.Sprintf("cdp error [%d]: %s (%s)", e.Code, e.Message, e.Data)
}

// EventHandler processes CDP asynchronous events.
type EventHandler func(method string, params json.RawMessage)

// CDPClient manages a WebSocket connection to Chrome DevTools Protocol.
type CDPClient struct {
	conn       *websocket.Conn
	mu         sync.Mutex
	writeMu    sync.Mutex
	reqCounter int64
	pending    map[int64]chan *CDPResponse
	listeners  []EventHandler
	closed     chan struct{}
	closeOnce  sync.Once
}

// NewCDPClient establishes a WebSocket connection to the DevTools URL.
func NewCDPClient(ctx context.Context, wsURL string) (*CDPClient, error) {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket dial to %s failed: %w", wsURL, err)
	}

	client := &CDPClient{
		conn:    conn,
		pending: make(map[int64]chan *CDPResponse),
		closed:  make(chan struct{}),
	}

	go client.readLoop()

	return client, nil
}

func (c *CDPClient) readLoop() {
	defer func() {
		c.closeOnce.Do(func() {
			close(c.closed)
		})
		c.mu.Lock()
		for _, ch := range c.pending {
			close(ch)
		}
		c.pending = make(map[int64]chan *CDPResponse)
		c.mu.Unlock()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		var resp CDPResponse
		if err := json.Unmarshal(message, &resp); err != nil {
			continue
		}

		// Check if it is a response to a pending request
		if resp.ID > 0 {
			c.mu.Lock()
			ch, ok := c.pending[resp.ID]
			if ok {
				delete(c.pending, resp.ID)
			}
			c.mu.Unlock()

			if ok && ch != nil {
				ch <- &resp
				close(ch)
				continue
			}
		}

		// Otherwise, it is an asynchronous event
		if resp.Method != "" {
			c.mu.Lock()
			handlers := make([]EventHandler, len(c.listeners))
			copy(handlers, c.listeners)
			c.mu.Unlock()

			for _, h := range handlers {
				go h(resp.Method, resp.Params)
			}
		}
	}
}

// OnEvent registers an asynchronous event listener.
func (c *CDPClient) OnEvent(handler EventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.listeners = append(c.listeners, handler)
}

// Call executes a CDP command synchronously, blocking until result or context cancellation.
func (c *CDPClient) Call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	return c.CallWithSession(ctx, "", method, params)
}

// CallWithSession executes a CDP command targeted to a specific tab session ID.
func (c *CDPClient) CallWithSession(ctx context.Context, sessionID, method string, params interface{}) (json.RawMessage, error) {
	select {
	case <-c.closed:
		return nil, errors.New("cdp client is closed")
	default:
	}

	id := atomic.AddInt64(&c.reqCounter, 1)
	req := CDPRequest{
		ID:        id,
		SessionID: sessionID,
		Method:    method,
		Params:    params,
	}

	resCh := make(chan *CDPResponse, 1)

	c.mu.Lock()
	c.pending[id] = resCh
	c.mu.Unlock()

	data, err := json.Marshal(req)
	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("failed marshaling cdp request: %w", err)
	}

	c.writeMu.Lock()
	err = c.conn.WriteMessage(websocket.TextMessage, data)
	c.writeMu.Unlock()

	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("failed sending cdp command %s: %w", method, err)
	}

	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	case <-c.closed:
		return nil, errors.New("cdp connection closed during command execution")
	case resp, ok := <-resCh:
		if !ok || resp == nil {
			return nil, errors.New("no response received from cdp")
		}
		if resp.Error != nil {
			return nil, resp.Error
		}
		return resp.Result, nil
	}
}

// Close terminates the websocket connection.
func (c *CDPClient) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.closed)
		err = c.conn.Close()
	})
	return err
}
