package main

import (
	"log"
	"net/http"
	"os"
	"spotify-chat/internal/ws"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	hub := ws.NewHub()
	go hub.Run()

	http.HandleFunc("/ws", func (w http.ResponseWriter, r *http.Request) {
		log.Print("User connected")
		ws.ServeWs(hub, w, r)
	})

	port := os.Getenv("SERVER_PORT")

	if port == "" {
		port = "8080"
	}

	log.Printf("Listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}