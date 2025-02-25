package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"websocket-server/processor"
	"websocket-server/storage"

	md "websocket-server/model"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

type RequestServer struct {
	Logf func(f string, v ...interface{})
}

func (s RequestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		s.Logf("WebSocket accept error: %v", err)
		return
	}
	defer c.Close(websocket.StatusNormalClosure, "Closing connection")

	l := rate.NewLimiter(rate.Every(time.Millisecond*100), 10) // Rate limiter

	pendingRequests := storage.GetPendingRequests()

	if len(pendingRequests) > 0 {
		resp := struct {
			Pending []string `json:"pending"`
		}{
			Pending: pendingRequests,
		}

		respByte, err := json.Marshal(resp)
		if err != nil {
			s.Logf("Error encoding pending requests: %v", err)
			return
		}

		err = c.Write(context.Background(), websocket.MessageText, respByte)
		if err != nil {
			s.Logf("Error sending pending requests: %v", err)
			return
		}
	}

	for {
		err = handleRequest(c, l)
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			return
		}
		if err != nil {
			s.Logf("Request processing error: %v", err)
			return
		}
	}
}

func handleRequest(conxn *websocket.Conn, l *rate.Limiter) error {
	ctx := context.Background()

	if err := l.Wait(ctx); err != nil {
		return err
	}

	typ, r, err := conxn.Reader(ctx)
	if err != nil {
		return err
	}

	var req md.Request
	if err := json.NewDecoder(r).Decode(&req); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	var resp md.Response

	if req.ID == "" {
		ID := uuid.New().String()
		storage.SaveRequest(ID, req.Data)
		resp.ID = ID

		respByte, err := json.Marshal(resp)
		if err != nil {
			return err
		}
		err = conxn.Write(ctx, typ, respByte)
		if err != nil {
			return err
		}

		func(requestID string, requestData any) {
			result := processor.Process(requestData)
			storage.SaveResult(requestID, result)
		}(resp.ID, req.Data)

		result, exists := storage.GetResult(ID)
		var newResp md.Response
		if exists {
			newResp.ID = ID
			newResp.Result = result
		}
		respByte, err = json.Marshal(newResp)
		if err != nil {
			return err
		}
		err = conxn.Write(ctx, typ, respByte)
		if err != nil {
			return err
		}

	} else if req.ID != "" {
		result, exists := storage.GetResult(req.ID)
		if exists {
			resp.ID = req.ID
			resp.Result = result
		} else {
			resp.ID = req.ID
			resp.Error = "Request ID not found"
		}

		respByte, err := json.Marshal(resp)
		if err != nil {
			return err
		}
		err = conxn.Write(ctx, typ, respByte)
		if err != nil {
			return err
		}
	}

	return nil
}
