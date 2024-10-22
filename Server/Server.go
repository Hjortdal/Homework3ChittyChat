package main

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"sync"

	pb "homework3/Proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	pb.UnimplementedChittyChatServer
	participants map[string]chan *pb.BroadcastMessage
	lamportTime  int64
	mu           sync.Mutex
}

func (s *server) Join(ctx context.Context, req *pb.JoinRequest) (*pb.JoinResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lamportTime++
	log.Printf("Join request from %s at Lamport time %d", req.ParticipantId, s.lamportTime)
	s.participants[req.ParticipantId] = make(chan *pb.BroadcastMessage, 100) // Buffered channel to avoid blocking
	message := &pb.BroadcastMessage{
		Message:     "Participant " + req.ParticipantId + " joined Chitty-Chat",
		LamportTime: s.lamportTime,
	}
	s.broadcast(message)
	return &pb.JoinResponse{Message: message.Message, LamportTime: s.lamportTime}, nil
}

func (s *server) Leave(ctx context.Context, req *pb.LeaveRequest) (*pb.LeaveResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lamportTime++
	log.Printf("Leave request from %s at Lamport time %d", req.ParticipantId, s.lamportTime)
	delete(s.participants, req.ParticipantId)
	message := &pb.BroadcastMessage{
		Message:     "Participant " + req.ParticipantId + " left Chitty-Chat",
		LamportTime: s.lamportTime,
	}
	s.broadcast(message)
	return &pb.LeaveResponse{Message: message.Message, LamportTime: s.lamportTime}, nil
}

func (s *server) Publish(ctx context.Context, req *pb.PublishRequest) (*pb.PublishResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lamportTime++
	log.Printf("Publish request from %s at Lamport time %d", req.ParticipantId, s.lamportTime)
	message := &pb.BroadcastMessage{
		Message:     req.ParticipantId + ": " + req.Message,
		LamportTime: s.lamportTime,
	}
	s.broadcast(message)
	return &pb.PublishResponse{Message: message.Message, LamportTime: s.lamportTime}, nil
}

func (s *server) Subscribe(req *pb.SubscribeRequest, stream pb.ChittyChat_SubscribeServer) error {
	participantId := req.ParticipantId
	log.Printf("Subscribe request from %s at Lamport time %d", participantId, s.lamportTime)
	if _, ok := s.participants[participantId]; !ok {
		return status.Errorf(codes.NotFound, "Participant %s not found", participantId)
	}
	for message := range s.participants[participantId] {
		s.mu.Lock()
		s.lamportTime++
		log.Printf("Sending message to %s: %s at Lamport time %d", participantId, message.Message, s.lamportTime)
		if err := stream.Send(message); err != nil {
			log.Printf("Error sending message to %s: %v at Lamport time %d", participantId, err, s.lamportTime)
			s.mu.Unlock()
			return err
		}
		s.mu.Unlock()
	}
	return nil
}

func (s *server) broadcast(message *pb.BroadcastMessage) {
	s.lamportTime++
	log.Printf("Broadcasting message: %s at Lamport time %d", message.Message, s.lamportTime)
	for participantId, ch := range s.participants {
		select {
		case ch <- message:
			log.Printf("Message sent to %s at Lamport time %d", participantId, s.lamportTime)
		default:
			log.Printf("Message to %s dropped (channel full) at Lamport time %d", participantId, s.lamportTime)
		}
	}
}

func main() {
	// Open a file for logging
	logFile, err := os.OpenFile("server.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}
	defer logFile.Close()

	// Create a multi-writer that writes to both the file and the terminal
	multiWriter := io.MultiWriter(os.Stderr, logFile)
	log.SetOutput(multiWriter)

	// Listen on all network interfaces
	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterChittyChatServer(s, &server{
		participants: make(map[string]chan *pb.BroadcastMessage),
	})
	log.Printf("Server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
