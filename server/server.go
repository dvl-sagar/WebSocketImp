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

type RequestServer struct {
	Logf func(f string, v ...interface{})
}

func (s RequestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{"requests"},
	})
	if err != nil {
		s.Logf("%v", err)
		return
	}
	defer func() {
		c.Close(websocket.StatusNormalClosure, "Closing connection")
	}()

	if c.Subprotocol() != "requests" {
		c.Close(websocket.StatusPolicyViolation, "client must speak the requests subprotocol")
		return
	}

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

func handleRequest(c *websocket.Conn, l *rate.Limiter) error {
	ctx := context.Background()

	if err := l.Wait(ctx); err != nil {
		return err
	}

	typ, r, err := c.Reader(ctx)
	if err != nil {
		return err
	}

	var req struct {
		// Type string `json:"type"` // "new" or "fetch"
		ID   string `json:"_id"`
		Data string `json:"data,omitempty"`
	}
	if err := json.NewDecoder(r).Decode(&req); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	var resp struct {
		ID     string `json:"_id"`
		Result string `json:"result,omitempty"`
		Error  string `json:"error,omitempty"`
	}

	if req.ID == "" {
		resp.ID = uuid.New().String()
		storage.SaveRequest(resp.ID, req.Data)

		func(requestID, requestData string) {
			result := processor.Process(requestData)
			storage.SaveResult(requestID, result)
		}(resp.ID, req.Data)

	} else if req.ID != "" {
		result, exists := storage.GetResult(req.ID)
		if exists {
			resp.ID = req.ID
			resp.Result = result
		} else {
			resp.ID = req.ID
			resp.Error = "Request ID not found"
		}
	}

	w, err := c.Writer(ctx, typ)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return w.Close()
}
