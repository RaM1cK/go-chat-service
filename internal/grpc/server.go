package grpc

import (
	"context"
	"fmt"
	"net"
	"spotify-chat/internal/service"
	"time"

	"github.com/RaM1cK/protos/pb"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type chatServer struct {
	chatService service.ChatService
	pb.UnimplementedChatServiceServer
}

func NewChatServer(chatService service.ChatService) *chatServer {
	return &chatServer{
		chatService: chatService,
	}
}

func (s *chatServer) GetChats(req *pb.GetChatsRequest, stream pb.ChatService_GetChatsServer) error {
	userID, err := uuid.Parse(req.GetUserId().GetValue())
	if err != nil {
		return status.Error(codes.InvalidArgument, "invalid user id")
	}

	ctx, cancel := context.WithTimeout(stream.Context(), 5*time.Second)
	defer cancel()

	chats, err := s.chatService.GetChatsByUserId(ctx, userID)
	if err != nil {
		return err
	}
	cancel()

	for _, chat := range chats {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}

		msg := &pb.ChatResponse{
			Id:   &pb.UUID{Value: chat.ID.String()},
			Type: int32(chat.Type),
		}

		if chat.Name != nil {
			msg.Name = *chat.Name
		}
		
		if chat.Logo != nil {
			msg.Logo = &pb.URL{Value: *chat.Logo}
		}

		if err := stream.Send(msg); err != nil {
			return err
		}
	}

	return nil
}

func (cs *chatServer) Run(port string) error {
	if port == "" {
		port = "50051"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	server := grpc.NewServer()
	pb.RegisterChatServiceServer(server, cs)

	if err := server.Serve(lis); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}