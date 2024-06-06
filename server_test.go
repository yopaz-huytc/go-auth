package main

import (
	"context"
	"testing"

	pb "github.com/yopaz-huytc/go-auth/protos/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLogin(t *testing.T) {
	s := Server{}
	ctx := context.Background()

	// Test case: User UID is empty
	req := &pb.LoginRequest{}
	resp, err := s.Login(ctx, req)
	if resp != nil || status.Code(err) != codes.InvalidArgument {
		t.Errorf("expected error with code InvalidArgument, got %v, %v", resp, err)
	}

	// TODO: Add more test cases for other scenarios, such as valid user UID, invalid user UID, etc.
	// You might need to mock the `fetchUserData` function and the Redis client to isolate the Login function for unit testing.
}
