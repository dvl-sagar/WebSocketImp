package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"websocket-server/processor"
	"websocket-server/storage"
)

var clients = make(map[*websocket.Conn]bool)
var mu sync.Mutex

type Request struct {
	// Type string `json:"type"` // "new" or "fetch"
	ID   string `json:"_id"`
	Data string `json:"data,omitempty"`
}

type Response struct {
	ID     string `json:"_id"`
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func HandleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{"json"},
	})
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "Closing connection")

	mu.Lock()
	clients[conn] = true
	mu.Unlock()

	for {
		var req Request
		_, data, err := conn.Read(r.Context())
		if err != nil {
			log.Println("Read error:", err)
			mu.Lock()
			delete(clients, conn)
			mu.Unlock()
			break
		}

		if err := json.Unmarshal(data, &req); err != nil {
			log.Println("Invalid JSON format")
			continue
		}

		if req.ID == "" {
			handleNewRequest(conn, req.Data)
		} else if req.ID != "" {
			handleFetchRequest(conn, req.ID)
		}
	}
}

func handleNewRequest(conn *websocket.Conn, data string) {
	id := uuid.New().String()
	storage.SaveRequest(id, data)

	resp := Response{ID: id}
	sendJSON(conn, resp)

	go func() {
		result := processor.Process(data)
		storage.SaveResult(id, result)

		mu.Lock()
		if _, ok := clients[conn]; ok {
			sendJSON(conn, Response{ID: id, Result: result})
		}
		mu.Unlock()
	}()
}

func handleFetchRequest(conn *websocket.Conn, id string) {
	result, exists := storage.GetResult(id)
	if !exists {
		sendJSON(conn, Response{ID: id, Error: "Request ID not found"})
		return
	}
	sendJSON(conn, Response{ID: id, Result: result})
}

func sendJSON(conn *websocket.Conn, data interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	jsonData, _ := json.Marshal(data)
	err := conn.Write(ctx, websocket.MessageText, jsonData)
	if err != nil {
		log.Println("Write error:", err)
	}
}
