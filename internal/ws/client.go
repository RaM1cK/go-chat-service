package ws

import (
	// "context"
	"encoding/json"
	"log"
	"net/http"
	// "spotify-chat/domain"
	"spotify-chat/internal/auth"
	// "spotify-chat/internal/service"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Message struct {
	Event    string          `json:"event"`
	Payload  json.RawMessage `json:"payload"`
	Room 	 string 		 `json:"-"`
	SenderID string			 `json:"-"`
}

type Client struct {
	Hub     *Hub
	Conn    *websocket.Conn
	Send    chan RoomMessage
	Rooms   map[string]struct{}
	Inbound chan Message
	UserID  string
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))

			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			out, err := json.Marshal(map[string]any{
				"event":   message.Event,
				"payload": message.Payload,
			})

			if err != nil {
				c.Conn.WriteMessage(websocket.CloseInvalidFramePayloadData, []byte{})
				return
			}

			w.Write(out)
			w.Close()
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))

			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) ReadPump() {
	defer func() {
		for room := range c.Rooms {
			c.Hub.unregister <- Registration{
				Client: c,
				Room:   room,
			}
		}

		close(c.Send)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, raw, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		switch msg.Event {
		case "join-room":
			var room string
			json.Unmarshal(msg.Payload, &room)

			c.Rooms[room] = struct{}{}
			c.Hub.register <- Registration{
				Client: c,
				Room:   room,
			}

		case "send-message":
			var payload struct {
				Room string          `json:"room"`
				Msg  json.RawMessage `json:"msg"`
			}

			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				continue
			}

			c.Inbound <- Message{
				Event: "receive-message",
				Payload: payload.Msg,
				Room: payload.Room,
				SenderID: c.UserID,
			}
		}
	}
}

// func (c *Client) HandleInbound(srv service.MessageService) {
// 	for m := range c.Inbound {
// 		ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
// 		defer cancel()

// 		params := domain.CreateMessageParams{

// 		}
// 	}
// }

func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
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

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{
		Hub:     hub,
		Conn:    conn,
		Send:    make(chan RoomMessage, 256),
		Rooms:   make(map[string]struct{}),
		Inbound: make(chan Message, 256),
		UserID:  userID,
	}

	go client.WritePump()
	go client.ReadPump()
}
