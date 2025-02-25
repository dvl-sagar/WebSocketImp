package main

import (
	"log"
	"net/http"
	"websocket-server/server"
	"websocket-server/storage"
)

func main() {
	storage.ConnectDB()

	http.Handle("/ws", server.RequestServer{
		Logf: log.Printf,
	})
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	log.Println("Server started on http://localhost:8080")
	log.Println("WebSocket running on ws://localhost:8080/ws")
	// log.Println("WebSocket server started on ws://localhost:8080/ws")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
