package grpc

import (
	"context"
	"testing"

	pb "users/proto"
)

func TestNewUsersServer(t *testing.T) {
	if NewUsersServer() == nil {
		t.Fatal("NewUsersServer nil")
	}
}

func TestUsersServer_GetUserByEmail(t *testing.T) {
	s := NewUsersServer()
	resp, err := s.GetUserByEmail(context.Background(), &pb.GetUserByEmailRequest{Email: "grpc@test"})
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil || resp.User == nil || resp.User.Email != "grpc@test" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestUsersServer_ValidateUser(t *testing.T) {
	s := NewUsersServer()
	resp, err := s.ValidateUser(context.Background(), &pb.ValidateUserRequest{Email: "any@test"})
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil || !resp.Valid {
		t.Fatalf("ValidateUser: %+v", resp)
	}
}
