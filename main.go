package main

import (
	"log"
	"net/http"
	"time"
	server "websocket-server/server"
	"websocket-server/storage"

	"golang.org/x/time/rate"
)

func main() {
	storage.ConnectDB()

	svr := &server.EchoServer{
		Logf:    log.Printf,
		Limiter: rate.NewLimiter(rate.Every(100*time.Millisecond), 10),
	}
	http.Handle("/ws", svr)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	log.Println("Server started on http://localhost:8080")
	log.Println("WebSocket running on ws://localhost:8080/ws")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
