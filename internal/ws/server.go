package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"spotify-chat/internal/auth"
	"spotify-chat/internal/dto"
	"spotify-chat/internal/service"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	sio "github.com/zishang520/socket.io/servers/socket/v3"
	"github.com/zishang520/socket.io/v3/pkg/slices"
	"github.com/zishang520/socket.io/v3/pkg/types"
)

const (
	roomSubjectPrefix = "room."
	roomPrefix        = "chat:"
	chatPath          = "/ws"
	natsTimeout       = 5 * time.Second
)

type Event string

const (
	MessageStatus  Event = "message-status"
	JoinRoom       Event = "join-room"
	SendMessage    Event = "send-message"
	ReceiveMessage Event = "receive-message"
)

// sendMessageRequest is the payload of the "send-message" event.
type sendMessageRequest struct {
	Room uuid.UUID               `json:"room"`
	Msg  dto.CreateMessageParams `json:"msg"`
}

// natsMessage is the envelope published on NATS to fan out messages across hubs.
type natsMessage struct {
	Event    Event       `json:"event"`
	Room     string      `json:"room"`
	Payload  dto.Message `json:"payload"`
	SenderID string      `json:"senderId"`
	// SenderSid is the id of the socket that sent the message, so the broadcast
	// can skip it while still reaching the sender's other devices.
	SenderSid string `json:"senderSid"`
}

type Server struct {
	io *sio.Server
	nc *nats.Conn
}

func chatRoom(id string) sio.Room {
	return sio.Room(roomPrefix + id)
}

func NewServer(srv service.MessageService) (*Server, error) {
	nc, err := nats.Connect(natsUrl(), nats.Name("spoty-chat-hub"), nats.Timeout(natsTimeout))
	if err != nil {
		return nil, err
	}

	s := &Server{nc: nc}

	opts := sio.DefaultServerOptions()
	opts.SetPath(chatPath)
	opts.SetAllowRequest(s.allowRequest)
	opts.SetCors(&types.Cors{Origin: os.Getenv("WS_ORIGIN")})

	s.io = sio.NewServer(nil, opts)
	s.io.On("connection", s.onConnection(srv))

	if _, err := nc.Subscribe(roomSubjectPrefix+">", s.deliverFromNats); err != nil {
		nc.Close()
		return nil, err
	}

	return s, nil
}

// Handler returns an http.Handler serving the socket.io endpoint.
func (s *Server) Handler() http.Handler {
	return s.io.ServeHandler(nil)
}

func (s *Server) Close() {
	if s.nc != nil {
		s.nc.Close()
	}
}

// allowRequest rejects handshakes without a valid JWT cookie.
func (s *Server) allowRequest(ctx *types.HttpContext) error {
	if _, err := authenticate(ctx.Request()); err != nil {
		log.Println("handshake rejected:", err)
		return err
	}
	return nil
}

func (s *Server) socketUserID(sock *sio.Socket) (uuid.UUID, error) {
	return authenticate(sock.Request().Request())
}

// authenticate resolves the user ID from the JWT "token" cookie.
func authenticate(req *http.Request) (uuid.UUID, error) {
	cookie, err := req.Cookie("token")
	if err != nil {
		return uuid.Nil, err
	}
	return auth.ValidateJWT(cookie.Value)
}

// onConnection wires up the per-socket event handlers.
func (s *Server) onConnection(srv service.MessageService) func(...any) {
	return connectSocket(func(sock *sio.Socket) {
		userID, err := s.socketUserID(sock)
		if err != nil {
			sock.Disconnect(false)
			return
		}
		sock.SetData(userID)

		sock.On(string(JoinRoom), s.onJoinRoom(sock))
		sock.On(string(SendMessage), s.onSendMessage(srv, sock))
	})
}

// connectSocket adapts the library's variadic listener to a typed socket callback.
func connectSocket(fn func(*sio.Socket)) func(...any) {
	return func(clients ...any) {
		sock, ok := sockFrom(clients)
		if !ok {
			return
		}
		fn(sock)
	}
}

// sockFrom extracts the *sio.Socket from the "connection" event arguments.
func sockFrom(clients []any) (*sio.Socket, bool) {
	return slices.GetAny[*sio.Socket](clients, 0)
}

func (s *Server) onJoinRoom(sock *sio.Socket) func(...any) {
	return func(args ...any) {
		// Each payload argument of the "join-room" event is a room id; the
		// client may emit several at once (socket.emit("join-room", ...chatIds)).
		rooms := make([]sio.Room, 0, len(args))
		for i := range args {
			if room, ok := slices.GetAny[string](args, i); ok {
				rooms = append(rooms, chatRoom(room))
			}
		}
		sock.Join(rooms...)
	}
}

func (s *Server) onSendMessage(srv service.MessageService, sock *sio.Socket) func(...any) {
	return func(args ...any) { s.handleSendMessage(srv, sock, args) }
}

func (s *Server) handleSendMessage(srv service.MessageService, sock *sio.Socket, args []any) {
	if len(args) == 0 {
		return
	}

	ack := ackFrom(args)
	req, err := decodeSendMessage(args[0])
	if err != nil {
		ackFail(ack, "invalid payload")
		return
	}

	params := req.Msg
	params.ChatID = req.Room
	params.SenderID, _ = sock.Data().(uuid.UUID)

	ctx, cancel := context.WithTimeout(context.Background(), natsTimeout)
	defer cancel()

	saved, err := srv.SaveMessage(ctx, params)
	if err != nil {
		ackFail(ack, "save failed")
		return
	}

	ackOK(ack, saved)
	s.publishMessage(saved, sock.Id())
}

func decodeSendMessage(v any) (*sendMessageRequest, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var req sendMessageRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func ackFrom(args []any) sio.Ack {
	ack, _ := slices.GetAny[sio.Ack](args, len(args)-1)
	return ack
}

func ackOK(ack sio.Ack, msg dto.Message) {
	if ack != nil {
		ack([]any{msg}, nil)
	}
}

func ackFail(ack sio.Ack, reason string) {
	if ack != nil {
		ack([]any{map[string]any{"error": reason}}, nil)
	}
}

// publishMessage broadcasts a saved message to every hub via NATS.
func (s *Server) publishMessage(msg dto.Message, senderSid sio.SocketId) {
	data, err := json.Marshal(natsMessage{
		Event:     ReceiveMessage,
		Room:      msg.ChatID.String(),
		Payload:   msg,
		SenderID:  msg.SenderID.String(),
		SenderSid: string(senderSid),
	})
	if err != nil {
		return
	}
	s.nc.Publish(roomSubjectPrefix+msg.ChatID.String(), data)
}

func (s *Server) deliverFromNats(m *nats.Msg) {
	var msg natsMessage
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		return
	}

	room := strings.TrimPrefix(m.Subject, roomSubjectPrefix)
	if room != msg.Room {
		return
	}

	// Skip only the socket that sent the message (device A already has its own
	// copy from the ack). The sender's other devices get the broadcast like
	// everyone else.
	nsp := s.io.Sockets()
	nsp.
		To(chatRoom(room)).
		Except(sio.Room(msg.SenderSid)).
		Emit(string(ReceiveMessage), msg.Payload)
}

func natsUrl() string {
	if url := os.Getenv("NATS_URL"); url != "" {
		return url
	}
	return "nats://localhost:4222"
}
