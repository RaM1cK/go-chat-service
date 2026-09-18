package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"spotify-chat/internal/service"

	"github.com/RaM1cK/protos/pb"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type chatServer struct {
	chatService    service.ChatService
	messageService service.MessageService
	pb.UnimplementedChatServiceServer
}

func NewChatServer(chatService service.ChatService, messageService service.MessageService) *chatServer {
	return &chatServer{
		chatService:    chatService,
		messageService: messageService,
	}
}

func pbUUID(id *uuid.UUID) *pb.UUID {
	if id == nil {
		return nil
	}
	return &pb.UUID{Value: id.String()}
}

func pbURL(u *string) *pb.URL {
	if u == nil {
		return nil
	}
	return &pb.URL{Value: *u}
}

func checkError(err error, op string) error {
	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "request timed out")
	default:
		return status.Error(codes.Internal, fmt.Sprintf("%s failed", op))
	}
}

func parseUUID(v, what string) (uuid.UUID, error) {
	id, err := uuid.Parse(v)
	if err != nil {
		return uuid.Nil, status.Errorf(codes.InvalidArgument, "invalid %s", what)
	}

	return id, nil
}

func (s *chatServer) GetChats(req *pb.GetChatsRequest, stream pb.ChatService_GetChatsServer) error {
	userID, err := parseUUID(req.UserId.Value, "user id")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(stream.Context(), 5*time.Second)
	defer cancel()

	chats, err := s.chatService.GetChatsByUserId(ctx, userID)
	if err != nil {
		return checkError(err, "get chats")
	}

	chatIDs := make([]uuid.UUID, len(chats))
	for i, chat := range chats {
		chatIDs[i] = chat.ID
	}

	lastMessages, err := s.messageService.GetLastMessagesByChatIDs(ctx, chatIDs)
	if err != nil {
		return checkError(err, "get last messages")
	}

	for _, chat := range chats {
		msg := &pb.ChatResponse{
			Id:   pbUUID(&chat.ID),
			Type: int32(chat.Type),
			Logo: pbURL(chat.Logo),
		}

		if chat.Name != nil {
			msg.Name = *chat.Name
		}

		if lm, ok := lastMessages[chat.ID]; ok {
			msg.LastMessage = &pb.MessagePreview{
				Data: lm.Data,
				DataType: int32(lm.DataType),
			}
		}

		if err := stream.Send(msg); err != nil {
			return err
		}
	}

	return nil
}

func (s *chatServer) GetMessages(req *pb.GetMessagesRequest, stream pb.ChatService_GetMessagesServer) error {
	chatID, err := parseUUID(req.ChatId.Value, "chat id")
	if err != nil {
		return err
	}

	var timeOffset *time.Time
	if t := req.TimeOffset; t != nil {
		tempTime := t.AsTime()
		timeOffset = &tempTime
	}

	ctx, cancel := context.WithTimeout(stream.Context(), 5*time.Second)
	defer cancel()

	messages, err := s.messageService.GetMessages(ctx, chatID, req.Limit, timeOffset)
	if err != nil {
		return checkError(err, "get messages")
	}

	for _, message := range messages {
		msg := &pb.MessageResponse{
			ChatId:    req.ChatId,
			CreatedAt: timestamppb.New(message.CreatedAt),
			Id:        pbUUID(&message.ID),
			SenderId:  pbUUID(&message.SenderID),
			Data:      message.Data,
			DataType:  int32(message.DataType),
			QuotedId:  pbUUID(message.QuotedID),
		}

		if err := stream.Send(msg); err != nil {
			return err
		}
	}

	return nil
}

func (s *chatServer) Run(port string) error {
	if port == "" {
		port = "50051"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	server := grpc.NewServer()
	pb.RegisterChatServiceServer(server, s)

	if err := server.Serve(lis); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}
