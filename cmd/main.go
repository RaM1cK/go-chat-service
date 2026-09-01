package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

	"spotify-chat/internal/db/pgsql"
	"spotify-chat/internal/repository"
	"spotify-chat/internal/service"
	"spotify-chat/internal/ws"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/gocql/gocql"
	"github.com/scylladb/gocqlx/v3"
	"github.com/scylladb/gocqlx/v3/migrate"
)

func main() {
	ctx := context.Background()

	pgMigrations, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer pgMigrations.Close()

	if err := goose.Up(pgMigrations, "migrations/pgsql"); err != nil {
		log.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	queries := pgsql.New(pool)

	cluster := gocql.NewCluster(os.Getenv("SCYLLA_URL"))

	initSession, err := cluster.CreateSession()
	if err != nil {
		log.Fatal(err)
	}
	keyspace := os.Getenv("SCYLLA_KEYSPACE")
	if err := initSession.Query(
		`CREATE KEYSPACE IF NOT EXISTS ` + keyspace +
			` WITH replication = {'class': 'NetworkTopologyStrategy', 'datacenter1': 1}`).Exec(); err != nil {
		log.Fatal(err)
	}
	initSession.Close()

	cluster.Keyspace = keyspace
	session, err := gocqlx.WrapSession(cluster.CreateSession())
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	if err := migrate.FromFS(context.Background(), session, os.DirFS("migrations/scylla")); err != nil {
		log.Fatal(err)
	}

	msgRepo := repository.NewMessageRepo(session)
	chatRepo := repository.NewChatRepo(queries)

	msgService := service.NewMessageService(msgRepo)
	chatService := service.NewChatService(chatRepo)

	_ = chatService

	hub := ws.NewHub()
	go hub.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWs(hub, msgService, w, r)
	})

	port := os.Getenv("SERVER_PORT")

	if port == "" {
		port = "8080"
	}

	log.Printf("Listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
