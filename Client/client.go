package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	pb "homework3/Proto"

	"google.golang.org/grpc"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <participant_id>", os.Args[0])
	}

	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := pb.NewChittyChatClient(conn)
	participantId := os.Args[1]

	log.Println("Joining the chat...")
	joinResponse, err := c.Join(context.Background(), &pb.JoinRequest{ParticipantId: participantId})
	if err != nil {
		log.Fatalf("could not join: %v", err)
	}
	log.Printf("Joined chat: %s", joinResponse.Message)

	go func() {
		log.Println("Subscribing to chat messages...")
		stream, err := c.Subscribe(context.Background(), &pb.SubscribeRequest{ParticipantId: participantId})
		if err != nil {
			log.Fatalf("could not subscribe: %v", err)
		}
		for {
			message, err := stream.Recv()
			if err != nil {
				log.Fatalf("could not receive message: %v", err)
			}
			log.Printf("Message: %s", message.Message)
		}
	}()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Leaving the chat...")
		_, err := c.Leave(context.Background(), &pb.LeaveRequest{ParticipantId: participantId})
		if err != nil {
			log.Fatalf("could not leave: %v", err)
		}
		os.Exit(0)
	}()

	reader := bufio.NewReader(os.Stdin)
	for {
		log.Print("Enter message: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("failed to read input: %v", err)
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		log.Println("Publishing message...")
		publishResponse, err := c.Publish(context.Background(), &pb.PublishRequest{
			ParticipantId: participantId,
			Message:       input,
		})
		if err != nil {
			log.Fatalf("could not publish: %v", err)
		}
		log.Printf("Message published: %s", publishResponse.Message)
	}
}
