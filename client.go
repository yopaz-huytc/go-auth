package main

import (
	"context"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"
	"time"

	pb "github.com/yopaz-huytc/go-auth/protos/auth"
	"google.golang.org/grpc"
)

func main() {
	// Set up a connection to the server.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	dialer := func(ctx context.Context, addr string) (net.Conn, error) {
		return net.Dial("tcp", addr)
	}

	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock(), grpc.WithContextDialer(dialer))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewAuthServiceClient(conn)

	// Prepare login request
	req := &pb.LoginRequest{
		Uid: "4fb6637b-c316-4c88-b078-5b27ad952140", // replace with actual UID
	}

	// Contact the server and print out its response.
	r, err := c.Login(ctx, req)
	if err != nil {
		log.Fatalf("could not login: %v", err)
	}
	log.Printf("Response: %s", r.GetToken())
}
