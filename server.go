package main

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	pb "github.com/yopaz-huytc/go-auth/protos/auth"
	"github.com/yopaz-huytc/go-auth/src/config"
	"github.com/yopaz-huytc/go-auth/src/controllers"
	"github.com/yopaz-huytc/go-auth/src/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
	"net"
)

var db = config.ConnectDB()

type Server struct {
	pb.UnimplementedAuthServiceServer
}

func (s *Server) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginResponse, error) {
	userUid := in.GetUid()
	if userUid == "" {
		return nil, status.Errorf(codes.InvalidArgument, "UID is required")
	}
	client, err := config.ConnectRedis()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error connecting to Redis: %v", err)
	}
	userData, err := fetchUserData(ctx, client, userUid)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error fetching user data: %v", err)
	}
	tokenString, err := controllers.CreateToken(userUid)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error creating token: %v", err)
	}
	return &pb.LoginResponse{
		Token: tokenString,
		Uid:   userUid,
		Email: userData["email"],
		Name:  userData["name"],
	}, nil
}

func (s *Server) RefreshToken(ctx context.Context, in *pb.RefreshTokenRequest) (*pb.RefreshTokenResponse, error) {
	return &pb.RefreshTokenResponse{
		Token: "TestRefreshToken",
	}, nil
}

func main() {
	defer config.DisconnectDB(db)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterAuthServiceServer(s, &Server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func fetchUserData(ctx context.Context, client *redis.Client, userUid string) (map[string]string, error) {
	userData := client.HGetAll(ctx, fmt.Sprintf("geomark-user:%s", userUid)).Val()
	if len(userData) == 0 {
		var db = config.ConnectDB()
		defer config.DisconnectDB(db)
		user, err := models.GetUserByUID(db, userUid)
		if err != nil {
			return nil, fmt.Errorf("error fetching user data from the database: %w", err)
		}
		userData = map[string]string{
			"id":          fmt.Sprint(user.ID),
			"name":        user.Name,
			"email":       user.Email,
			"description": user.Description,
			"uid":         user.UID,
			"parent_id":   fmt.Sprint(user.ParentId),
		}
		for k, v := range userData {
			err := client.HSet(ctx, fmt.Sprintf("geomark-user:%s", userUid), k, v).Err()
			if err != nil {
				return nil, fmt.Errorf("error setting user data in Redis: %w", err)
			}
		}
	}
	return userData, nil
}
