package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"spotify-chat/internal/db"
	"spotify-chat/internal/db/pgsql"
	"spotify-chat/internal/grpc"
	"spotify-chat/internal/repository"
	"spotify-chat/internal/service"
	"spotify-chat/internal/ws"
)

func main() {
	ctx := context.Background()

	if err := db.MigratePgsql(ctx); err != nil {
		log.Fatal(err)
	}

	pool, err := db.NewPgxPool(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	queries := pgsql.New(pool)

	session, err := db.NewScylla(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	msgRepo := repository.NewMessageRepo(session)
	chatRepo := repository.NewChatRepo(queries)

	msgService := service.NewMessageService(msgRepo)
	chatService := service.NewChatService(chatRepo)

	chatServer := grpc.NewChatServer(chatService)

	srv, err := ws.NewServer(msgService)
	if err != nil {
		log.Fatalf("start websocket server: %v", err)
	}
	defer srv.Close()

	http.HandleFunc("/ws", srv.Handle)
	go func() {
		if err := chatServer.Run(os.Getenv("GRPC_PORT")); err != nil {
			log.Fatalf("grpc server: %v", err)
		}
	}()

	port := os.Getenv("SERVER_PORT")

	if port == "" {
		port = "8080"
	}

	log.Printf("Listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
