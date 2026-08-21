package ws

import "sync"

type Registration struct {
	Client *Client
	Room string
}

type Hub struct {
	mu         sync.RWMutex
	rooms      map[string]map[*Client]struct{}
	broadcast  chan RoomMessage
	register   chan Registration
	unregister chan Registration
}

type RoomMessage struct {
	Room  string
	Event string
	Payload  any
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]struct{}),
		broadcast:  make(chan RoomMessage),
		register:   make(chan Registration),
		unregister: make(chan Registration),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case reg := <-h.register:
			h.mu.Lock()

			if h.rooms[reg.Room] == nil {
				h.rooms[reg.Room] = make(map[*Client]struct{})
			}

			h.rooms[reg.Room][reg.Client] = struct{}{}
			h.mu.Unlock()
		case reg := <-h.unregister:
			h.mu.Lock()

			if clients, ok := h.rooms[reg.Room]; ok {
				delete(clients, reg.Client)

				if len(clients) == 0 {
					delete(h.rooms, reg.Room)
				}
			}

			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.RLock()
			clients := h.rooms[message.Room]
			h.mu.RUnlock()

			var dead []*Client
			for c := range clients {
				select {
				case c.Send <- message:
				default:
					dead = append(dead, c)
				}
			}

			if len(dead) > 0 {
				h.mu.Lock()
				for _, c := range dead {
					delete(clients, c)
				}

				if len(clients) == 0 {
					delete(h.rooms, message.Room)
				}
				h.mu.Unlock()
			}
		}
	}
}

func (h *Hub) BroadcastToRoom(room, event string, payload any) {
	h.broadcast <- RoomMessage{
		Room: room,
		Event: event,
		Payload: payload,
	}
}