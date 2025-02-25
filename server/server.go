package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"websocket-server/processor"
	"websocket-server/storage"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

// requestServer handles WebSocket connections for processing requests.
type RequestServer struct {
	Logf func(f string, v ...interface{})
}

// ServeHTTP handles incoming WebSocket connections.
func (s RequestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{"requests"},
	})
	if err != nil {
		s.Logf("%v", err)
		return
	}
	defer c.CloseNow()

	// Enforce subprotocol (optional)
	if c.Subprotocol() != "requests" {
		c.Close(websocket.StatusPolicyViolation, "client must speak the requests subprotocol")
		return
	}

	// Limit: 1 request per 100ms with burst capacity of 10
	l := rate.NewLimiter(rate.Every(time.Millisecond*100), 10)

	for {
		err = handleRequest(c, l)
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			return
		}
		if err != nil {
			s.Logf("Failed to process request from %v: %v", r.RemoteAddr, err)
			return
		}
	}
}

// handleRequest processes incoming WebSocket messages.
func handleRequest(c *websocket.Conn, l *rate.Limiter) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Apply rate limiting
	if err := l.Wait(ctx); err != nil {
		return err
	}

	// Read message from client
	typ, r, err := c.Reader(ctx)
	if err != nil {
		return err
	}

	// Read JSON request
	var req struct {
		Type string `json:"type"` // "new" or "fetch"
		ID   string `json:"id,omitempty"`
		Data string `json:"data,omitempty"`
	}
	if err := json.NewDecoder(r).Decode(&req); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	// Handle request
	var resp struct {
		ID     string `json:"id"`
		Result string `json:"result,omitempty"`
		Error  string `json:"error,omitempty"`
	}

	if req.Type == "new" {
		// Generate request ID, store request, send response
		resp.ID = uuid.New().String()
		storage.SaveRequest(resp.ID, req.Data)

		go func(requestID, requestData string) {
			result := processor.Process(requestData)
			storage.SaveResult(requestID, result)
		}(resp.ID, req.Data)

	} else if req.Type == "fetch" {
		// Fetch stored result
		result, exists := storage.GetResult(req.ID)
		if exists {
			resp.ID = req.ID
			resp.Result = result
		} else {
			resp.ID = req.ID
			resp.Error = "Request ID not found"
		}
	}

	// Send response
	w, err := c.Writer(ctx, typ)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return w.Close()
}
