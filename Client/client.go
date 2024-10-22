package main

import (
	"bufio"
	"context"
	"log"
	"os"

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

	log.Println("Attempting to join the chat...")
	joinResponse, err := c.Join(context.Background(), &pb.JoinRequest{ParticipantId: participantId})
	if err != nil {
		log.Fatalf("could not join: %v", err)
	}
	log.Printf("Join Response: %s at Lamport time %d", joinResponse.Message, joinResponse.LamportTime)

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
			log.Printf("Broadcast: %s at Lamport time %d", message.Message, message.LamportTime)
		}
	}()

	reader := bufio.NewReader(os.Stdin)
	for {
		log.Print("Enter message: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("failed to read input: %v", err)
		}

		log.Println("Publishing message...")
		publishResponse, err := c.Publish(context.Background(), &pb.PublishRequest{
			ParticipantId: participantId,
			Message:       input,
		})
		if err != nil {
			log.Fatalf("could not publish: %v", err)
		}
		log.Printf("Publish Response: %s at Lamport time %d", publishResponse.Message, publishResponse.LamportTime)
	}
}
