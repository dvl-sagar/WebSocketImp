package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	md "websocket-server/model"
	pc "websocket-server/processor"
	st "websocket-server/storage"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

type EchoServer struct {
	Logf    func(f string, v ...any)
	Limiter *rate.Limiter
}

func (s *EchoServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		s.Logf("WebSocket accept error: %v", err)
		return
	}
	defer c.Close(websocket.StatusNormalClosure, "Closing connection")

	pendingRequests := st.GetPendingRequests()
	if len(pendingRequests) > 0 {
		resp := struct {
			Pending []string `json:"pending"`
		}{Pending: pendingRequests}
		if err := sendResponse(context.Background(), c, websocket.MessageText, resp); err != nil {
			s.Logf("Error sending pending requests: %v", err)
			return
		}
	}

	handler := RequestHandler{Limiter: s.Limiter}
	for {
		if err := handler.HandleRequest(c); err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
				return
			}
			s.Logf("Request processing error: %v", err)
			return
		}
	}
}

type RequestHandler struct {
	Limiter *rate.Limiter
}

func (h *RequestHandler) HandleRequest(conxn *websocket.Conn) error {
	ctx := context.Background()

	if err := h.Limiter.Wait(ctx); err != nil {
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

	if req.ID == "" {
		return h.handleNewRequest(ctx, conxn, typ, req)
	}
	return h.handleExistingRequest(ctx, conxn, typ, req)
}

func (h *RequestHandler) handleNewRequest(ctx context.Context, conxn *websocket.Conn, typ websocket.MessageType, req md.Request) error {
	ID := uuid.New().String()
	st.SaveRequest(ID, req.Data)
	resp := md.Response{ID: ID}

	if err := sendResponse(ctx, conxn, typ, resp); err != nil {
		return err
	}

	h.processAndStoreResult(ID, req.Data)
	return h.sendStoredResult(ctx, conxn, typ, ID)
}

func (h *RequestHandler) handleExistingRequest(ctx context.Context, conxn *websocket.Conn, typ websocket.MessageType, req md.Request) error {
	result, exists := st.GetResult(req.ID)
	resp := md.Response{ID: req.ID}

	if exists {
		resp.Result = result
	} else {
		resp.Error = "Request ID not found"
	}

	return sendResponse(ctx, conxn, typ, resp)
}

func (h *RequestHandler) processAndStoreResult(requestID string, requestData any) {
	result := pc.Process(requestData)
	st.SaveResult(requestID, result)
}

func (h *RequestHandler) sendStoredResult(ctx context.Context, conxn *websocket.Conn, typ websocket.MessageType, ID string) error {
	result, exists := st.GetResult(ID)
	if !exists {
		return nil
	}

	resp := md.Response{ID: ID, Result: result}
	return sendResponse(ctx, conxn, typ, resp)
}

func sendResponse(ctx context.Context, conxn *websocket.Conn, typ websocket.MessageType, resp interface{}) error {
	respByte, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return conxn.Write(ctx, typ, respByte)
}
