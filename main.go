package main

import (
	"log"
	"net/http"
	"websocket-server/server"
	"websocket-server/storage"
)

func main() {
	storage.InitDB()

	http.Handle("/ws", server.RequestServer{
		Logf: log.Printf,
	})
	log.Println("WebSocket server started on ws://localhost:8080/ws")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
