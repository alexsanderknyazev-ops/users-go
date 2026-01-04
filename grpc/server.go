package grpc

import (
	"context"
	"log"
	"net"

	pb "users/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// UsersServer реализует UsersService
type UsersServer struct {
	pb.UnimplementedUsersServiceServer
}

// NewUsersServer создает новый сервер
func NewUsersServer() *UsersServer {
	return &UsersServer{}
}

// GetUserByEmail - получение пользователя по email
func (s *UsersServer) GetUserByEmail(ctx context.Context, req *pb.GetUserByEmailRequest) (*pb.GetUserByEmailResponse, error) {
	email := req.GetEmail()
	log.Printf("GRPC GetUserByEmail called for email: %s", email)

	// TODO: Здесь подключите вашу реальную логику из service
	// user, err := userService.GetByEmail(ctx, email)
	// if err != nil {
	//     return nil, err
	// }

	// Заглушка - возвращаем тестового пользователя
	user := &pb.User{
		Id:        1,
		Email:     email,
		NickName:  "TestUser",
		Phone:     "+1234567890",
		Rate:      100,
		CreatedAt: timestamppb.Now(),
		DeletedAt: nil,
	}

	return &pb.GetUserByEmailResponse{
		User: user,
	}, nil
}

// ValidateUser - проверка существования пользователя
func (s *UsersServer) ValidateUser(ctx context.Context, req *pb.ValidateUserRequest) (*pb.ValidateUserResponse, error) {
	email := req.GetEmail()
	log.Printf("GRPC ValidateUser called for email: %s", email)

	// TODO: Здесь подключите вашу реальную логику
	// exists := userService.ExistsByEmail(ctx, email)

	return &pb.ValidateUserResponse{
		Valid: true, // Заглушка
	}, nil
}

// StartServer запускает GRPC сервер
func StartServer(port string) error {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	s := grpc.NewServer()
	server := NewUsersServer()
	pb.RegisterUsersServiceServer(s, server)

	log.Printf("Users GRPC Server listening on :%s", port)
	return s.Serve(lis)
}