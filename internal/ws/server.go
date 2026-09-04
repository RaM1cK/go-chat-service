package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"spotify-chat/internal/auth"
	"spotify-chat/internal/dto"
	"spotify-chat/internal/service"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/olahol/melody"
)

const (
	roomSubjectPrefix = "room."
	natsTimeout       = 5 * time.Second
)

type Event string

const (
	MessageStatus  Event = "message-status"
	JoinRoom       Event = "join-room"
	SendMessage    Event = "send-message"
	ReceiveMessage Event = "receive-message"
)

type Message[T any] struct {
	Event   Event  `json:"event"`
	Payload T      `json:"payload"`
	Room    string `json:"room"`
}

type Server struct {
	m  *melody.Melody
	nc *nats.Conn
}

func NewServer(srv service.MessageService) (*Server, error) {
	nc, err := nats.Connect(natsUrl(), nats.Name("spoty-chat-hub"), nats.Timeout(natsTimeout))
	if err != nil {
		return nil, err
	}

	s := &Server{m: melody.New(), nc: nc}
	s.m.Upgrader.CheckOrigin = func(*http.Request) bool { return true }
	s.m.HandleMessage(s.handleMessage(srv))

	if _, err := nc.Subscribe(roomSubjectPrefix+">", s.deliverFromNats); err != nil {
		nc.Close()
		return nil, err
	}

	return s, nil
}

func (s *Server) Close() {
	if s.nc != nil {
		s.nc.Close()
	}
}

func (s *Server) Handle(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := auth.ValidateJWT(cookie.Value)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	s.m.HandleRequestWithKeys(w, r, map[string]any{
		"userID": userID,
		"rooms":  []string{},
	})
}

func (s *Server) handleMessage(srv service.MessageService) func(*melody.Session, []byte) {
	return func(sess *melody.Session, raw []byte) {
		var msg Message[json.RawMessage]
		if err := json.Unmarshal(raw, &msg); err != nil {
			return
		}

		switch msg.Event {
		case JoinRoom:
			var room string
			if err := json.Unmarshal(msg.Payload, &room); err != nil {
				return
			}
			rooms := sess.Keys["rooms"].([]string)
			rooms = append(rooms, room)
			sess.Set("rooms", rooms)

		case SendMessage:
			var payload struct {
				Room string          `json:"room"`
				Msg  json.RawMessage `json:"msg"`
			}
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				return
			}
			s.sendMessage(srv, sess, payload.Room, payload.Msg)

		default:
		}
	}
}

func (s *Server) sendMessage(srv service.MessageService, sess *melody.Session, room string, raw json.RawMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var params dto.CreateMessageParams
	if err := json.Unmarshal(raw, &params); err != nil {
		return
	}

	params.SenderID = s.userID(sess)

	chatID, err := uuid.Parse(room)
	if err != nil {
		return
	}
	params.ChatID = chatID

	saved, err := srv.SaveMessage(ctx, params)
	if err != nil {
		s.writeStatus(sess, params.ID, "failed")
		return
	}

	s.writeStatus(sess, params.ID, "sent")

	data, err := json.Marshal(Message[any]{
		Room:    room,
		Event:   ReceiveMessage,
		Payload: saved,
	})
	if err != nil {
		return
	}

	s.nc.Publish(roomSubjectPrefix+room, data)
}

func (s *Server) writeStatus(sess *melody.Session, id uuid.UUID, status string) {
	data, err := json.Marshal(map[string]any{
		"event": MessageStatus,
		"payload": map[string]any{
			"id":     id,
			"status": status,
		},
	})
	if err != nil {
		return
	}
	sess.Write(data)
}

func (s *Server) userID(sess *melody.Session) uuid.UUID {
	id, _ := sess.Get("userID")
	uid, ok := id.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return uid
}

func (s *Server) deliverFromNats(m *nats.Msg) {
	var msg Message[any]
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		return
	}

	room := strings.TrimPrefix(m.Subject, roomSubjectPrefix)
	if room != msg.Room {
		return
	}

	wire, err := json.Marshal(map[string]any{
		"event":   msg.Event,
		"payload": msg.Payload,
	})
	if err != nil {
		return
	}

	s.m.BroadcastFilter(wire, func(q *melody.Session) bool {
		rooms, ok := q.Get("rooms")
		if !ok || rooms == nil {
			return false
		}
		return slices.Contains(rooms.([]string), room)
	})
}

func natsUrl() string {
	if url := os.Getenv("NATS_URL"); url != "" {
		return url
	}
	return "nats://localhost:4222"
}
